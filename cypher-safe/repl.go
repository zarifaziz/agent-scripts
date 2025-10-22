package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
	"github.com/peterh/liner"
	"github.com/rlch/neogo"
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

func startRepl(ctx context.Context, client neogo.Driver, cfg replConfig) error {
	line := liner.NewLiner()
	line.SetCtrlCAborts(true)
	defer func() {
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
			action := handleReplCommand(ctx, client, trimmed, &cfg)
			if action.handled {
				if action.exit {
					return nil
				}
				if action.clearBuffer {
					buffer = nil
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
					if err := runReplQuery(ctx, client, line, query, &cfg); err != nil {
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
			if err := runReplQuery(ctx, client, line, query, &cfg); err != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			}
		}
	}
}

func runReplQuery(ctx context.Context, client neogo.Driver, line *liner.State, query string, cfg *replConfig) error {
	historyEntry := strings.ReplaceAll(strings.TrimSpace(query), "\n", " ")
	if historyEntry != "" {
		line.AppendHistory(historyEntry)
	}
	return executeReplQuery(ctx, client, query, *cfg)
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

func handleReplCommand(ctx context.Context, client neogo.Driver, raw string, cfg *replConfig) replCommandAction {
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
		if err := commitSession(ctx, client, cfg.SessionName, cfg.Database); err != nil {
			fmt.Fprintf(os.Stderr, "Commit failed: %v\n", err)
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
	case ":exec", ":run":
		return replCommandAction{handled: true, executeBuffer: true}
	case ":clear":
		return replCommandAction{handled: true, clearBuffer: true}
	default:
		fmt.Printf("Unknown command: %s\n", parts[0])
	}
	return replCommandAction{handled: true}
}

func executeReplQuery(ctx context.Context, client neogo.Driver, query string, cfg replConfig) error {
	if cfg.SessionName != "" {
		return executeRawQueryInSession(ctx, client, cfg.SessionName, cfg.Database, query, cfg.Params, cfg.OutputFormat)
	}
	return executeRawQuery(ctx, client, query, cfg.Params, cfg.OutputFormat, cfg.ReadOnly)
}

func printReplBanner(cfg replConfig) {
	if cfg.SessionName != "" {
		fmt.Printf("cypher-query REPL (session: %s, database: %s)\n", cfg.SessionName, cfg.Database)
		fmt.Println(" - All queries replay from stored session history")
		fmt.Println(" - Use :commit to apply the session and delete it")
	} else {
		fmt.Printf("cypher-query REPL (database: %s)\n", cfg.Database)
		if cfg.ReadOnly {
			fmt.Println(" - Running in read-only mode")
		} else {
			fmt.Println(" - Running with write access")
		}
	}
	fmt.Println("Commands: :help, :format, :params, :show, :sessions, :exec, :clear, :commit (session only), :quit")
	fmt.Println("Enter queries across multiple lines; finish with ';' to execute")
}

func printReplHelp(sessionMode bool) {
	fmt.Println("Commands:")
	fmt.Println("  :help         Show this help message")
	fmt.Println("  :format <f>   Set output format (json, compact, csv, table)")
	if sessionMode {
		fmt.Println("  :commit       Apply all recorded queries and delete the session")
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
	fmt.Println("  - Session mode replays all queries on each run; :commit persists them")
}

func replPrompt(sessionName string, inMultiline bool) string {
	label := "cypher"
	if sessionName != "" {
		label = fmt.Sprintf("cypher(%s)", sessionName)
	}
	if inMultiline {
		return "... "
	}
	return label + "> "
}

func commitSession(ctx context.Context, client neogo.Driver, sessionName, database string) error {
	sessionPath, err := getSessionPath(sessionName)
	if err != nil {
		return err
	}
	sessionData, err := loadSession(sessionPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("session '%s' not found", sessionName)
		}
		return err
	}
	if sessionData.Database != "" && sessionData.Database != database {
		return fmt.Errorf("session configured for database %s (requested %s)", sessionData.Database, database)
	}
	if len(sessionData.Queries) == 0 {
		return fmt.Errorf("session '%s' has no queries to commit", sessionName)
	}

	rawDriver := client.DB()
	session := rawDriver.NewSession(ctx, neo4j.SessionConfig{
		DatabaseName: database,
		AccessMode:   neo4j.AccessModeWrite,
	})
	defer func() {
		_ = session.Close(ctx)
	}()

	tx, err := session.BeginTransaction(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}

	for idx, qp := range sessionData.Queries {
		if _, err := tx.Run(ctx, qp.Query, qp.Params); err != nil {
			_ = tx.Rollback(ctx)
			return fmt.Errorf("commit aborted at query %d: %w", idx+1, err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit session: %w", err)
	}

	if err := deleteSession(sessionName); err != nil {
		return fmt.Errorf("session committed but cleanup failed: %w", err)
	}

	fmt.Printf("✓ Session '%s' committed (%d queries applied)\n", sessionName, len(sessionData.Queries))
	return nil
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
		fmt.Printf("\n[%d]\n%s\n", i+1, q)
	}
	if len(session.Queries) == 0 {
		fmt.Println("  (no queries recorded)")
	}
	return nil
}
