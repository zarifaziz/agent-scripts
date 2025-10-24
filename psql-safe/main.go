package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"
	"time"
)

const helpText = `psql-safe - Execute SQL queries against PostgreSQL/CockroachDB safely

USAGE:
    psql-safe [OPTIONS] <QUERY>              # Read-only mode
    psql-safe --session <NAME> <QUERY>       # Session mode (safe write testing)
    psql-safe --session <NAME> --repl        # Session REPL with :commit support
    psql-safe --list                         # List active sessions
    psql-safe --show <SESSION_NAME>          # Display session queries
    psql-safe --drop <SESSION_NAME>          # Delete a session

IMPORTANT: All flags MUST come BEFORE the query string, not after.

DESCRIPTION:
    A command-line tool for executing SQL queries against PostgreSQL/CockroachDB.
    Results are printed as JSON array suitable for piping to jq.

    TWO MODES:

    1. DEFAULT MODE (Read-Only):
       Perfect for safe data exploration and reporting.
       - Executes queries in read-only transactions
       - Automatically rolls back after execution
       - Blocks write operations (INSERT, UPDATE, DELETE, etc.)

    2. SESSION MODE (Safe Write Testing):
       Use --session <name> to create a safe testing environment where you can:
       - Run multiple write queries (INSERT, UPDATE, DELETE, etc.)
       - Inspect intermediate results after each query
       - See how data evolves across multiple operations
       - Everything is AUTOMATICALLY ROLLED BACK (nothing persists to DB)
       - All queries replay when you access the same session
       - Perfect for testing complex write operations before committing

OPTIONS:
    --host <HOST>         Database host address (default: localhost)
    --port <PORT>         Database port (default: 26257 for CockroachDB)
    --user <USER>         Database username (default: root)
    --database <DATABASE> Database name (default: defaultdb)
    --password <PASS>     Database password (reads from env if not set)
    --ssl-mode <MODE>     SSL mode: disable, require, verify-full (default: require)
    --cert-dir <PATH>     Path to certificate directory for TLS
    --preset <NAME>       Use connection preset
    --session <NAME>      Execute query in named session (auto-creates if new)
    --params <JSON>       Query parameters as JSON object (e.g. '{"name":"Alice","age":30}')
    --format <FORMAT>     Output format: json, compact, csv, table (default: json)
    --file <PATH>         Read SQL from file (- for stdin)
    --timeout <DURATION>  Query timeout (default: 30s)
    --list                List all active sessions with metadata
    --show <NAME>         Display session queries without executing
    --drop <NAME>         Delete a session (all queries discarded)
    --commit              Commit session and delete it (use with --session)
    --cleanup-days <INT>  Clean sessions older than N days (default: 14)
    --repl                Start interactive REPL
    --help                Show this help message

ENVIRONMENT VARIABLES:
    COCKROACH_URL         Full connection URL (overrides individual flags)
    COCKROACH_PASSWORD    Database password
    PGPASSWORD            PostgreSQL password (fallback)
    PGSSLMODE             SSL mode (fallback for --ssl-mode)

EXAMPLES:

  Read-only queries (default mode):
    $ export COCKROACH_PASSWORD=mypassword
    $ psql-safe "SELECT * FROM customers LIMIT 5"
    $ psql-safe "SELECT count(*) as total FROM orders"
    $ psql-safe "SELECT email, balance FROM customers ORDER BY balance DESC"

  Session mode - safe write testing workflow:

    # Create first test record (auto-creates session "test")
    $ psql-safe --session test \
        "INSERT INTO customers (email, balance) VALUES ('alice@example.com', 125.50) RETURNING *"

    # Add second record - sees Alice from replayed INSERT
    $ psql-safe --session test \
        "INSERT INTO customers (email, balance) VALUES ('bob@example.com', 99.99) RETURNING *"

    # Query all people - sees BOTH Alice and Bob
    $ psql-safe --session test \
        "SELECT email, balance FROM customers ORDER BY balance"

    # Update records and inspect
    $ psql-safe --session test \
        "UPDATE customers SET balance = balance + 10 WHERE email = 'alice@example.com' RETURNING *"

    # CRITICAL: Verify database is UNCHANGED (all rolled back!)
    $ psql-safe "SELECT count(*) FROM customers WHERE email = 'alice@example.com'"
    [{"count": 0}]  # Nothing was committed!

    # Inspect session metadata
    $ psql-safe --list
    Session: test
      Database: defaultdb
      Created: 2025-10-24T10:30:00Z
      Queries: 4

    # When ready, commit the session
    $ psql-safe --session test --commit
    ✓ Session 'test' committed (4 queries applied)

    # Or drop session without committing
    $ psql-safe --drop test
    ✓ Session 'test' dropped

  Interactive REPL:
    $ psql-safe --repl
    $ psql-safe --session test --repl

  Pipe results to jq:
    $ psql-safe "SELECT email FROM customers" | jq -r '.[].email'

  Connect to remote server:
    $ psql-safe --host prod.example.com --port 26257 \
        --database mydb "SELECT count(*) as total FROM users"

  Using presets:
    $ psql-safe --preset dev "SELECT * FROM products LIMIT 10"

  Named parameters:
    $ psql-safe --params '{"min_balance": 100}' \
        "SELECT * FROM customers WHERE balance > :min_balance"

WHEN TO USE EACH MODE:
    - Use DEFAULT for: Reading data, generating reports, exploring tables
    - Use SESSION for: Testing writes, debugging mutations, building complex updates
`

