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

	neogotooling "github.com/MathGaps/resources/pkg/neogo_tooling"
)

const helpText = `cypher-safe - Execute Cypher queries against a Neo4j database

USAGE:
    cypher-safe [OPTIONS] <QUERY>              # Read-only mode
    cypher-safe --session <NAME> <QUERY>       # Session mode (safe write testing)
    cypher-safe --session <NAME> --repl        # Session REPL with :commit support
    cypher-safe --list                         # List active sessions
    cypher-safe --show <SESSION_NAME>          # Display session queries
    cypher-safe --drop <SESSION_NAME>          # Delete a session

IMPORTANT: All flags MUST come BEFORE the query string, not after.

DESCRIPTION:
    A command-line tool for executing Cypher queries against Neo4j.
		Results are printed as JSON array suitable for piping to jq.

    TWO MODES:

    1. DEFAULT MODE (Read-Only):
       Perfect for safe data exploration and reporting.

    2. SESSION MODE (Safe Write Testing):
       Use --session <name> to create a safe testing environment where you can:
       - Run multiple write queries (CREATE, MERGE, SET, DELETE ...)
       - Inspect intermediate results after each query
       - See how data evolves across multiple operations
       - Everything is AUTOMATICALLY ROLLED BACK (nothing persists to DB), and replayed when you access the same session.
       - Think of it like Cypher Shell's :begin/:rollback workflow
       - Perfect for testing complex write operations before committing

OPTIONS:
    -h, --host <HOST>         Neo4j host address (default: localhost)
    -p, --port <PORT>         Neo4j bolt port (default: 7686)
    -u, --user <USER>         Neo4j username (default: neo4j)
    -d, --db <DATABASE>       Neo4j database name (default: neo4j)
    --preset <NAME>           Use connection preset (cannot combine with --host/--port)
    --session <NAME>          Execute query in named session (auto-creates if new)
    --params <JSON>           Query parameters as JSON object (e.g. '{"name":"Alice","age":30}')
    --format <FORMAT>         Output format: json, compact, csv, table (default: json)
    --list                    List all active sessions with metadata
    --show <SESSION_NAME>     Display session queries without executing
    --drop <SESSION_NAME>     Delete a session (all queries discarded)
    --help                    Show this help message

ENVIRONMENT VARIABLES:
    NEO4J_PASSWORD            Neo4j password (defaults to "password" if not set)

EXAMPLES:

  Read-only queries (default mode):
    $ export NEO4J_PASSWORD=password
    $ cypher-safe "MATCH (n) RETURN n LIMIT 5"
    $ cypher-safe "MATCH (n) RETURN count(n) as total"
    $ cypher-safe "MATCH (p:Person) RETURN p.name, p.age ORDER BY p.age DESC"

  Session mode - safe write testing workflow:

    # Create first test record (auto-creates session "test")
    $ cypher-safe --session test \
        "CREATE (p:Person {name: 'Alice', age: 30}) RETURN p"
    [{"p": {"name": "Alice", "age": 30}}]
    
    # Add second record - sees Alice from replayed CREATE
    $ cypher-safe --session test \
        "CREATE (p:Person {name: 'Bob', age: 25}) RETURN p.name, p.age"
    [{"p.age": 25, "p.name": "Bob"}]
    
    # Query all people - sees BOTH Alice and Bob
    $ cypher-safe --session test \
        "MATCH (p:Person) RETURN p.name as name, p.age as age ORDER BY p.age"
    [{"age": 25, "name": "Bob"}, {"age": 30, "name": "Alice"}]
    
    # Modify records and inspect
    $ cypher-safe --session test \
        "MATCH (p:Person) WHERE p.age > 28 SET p.senior = true RETURN p"
    [{"p": {"age": 30, "name": "Alice", "senior": true}}]
    
    # Create relationships
    $ cypher-safe --session test \
        "MATCH (a:Person {name: 'Alice'}), (b:Person {name: 'Bob'})
         CREATE (a)-[r:KNOWS]->(b) RETURN type(r) as relationship"
    [{"relationship": "KNOWS"}]
    
    # Query the graph structure you built
    $ cypher-safe --session test \
        "MATCH (a:Person)-[r:KNOWS]->(b:Person) 
         RETURN a.name as from, b.name as to"
    [{"from": "Alice", "to": "Bob"}]
    
    # CRITICAL: Verify database is UNCHANGED (all rolled back!)
    $ cypher-safe "MATCH (p:Person) RETURN count(p) as total"
    [{"total": 0}]  # Nothing was committed!
    
    # Inspect session metadata
    $ cypher-safe --list
    Session: test
      Database: neo4j
      Created: 2025-10-18T22:15:23+05:45
      Queries: 6
    
    # When done testing, drop session (or keep for later)
    $ cypher-safe --drop test
    ✓ Session 'test' dropped

  Pipe results to jq:
    $ cypher-safe "MATCH (t:Topic) RETURN t.name as name" | jq -r '.[].name'

  Connect to remote server:
    $ cypher-safe -h prod.example.com -p 7687 \
        "MATCH (n) RETURN count(n) as total"

  Using presets for easy connection:
    $ cypher-safe --preset resources-dev "MATCH (n) RETURN count(n) as total"

  Multiple sessions for different tests:
    $ cypher-safe --session user-test "CREATE (u:User {id: 1}) WITH u as results RETURN results"
    $ cypher-safe --session product-test "CREATE (p:Product {id: 1}) WITH p as results RETURN results"
    $ cypher-safe --list
    # Shows both sessions independently

WHEN TO USE EACH MODE:
    - Use DEFAULT for: Reading data, generating reports, exploring the graph
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
	var host, port, username, database string
	var showHelp, listSessions bool
	var sessionName, dropSession, showSession, paramsJSON, presetName string
	var cleanupDays int
	var replMode bool
	format := formatFlag{value: "json"}

	flag.StringVar(&host, "host", "localhost", "")
	flag.StringVar(&host, "h", "localhost", "")
	flag.StringVar(&port, "port", "7686", "")
	flag.StringVar(&port, "p", "7686", "")
	flag.StringVar(&username, "user", "neo4j", "")
	flag.StringVar(&username, "u", "neo4j", "")
	flag.StringVar(&database, "db", "neo4j", "")
	flag.StringVar(&database, "d", "neo4j", "")
	flag.StringVar(&presetName, "preset", "", "")
	flag.StringVar(&sessionName, "session", "", "")
	flag.StringVar(&dropSession, "drop", "", "")
	flag.StringVar(&showSession, "show", "", "")
	flag.StringVar(&paramsJSON, "params", "", "")
	flag.Var(&format, "format", "")
	flag.IntVar(&cleanupDays, "cleanup-days", 30, "")
	flag.BoolVar(&replMode, "repl", false, "")
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

	if err := ensureDefaultPresets(); err != nil {
		log.Printf("Warning: Failed to initialize default presets: %v", err)
	}

	if presetName != "" {
		hostProvided := false
		portProvided := false
		flag.Visit(func(f *flag.Flag) {
			if f.Name == "host" || f.Name == "h" {
				hostProvided = true
			}
			if f.Name == "port" || f.Name == "p" {
				portProvided = true
			}
		})

		if hostProvided || portProvided {
			log.Fatalf("Error: Cannot use --preset with --host/-h or --port/-p flags.\n\nWhen using a preset, connection details are loaded from the preset configuration.")
		}

		preset, err := getPreset(presetName)
		if err != nil {
			log.Fatalf("Failed to load preset '%s': %v\n\nAvailable presets can be viewed in: ~/.cache/scripts/cypher-safe/presets.json", presetName, err)
		}

		host = preset.URI
		username = preset.Username
		database = preset.Database
		if preset.Password != "" {
			_ = os.Setenv("NEO4J_PASSWORD", preset.Password)
		}
	}

	if cleanupDays > 0 {
		if err := cleanStaleSessionsIfNeeded(cleanupDays); err != nil {
			log.Printf("Warning: Failed to clean stale sessions: %v", err)
		}
	}

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
		sessionPath, err := getSessionPath(showSession)
		if err != nil {
			log.Fatalf("Failed to get session path: %v", err)
		}
		session, err := loadSession(sessionPath)
		if err != nil {
			log.Fatalf("Failed to load session: %v", err)
		}
		fmt.Printf("Session: %s\n", session.Name)
		fmt.Printf("Database: %s\n", session.Database)
		fmt.Printf("Created: %s\n", session.CreatedAt.Format(time.RFC3339))
		fmt.Printf("Queries (%d):\n", len(session.Queries))
		for i, qp := range session.Queries {
			fmt.Printf("\n[%d]\n%s\n", i+1, qp.Query)
			if len(qp.Params) > 0 {
				paramsJSON, _ := json.MarshalIndent(qp.Params, "  ", "  ")
				fmt.Printf("  Params: %s\n", string(paramsJSON))
			}
		}
		return
	}

	if flag.NArg() == 0 && !replMode {
		fmt.Fprint(os.Stderr, "Error: No query provided\n\n")
		fmt.Fprint(os.Stderr, helpText)
		os.Exit(1)
	}

	var cypherQuery string
	if flag.NArg() > 0 {
		args := flag.Args()
		for i, arg := range args {
			if strings.HasPrefix(arg, "-") {
				log.Fatalf("Error: Flags must come BEFORE the query. Found flag-like argument '%s' at position %d.\n\nCorrect usage:\n  cypher-safe [FLAGS] \"QUERY\"\n  cypher-safe --format table \"MATCH (n) RETURN n\"\n\nIncorrect:\n  cypher-safe \"MATCH (n) RETURN n\" --format table", arg, i+1)
			}
		}
		cypherQuery = strings.Join(args, " ")
	}

	var params map[string]any
	if paramsJSON != "" {
		if err := json.Unmarshal([]byte(paramsJSON), &params); err != nil {
			log.Fatalf("Failed to parse --params JSON: %v", err)
		}
	}

	if outputFormat != "json" && outputFormat != "compact" && outputFormat != "csv" && outputFormat != "table" {
		log.Fatalf("Invalid format: %s (must be json, compact, csv, or table)", outputFormat)
	}

	password := os.Getenv("NEO4J_PASSWORD")
	if password == "" {
		password = "password"
	}

	var uri string
	if presetName != "" {
		uri = host
	} else {
		uri = fmt.Sprintf("neo4j://%s:%s", host, port)
	}

	config := neogotooling.Config{
		URI:      uri,
		Username: username,
		Password: password,
	}

	deps := neogotooling.Dependencies{
		Config: config,
		CompositeDatabaseConfig: neogotooling.CompositeDatabaseConfig{
			Enabled: false,
		},
	}

	client, err := neogotooling.New(deps)
	if err != nil {
		log.Fatalf("Failed to create Neo4j client: %v", err)
	}
	defer func() {
		if err := client.DB().Close(context.Background()); err != nil {
			log.Printf("Error closing Neo4j client: %v", err)
		}
	}()

	ctx := context.Background()

	if replMode {
		replFormat := outputFormat
		if !formatWasSet {
			replFormat = "table"
		}
		cfg := replConfig{
			SessionName:  sessionName,
			Database:     database,
			OutputFormat: replFormat,
			ReadOnly:     sessionName == "",
			Params:       params,
		}
		if err := startRepl(ctx, client, cfg); err != nil {
			log.Fatalf("REPL exited with error: %v", err)
		}
		return
	}

	if sessionName != "" {
		if err := executeRawQueryInSession(ctx, client, sessionName, database, cypherQuery, params, outputFormat); err != nil {
			log.Fatalf("Query execution failed: %v", err)
		}
	} else {
		if err := executeRawQuery(ctx, client, cypherQuery, params, outputFormat, true); err != nil {
			log.Fatalf("Query execution failed: %v", err)
		}
	}
}
