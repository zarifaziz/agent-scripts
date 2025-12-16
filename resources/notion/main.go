package main

import (
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/joho/godotenv"
)

//go:embed USAGE.md
var usageMD string

func main() {
	_ = godotenv.Load()

	if len(os.Args) < 2 {
		fmt.Println("notion <query|get|content|schema|users> | echo '{\"filter\":{...}}' | notion query")
		fmt.Println("notion --help for full docs")
		os.Exit(1)
	}

	cmd := os.Args[1]
	args := os.Args[2:]

	// Commands that don't need a token
	switch cmd {
	case "help", "-h", "--help":
		fmt.Println(usageMD)
		return
	case "users", "u":
		runUsers()
		return
	}

	// Commands that need a token
	token := getToken()
	if token == "" {
		fmt.Fprintln(os.Stderr, "Error: NOTION_TOKEN not set")
		os.Exit(1)
	}

	ctx := context.Background()
	client := NewClient(token)

	switch cmd {
	case "query", "q":
		runQuery(ctx, client, args)
	case "get", "g":
		runGet(ctx, client, args)
	case "content", "c":
		runContent(ctx, client, args)
	case "schema", "s":
		runSchema(ctx, client, args)
	default:
		if isIssueID(cmd) {
			runGet(ctx, client, []string{cmd})
		} else {
			runQuery(ctx, client, []string{"--title", strings.Join(os.Args[1:], " ")})
		}
	}
}

// QueryInput is the JSON structure LLMs send via stdin
type QueryInput struct {
	Filter        json.RawMessage `json:"filter,omitempty"`
	Sorts         json.RawMessage `json:"sorts,omitempty"`
	DatabaseID    string          `json:"database_id,omitempty"`
	Limit         int             `json:"limit,omitempty"`
	PageSize      int             `json:"page_size,omitempty"`
	ResolvePeople *bool           `json:"resolve_people,omitempty"`
}