type formatFlag struct {
	value string
	set   bool
}

func (f *formatFlag) String() string {
	return f.value
}

func (f *formatFlag) Set(v string) error {
	f.value = v
	f.set = true
	return nil
}

func main() {
	var host, port, username, database, password, sslMode, certDir string
	var showHelp, listSessions bool
	var sessionName, dropSession, showSession, paramsJSON, presetName, filePath string
	var cleanupDays, timeout int
	var replMode, commitFlag bool
	format := formatFlag{value: "json"}

	flag.StringVar(&host, "host", "localhost", "")
	flag.StringVar(&port, "port", "26257", "")
	flag.StringVar(&username, "user", "root", "")
	flag.StringVar(&database, "database", "defaultdb", "")
	flag.StringVar(&password, "password", "", "")
	flag.StringVar(&sslMode, "ssl-mode", "require", "")
	flag.StringVar(&certDir, "cert-dir", "", "")
	flag.StringVar(&presetName, "preset", "", "")
	flag.StringVar(&sessionName, "session", "", "")
	flag.StringVar(&dropSession, "drop", "", "")
	flag.StringVar(&showSession, "show", "", "")
	flag.StringVar(&paramsJSON, "params", "", "")
	flag.StringVar(&filePath, "file", "", "")
	flag.Var(&format, "format", "")
	flag.IntVar(&cleanupDays, "cleanup-days", 14, "")
	flag.IntVar(&timeout, "timeout", 30, "")
	flag.BoolVar(&replMode, "repl", false, "")
	flag.BoolVar(&commitFlag, "commit", false, "")
	flag.BoolVar(&listSessions, "list", false, "")
	flag.BoolVar(&showHelp, "help", false, "")

	flag.Usage = func() {
		fmt.Fprint(os.Stderr, helpText)
	}

	flag.Parse()

	outputFormat := format.value
	formatWasSet := format.set

	if showHelp {
		fmt.Print(helpText)
		os.Exit(0)
	}

	// Clean stale sessions
	if cleanupDays > 0 {
		if err := cleanStaleSessionsIfNeeded(cleanupDays); err != nil {
			log.Printf("Warning: Failed to clean stale sessions: %v", err)
		}
	}

	// Handle session list/show/drop commands
	if listSessions {
		sessions, err := getSessions()
		if err != nil {
			log.Fatalf("Failed to list sessions: %v", err)
		}
		if len(sessions) == 0 {
			fmt.Println("No active sessions")
			return
		}
		for _, s := range sessions {
			fmt.Printf("Session: %s\n", s.Name)
			fmt.Printf("  Database: %s\n", s.Database)
			fmt.Printf("  Created: %s\n", s.CreatedAt.Format(time.RFC3339))
			fmt.Printf("  Queries: %d\n", len(s.Queries))
			fmt.Println()
		}
		return
	}

	if dropSession != "" {
		if err := deleteSession(dropSession); err != nil {
			log.Fatalf("Failed to drop session: %v", err)
		}
		fmt.Printf("✓ Session '%s' dropped\n", dropSession)
		return
	}

	if showSession != "" {
		if err := printSessionDetails(showSession); err != nil {
			log.Fatalf("Failed to show session: %v", err)
		}
		return
	}

	// Handle preset loading
	if presetName != "" {
		preset, err := getPreset(presetName)
		if err != nil {
			log.Fatalf("Failed to load preset '%s': %v\n\nPresets are stored in: ~/.cache/psql-safe/presets.json", presetName, err)
		}
		host = preset.Host
		port = fmt.Sprintf("%d", preset.Port)
		username = preset.User
		database = preset.Database
		if preset.Password != "" {
			password = preset.Password
		}
		if preset.SSLMode != "" {
			sslMode = preset.SSLMode
		}
		if preset.CertDir != "" {
			certDir = expandPath(preset.CertDir)
		}
	}

	// Get password from environment if not set
	// Leave empty to allow cert-based or passwordless auth
	if password == "" {
		password = os.Getenv("COCKROACH_PASSWORD")
		if password == "" {
			password = os.Getenv("PGPASSWORD")
		}
		// Don't set a fake default - let pgx use certificates or configured auth
	}

	// Get SSL mode from environment if not set
	if sslMode == "" {
		sslMode = os.Getenv("PGSSLMODE")
		if sslMode == "" {
			sslMode = "require"
		}
	}

	// Validate output format
	if outputFormat != "json" && outputFormat != "compact" && outputFormat != "csv" && outputFormat != "table" {
		log.Fatalf("Invalid format: %s (must be json, compact, csv, or table)", outputFormat)
	}

	// Get SQL query
	var sqlQuery string
	if filePath != "" {
		if filePath == "-" {
			// Read from stdin
			content, err := os.ReadFile("/dev/stdin")
			if err != nil {
				log.Fatalf("Failed to read from stdin: %v", err)
			}
			sqlQuery = string(content)
		} else {
			content, err := os.ReadFile(filePath)
			if err != nil {
				log.Fatalf("Failed to read file %s: %v", filePath, err)
			}
			sqlQuery = string(content)
		}
	} else if flag.NArg() > 0 && !replMode {
		args := flag.Args()
		for i, arg := range args {
			if strings.HasPrefix(arg, "-") {
				log.Fatalf("Error: Flags must come BEFORE the query. Found flag-like argument '%s' at position %d.\n\nCorrect usage:\n  psql-safe [FLAGS] \"QUERY\"\n  psql-safe --format table \"SELECT * FROM customers\"\n\nIncorrect:\n  psql-safe \"SELECT * FROM customers\" --format table", arg, i+1)
			}
		}
		sqlQuery = strings.Join(args, " ")
	} else if !replMode && !commitFlag {
		// Allow no query if --commit is specified (will just commit the session)
		fmt.Fprint(os.Stderr, "Error: No query provided\n\n")
		fmt.Fprint(os.Stderr, helpText)
		os.Exit(1)
	}

	// Parse parameters
	var params map[string]any
	if paramsJSON != "" {
		if err := json.Unmarshal([]byte(paramsJSON), &params); err != nil {
			log.Fatalf("Failed to parse --params JSON: %v", err)
		}
	}

	// Build connection config
	cfg := ConnConfig{
		Host:     host,
		Port:     port,
		User:     username,
		Database: database,
		Password: password,
		SSLMode:  sslMode,
		CertDir:  certDir,
	}

	// Create connection pool
	ctx := context.Background()
	pool, err := createPool(ctx, cfg, timeout)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer pool.Close()

	// Handle REPL mode
	if replMode {
		replFormat := outputFormat
		if !formatWasSet {
			replFormat = "table"
		}
		replCfg := replConfig{
			SessionName:  sessionName,
			Database:     database,
			OutputFormat: replFormat,
			ReadOnly:     sessionName == "",
			Params:       params,
		}
		if err := startRepl(ctx, pool, replCfg); err != nil {
			log.Fatalf("REPL exited with error: %v", err)
		}
		return
	}

	// Handle commit-only mode (no new query, just commit existing session)
	if commitFlag && sqlQuery == "" {
		if sessionName == "" {
			log.Fatalf("--commit requires --session to specify which session to commit")
		}
		if err := commitSession(ctx, pool, sessionName, database); err != nil {
			log.Fatalf("Commit failed: %v", err)
		}
		return
	}

	// Handle session mode
	if sessionName != "" {
		if err := executeInSession(ctx, pool, sessionName, database, sqlQuery, params, outputFormat, commitFlag); err != nil {
			log.Fatalf("Query execution failed: %v", err)
		}
	} else {
		// Default read-only mode
		if err := executeReadOnly(ctx, pool, sqlQuery, params, outputFormat); err != nil {
			log.Fatalf("Query execution failed: %v", err)
		}
	}
}
