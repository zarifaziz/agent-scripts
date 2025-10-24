package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/peterh/liner"
)

type replConfig struct {
	SessionName  string
	Database     string
	OutputFormat string
	ReadOnly     bool
	Params       map[string]any
}

var supportedFormats = map[string]struct{}{
	"json":    {},
	"compact": {},
	"csv":     {},
	"table":   {},
}

type replCommandAction struct {
	handled       bool
	exit          bool
	executeBuffer bool
	clearBuffer   bool
}

func getHistoryPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	cacheDir := filepath.Join(home, ".cache", "psql-safe")
	if err := os.MkdirAll(cacheDir, 0o755); err != nil {
		return "", err
	}
	return filepath.Join(cacheDir, "history"), nil
}

func startRepl(ctx context.Context, pool *pgxpool.Pool, cfg replConfig) error {
	line := liner.NewLiner()
	line.SetCtrlCAborts(true)

	// Load history
	historyPath, err := getHistoryPath()
	if err == nil {
		if f, err := os.Open(historyPath); err == nil {
			_, _ = line.ReadHistory(f)
			f.Close()
		}
	}

	defer func() {
		// Save history
		if historyPath != "" {
			if f, err := os.Create(historyPath); err == nil {
				_, _ = line.WriteHistory(f)
				f.Close()
			}
		}
		_ = line.Close()
	}()

	if cfg.OutputFormat == "" {
		cfg.OutputFormat = "table"
	}
	if _, ok := supportedFormats[cfg.OutputFormat]; !ok {
		return fmt.Errorf("unsupported output format: %s", cfg.OutputFormat)
	}

	printReplBanner(cfg)

	var buffer []string

	if cfg.Params == nil {
		cfg.Params = map[string]any{}
	}

	for {
		prompt := replPrompt(cfg.SessionName, len(buffer) > 0)
		input, err := line.Prompt(prompt)
		if err != nil {
			if errors.Is(err, liner.ErrPromptAborted) {
				fmt.Println("^C")
				buffer = nil
				continue
			}
			if errors.Is(err, io.EOF) {
				fmt.Println()
				return nil
			}
			return err
		}

		trimmed := strings.TrimSpace(input)

		if strings.HasPrefix(trimmed, ":") {
			action := handleReplCommand(ctx, pool, trimmed, &cfg)
			if action.handled {
				if action.exit {
					return nil
				}
				if action.clearBuffer {
					buffer = nil
					fmt.Println("Buffer cleared")
				}
				if action.executeBuffer {
					if len(buffer) == 0 {
						fmt.Println("Nothing to execute (query buffer is empty)")
						continue
					}
					query := strings.Join(buffer, "\n")
					buffer = nil
					query = strings.TrimSpace(query)
					query = strings.TrimSuffix(query, ";")
					query = strings.TrimSpace(query)
					if query == "" {
						continue
					}
					if err := runReplQuery(ctx, pool, line, query, &cfg); err != nil {
						fmt.Fprintf(os.Stderr, "Error: %v\n", err)
					}
				}
				continue
			}
		}

		if trimmed == "" && len(buffer) == 0 {
			continue
		}

		buffer = append(buffer, input)

		if shouldExecuteBuffer(buffer) {
			query := strings.Join(buffer, "\n")
			buffer = nil
			query = strings.TrimSpace(query)
			query = strings.TrimSuffix(query, ";")
			query = strings.TrimSpace(query)
			if query == "" {
				continue
			}
			if err := runReplQuery(ctx, pool, line, query, &cfg); err != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			}
		}
	}
}

func runReplQuery(ctx context.Context, pool *pgxpool.Pool, line *liner.State, query string, cfg *replConfig) error {
	historyEntry := strings.ReplaceAll(strings.TrimSpace(query), "\n", " ")
	if historyEntry != "" {
		line.AppendHistory(historyEntry)
	}

	if cfg.SessionName != "" {
		return executeInSession(ctx, pool, cfg.SessionName, cfg.Database, query, cfg.Params, cfg.OutputFormat, false)
	}
	return executeReadOnly(ctx, pool, query, cfg.Params, cfg.OutputFormat)
}

func shouldExecuteBuffer(lines []string) bool {
	for i := len(lines) - 1; i >= 0; i-- {
		trimmed := strings.TrimSpace(lines[i])
		if trimmed == "" {
			continue
		}
		return strings.HasSuffix(trimmed, ";")
	}
	return false
}