func runQuery(ctx context.Context, client *Client, args []string) {
	opts := QueryOptions{
		PageSize:      100,
		Limit:         100,
		ResolvePeople: true,
		Format:        "json",
	}

	// Check if stdin has data (primary interface for LLMs)
	// But not if we have CLI flags that look like shorthands
	hasShorthand := false
	for _, a := range args {
		if a == "--status" || a == "--assign" || a == "--qa" || a == "--priority" || a == "--title" {
			hasShorthand = true
			break
		}
	}

	stat, _ := os.Stdin.Stat()
	if !hasShorthand && (stat.Mode()&os.ModeCharDevice) == 0 {
		// Data piped via stdin - parse as JSON
		data, err := io.ReadAll(os.Stdin)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error reading stdin: %v\n", err)
			os.Exit(1)
		}

		var input QueryInput
		if err := json.Unmarshal(data, &input); err != nil {
			fmt.Fprintf(os.Stderr, "Invalid JSON input: %v\n", err)
			os.Exit(1)
		}

		if len(input.Filter) > 0 {
			opts.Filter = input.Filter
		}
		if len(input.Sorts) > 0 {
			opts.Sorts = input.Sorts
		}
		if input.DatabaseID != "" {
			opts.DatabaseID = input.DatabaseID
		}
		if input.Limit > 0 {
			opts.Limit = input.Limit
		}
		if input.PageSize > 0 {
			opts.PageSize = input.PageSize
		}
		if input.ResolvePeople != nil {
			opts.ResolvePeople = *input.ResolvePeople
		}

		// Parse remaining args for format/dry-run only
		for i := 0; i < len(args); i++ {
			switch args[i] {
			case "--format", "-o":
				if i+1 < len(args) {
					opts.Format = args[i+1]
					i++
				}
			case "--dry-run":
				opts.DryRun = true
			}
		}
	} else {
		// CLI args mode (convenience shorthands)
		var shorthandFilters []string

		for i := 0; i < len(args); i++ {
			switch args[i] {
			case "--db", "-d":
				if i+1 < len(args) {
					opts.DatabaseID = args[i+1]
					i++
				}
			case "--limit", "-l":
				if i+1 < len(args) {
					fmt.Sscanf(args[i+1], "%d", &opts.Limit)
					i++
				}
			case "--format", "-o":
				if i+1 < len(args) {
					opts.Format = args[i+1]
					i++
				}
			case "--no-resolve":
				opts.ResolvePeople = false
			case "--dry-run":
				opts.DryRun = true
			case "--status":
				if i+1 < len(args) {
					shorthandFilters = append(shorthandFilters, buildFilter("Status", "status", "equals", args[i+1]))
					i++
				}
			case "--assign":
				if i+1 < len(args) {
					shorthandFilters = append(shorthandFilters, buildFilter("Assign", "people", "contains", args[i+1]))
					i++
				}
			case "--qa":
				if i+1 < len(args) {
					shorthandFilters = append(shorthandFilters, buildFilter("QA Engineer", "people", "contains", args[i+1]))
					i++
				}
			case "--priority":
				if i+1 < len(args) {
					shorthandFilters = append(shorthandFilters, buildFilter("Priority", "select", "equals", args[i+1]))
					i++
				}
			case "--title":
				if i+1 < len(args) {
					shorthandFilters = append(shorthandFilters, buildFilter("Issue Title", "rich_text", "contains", args[i+1]))
					i++
				}
			default:
				if strings.HasPrefix(args[i], "{") {
					opts.Filter = json.RawMessage(args[i])
				}
			}
		}

		if len(opts.Filter) == 0 && len(shorthandFilters) > 0 {
			if len(shorthandFilters) == 1 {
				opts.Filter = json.RawMessage(shorthandFilters[0])
			} else {
				// Build compound filter safely via json.Marshal
				var filters []json.RawMessage
				for _, f := range shorthandFilters {
					filters = append(filters, json.RawMessage(f))
				}
				compound := map[string][]json.RawMessage{"and": filters}
				opts.Filter, _ = json.Marshal(compound)
			}
		}
	}

	result, err := client.Query(ctx, opts)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	switch opts.Format {
	case "json":
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		enc.Encode(result)
	case "ids":
		for _, p := range result.Pages {
			fmt.Println(p.IssueID)
		}
	case "urls":
		for _, p := range result.Pages {
			fmt.Println(p.URL)
		}
	case "titles":
		for _, p := range result.Pages {
			fmt.Printf("%s: %s\n", p.IssueID, p.Title)
		}
	case "table":
		printTable(result)
	default:
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		enc.Encode(result)
	}
}

func runGet(ctx context.Context, client *Client, args []string) {
	if len(args) < 1 {
		fmt.Fprintln(os.Stderr, "Usage: notion get <ISSUE-ID>")
		os.Exit(1)
	}

	issueID := args[0]
	page, err := client.GetPageByIssueID(ctx, issueID)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	format := "json"
	if len(args) > 1 && args[1] == "--format" && len(args) > 2 {
		format = args[2]
	}

	result := client.pageToResult(page)

	switch format {
	case "json":
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		enc.Encode(result)
	default:
		fmt.Printf("ID: %s\n", result.IssueID)
		fmt.Printf("Title: %s\n", result.Title)
		fmt.Printf("Status: %s\n", result.Status)
		fmt.Printf("Priority: %s\n", result.Priority)
		fmt.Printf("Assign: %s\n", strings.Join(result.Assign, ", "))
		fmt.Printf("QA: %s\n", strings.Join(result.QA, ", "))
		fmt.Printf("URL: %s\n", result.URL)
	}
}