func handleReplCommand(ctx context.Context, pool *pgxpool.Pool, raw string, cfg *replConfig) replCommandAction {
	parts := strings.Fields(raw)
	if len(parts) == 0 {
		return replCommandAction{handled: true}
	}

	switch parts[0] {
	case ":quit", ":exit":
		return replCommandAction{handled: true, exit: true}
	case ":help":
		printReplHelp(cfg.SessionName != "")
	case ":format":
		if len(parts) != 2 {
			fmt.Println("Usage: :format <json|compact|csv|table>")
			return replCommandAction{handled: true}
		}
		format := strings.ToLower(parts[1])
		if _, ok := supportedFormats[format]; !ok {
			fmt.Printf("Unknown format: %s\n", format)
			return replCommandAction{handled: true}
		}
		cfg.OutputFormat = format
		fmt.Printf("Output format set to %s\n", format)
	case ":commit":
		if cfg.SessionName == "" {
			fmt.Println("Commit is only available in session mode.")
			return replCommandAction{handled: true}
		}
		if err := commitSession(ctx, pool, cfg.SessionName, cfg.Database); err != nil {
			fmt.Fprintf(os.Stderr, "Commit failed: %v\n", err)
		} else {
			fmt.Println("Session committed. Exiting REPL.")
			return replCommandAction{handled: true, exit: true}
		}
	case ":sessions":
		sessions, err := getSessions()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to list sessions: %v\n", err)
			return replCommandAction{handled: true}
		}
		if len(sessions) == 0 {
			fmt.Println("No active sessions")
			return replCommandAction{handled: true}
		}
		for _, s := range sessions {
			fmt.Printf("%s (db: %s, queries: %d, created: %s)\n", s.Name, s.Database, len(s.Queries), s.CreatedAt.Format(time.RFC3339))
		}
	case ":show":
		if len(parts) != 2 {
			fmt.Println("Usage: :show <session>")
			return replCommandAction{handled: true}
		}
		if err := printSessionDetails(parts[1]); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to show session: %v\n", err)
		}
	case ":params":
		arg := strings.TrimSpace(strings.TrimPrefix(raw, parts[0]))
		switch {
		case arg == "":
			printParams(cfg.Params)
		case strings.EqualFold(arg, "clear"), strings.EqualFold(arg, "reset"):
			cfg.Params = map[string]any{}
			fmt.Println("Parameters cleared")
		default:
			var parsed map[string]any
			if err := json.Unmarshal([]byte(arg), &parsed); err != nil {
				fmt.Fprintf(os.Stderr, "Failed to parse parameters (expect JSON object): %v\n", err)
				return replCommandAction{handled: true}
			}
			cfg.Params = parsed
			fmt.Println("Parameters updated")
		}
	case ":history":
		if cfg.SessionName == "" {
			fmt.Println("History is only available in session mode.")
			return replCommandAction{handled: true}
		}
		if err := printSessionDetails(cfg.SessionName); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to show session history: %v\n", err)
		}
	case ":exec", ":run":
		return replCommandAction{handled: true, executeBuffer: true}
	case ":clear":
		return replCommandAction{handled: true, clearBuffer: true}
	default:
		fmt.Printf("Unknown command: %s (type :help for available commands)\n", parts[0])
	}
	return replCommandAction{handled: true}
}

func printReplBanner(cfg replConfig) {
	if cfg.SessionName != "" {
		fmt.Printf("psql-safe REPL (session: %s, database: %s)\n", cfg.SessionName, cfg.Database)
		fmt.Println(" - All queries replay from stored session history")
		fmt.Println(" - Changes are automatically rolled back unless you :commit")
		fmt.Println(" - Use :commit to apply all queries and exit")
	} else {
		fmt.Printf("psql-safe REPL (database: %s)\n", cfg.Database)
		if cfg.ReadOnly {
			fmt.Println(" - Running in read-only mode")
			fmt.Println(" - Write queries are blocked")
		} else {
			fmt.Println(" - Running with write access")
		}
	}
	fmt.Println("Commands: :help, :format, :params, :show, :sessions, :history, :exec, :clear, :commit (session only), :quit")
	fmt.Println("Enter queries across multiple lines; finish with ';' to execute")
	fmt.Println()
}

func printReplHelp(sessionMode bool) {
	fmt.Println("Commands:")
	fmt.Println("  :help         Show this help message")
	fmt.Println("  :format <f>   Set output format (json, compact, csv, table)")
	if sessionMode {
		fmt.Println("  :commit       Apply all recorded queries, delete session, and exit")
		fmt.Println("  :history      Show all queries stored in current session")
	}
	fmt.Println("  :params [JSON|clear]  View or set query parameters (JSON object)")
	fmt.Println("  :show <name>          Display stored queries for a session")
	fmt.Println("  :sessions             List available sessions")
	fmt.Println("  :exec | :run  Execute the current query buffer")
	fmt.Println("  :clear        Discard the current query buffer")
	fmt.Println("  :quit         Exit the REPL")
	fmt.Println()
	fmt.Println("Hints:")
	fmt.Println("  - Press Enter to keep writing; queries execute when the final line ends with ';'")
	fmt.Println("  - Use :exec to run the current buffer without adding ';'")
	fmt.Println("  - Results default to table format for readability")
	if sessionMode {
		fmt.Println("  - Session mode replays all queries on each run; :commit persists them")
	}
}

func replPrompt(sessionName string, inMultiline bool) string {
	label := "psql"
	if sessionName != "" {
		label = fmt.Sprintf("psql(%s)", sessionName)
	}
	if inMultiline {
		return "... "
	}
	return label + "> "
}

func printParams(params map[string]any) {
	if len(params) == 0 {
		fmt.Println("No parameters set")
		return
	}
	data, err := json.MarshalIndent(params, "", "  ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to marshal parameters: %v\n", err)
		return
	}
	fmt.Println(string(data))
}

func printSessionDetails(sessionName string) error {
	sessionPath, err := getSessionPath(sessionName)
	if err != nil {
		return err
	}
	session, err := loadSession(sessionPath)
	if err != nil {
		return err
	}

	fmt.Printf("Session: %s\n", session.Name)
	fmt.Printf("Database: %s\n", session.Database)
	fmt.Printf("Created: %s\n", session.CreatedAt.Format(time.RFC3339))
	fmt.Printf("Queries (%d):\n", len(session.Queries))
	for i, q := range session.Queries {
		fmt.Printf("\n[%d]\n%s\n", i+1, q.SQL)
		if len(q.Params) > 0 {
			paramsJSON, _ := json.MarshalIndent(q.Params, "  ", "  ")
			fmt.Printf("  Params: %s\n", string(paramsJSON))
		}
	}
	if len(session.Queries) == 0 {
		fmt.Println("  (no queries recorded)")
	}
	return nil
}