func runContent(ctx context.Context, client *Client, args []string) {
	if len(args) < 1 {
		fmt.Fprintln(os.Stderr, "Usage: notion content <ISSUE-ID|page-id>")
		os.Exit(1)
	}

	pageID := args[0]

	if isIssueID(pageID) {
		page, err := client.GetPageByIssueID(ctx, pageID)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		pageID = string(page.ID)
	}

	blocks, err := client.GetPageContent(ctx, pageID)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	for _, block := range blocks {
		text := BlockToText(block)
		if text != "" {
			fmt.Println(text)
		}
	}
}

func runSchema(ctx context.Context, client *Client, args []string) {
	dbID := TechIssuesDBID
	refresh := false
	cacheFile := getCacheFile()

	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--db":
			if i+1 < len(args) {
				dbID = args[i+1]
				i++
			}
		case "--refresh", "-r":
			refresh = true
		case "--cache-file":
			if i+1 < len(args) {
				cacheFile = args[i+1]
				i++
			}
		}
	}

	// Try cache first unless refresh requested
	if !refresh {
		if data, err := os.ReadFile(cacheFile); err == nil {
			var schema DatabaseSchema
			if json.Unmarshal(data, &schema) == nil {
				enc := json.NewEncoder(os.Stdout)
				enc.SetIndent("", "  ")
				enc.Encode(schema)
				return
			}
		}
	}

	// Fetch fresh schema
	schema, err := client.GetSchema(ctx, dbID)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	// Cache it
	if data, err := json.MarshalIndent(schema, "", "  "); err == nil {
		os.MkdirAll(getCacheDir(), 0o755)
		os.WriteFile(cacheFile, data, 0o644)
	}

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	enc.Encode(schema)
}

func getCacheDir() string {
	if xdg := os.Getenv("XDG_CACHE_HOME"); xdg != "" {
		return xdg + "/notion"
	}
	home, _ := os.UserHomeDir()
	return home + "/.cache/notion"
}

func getCacheFile() string {
	return getCacheDir() + "/schema.json"
}

func runUsers() {
	users := make(map[string]UserInfo)

	// Build reverse map: ID -> names that map to it
	idToNames := make(map[string][]string)
	for name, id := range userIDMap {
		idToNames[id] = append(idToNames[id], name)
	}

	// Output as user-centric map
	for id, names := range idToNames {
		// Pick the shortest name as primary (usually first name)
		primary := names[0]
		for _, n := range names {
			if len(n) < len(primary) {
				primary = n
			}
		}
		users[primary] = UserInfo{
			ID:      id,
			Aliases: names,
		}
	}

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	enc.Encode(users)
}

type UserInfo struct {
	ID      string   `json:"id"`
	Aliases []string `json:"aliases"`
}

func printTable(result *QueryResult) {
	if len(result.Pages) == 0 {
		fmt.Println("No results found")
		return
	}

	fmt.Printf("Found %d results:\n", result.TotalCount)
	fmt.Println(strings.Repeat("=", 80))

	for _, p := range result.Pages {
		fmt.Printf("%s | %s | %s\n", p.IssueID, p.Status, p.Title)
		if len(p.Assign) > 0 || len(p.QA) > 0 {
			fmt.Printf("  Assign: %s | QA: %s\n", strings.Join(p.Assign, ", "), strings.Join(p.QA, ", "))
		}
		fmt.Printf("  %s\n", p.URL)
		fmt.Println()
	}
}

// buildFilter creates a JSON filter string safely (escapes user input)
func buildFilter(property, filterType, condition, value string) string {
	filter := map[string]any{
		"property": property,
		filterType: map[string]any{
			condition: value,
		},
	}
	b, _ := json.Marshal(filter)
	return string(b)
}

func getToken() string {
	if t := os.Getenv("NOTION_TOKEN"); t != "" {
		return t
	}
	if t := os.Getenv("NOTION_API_TOKEN"); t != "" {
		return t
	}
	return os.Getenv("CFG_NOTIONTOKEN")
}

func isIssueID(s string) bool {
	s = strings.ToUpper(strings.TrimSpace(s))
	return strings.HasPrefix(s, "ISSUE-") || strings.HasPrefix(s, "TASK-")
}
