package main

import (
	"context"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// enforceReadOnly checks if the SQL contains write keywords
func enforceReadOnly(sql string) error {
	upperSQL := strings.ToUpper(strings.TrimSpace(sql))

	writeKeywords := []string{
		"INSERT", "UPDATE", "DELETE", "UPSERT",
		"ALTER", "CREATE", "DROP",
		"TRUNCATE", "GRANT", "REVOKE",
		"COPY",
	}

	for _, keyword := range writeKeywords {
		// Check if keyword appears as a separate word
		if strings.Contains(upperSQL, keyword+" ") ||
			strings.HasPrefix(upperSQL, keyword) {
			return fmt.Errorf("write operation detected: %s\n\nWrites are disabled in default mode. Use --session to safely test write queries:\n  psql-safe --session test-name \"YOUR_QUERY\"", keyword)
		}
	}

	return nil
}

// executeReadOnly executes a query in read-only mode
func executeReadOnly(ctx context.Context, pool *pgxpool.Pool, sql string, params map[string]any, format string) error {
	// Enforce DDL policy
	if err := enforceDDLPolicy(sql); err != nil {
		return err
	}

	// Reject multi-statement queries
	if err := rejectMultiStatement(sql); err != nil {
		return err
	}

	// Check for write operations
	if err := enforceReadOnly(sql); err != nil {
		return err
	}

	// Begin a read-only transaction
	tx, err := pool.BeginTx(ctx, pgx.TxOptions{
		IsoLevel: pgx.Serializable,
		AccessMode: pgx.ReadOnly,
	})
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	// Prepare parameters
	var queryParams []any
	rewrittenSQL := sql

	if len(params) > 0 {
		// Try named parameter rewriting
		if strings.Contains(sql, ":") {
			var err error
			rewrittenSQL, queryParams, err = rewriteNamedParams(sql, params)
			if err != nil {
				return fmt.Errorf("failed to rewrite named parameters: %w", err)
			}
		} else {
			// Try converting map to slice for positional params
			var err error
			queryParams, err = convertMapToSlice(params)
			if err != nil {
				return fmt.Errorf("failed to convert parameters: %w", err)
			}
		}
	}

	// Execute the query
	rows, err := tx.Query(ctx, rewrittenSQL, queryParams...)
	if err != nil {
		return fmt.Errorf("failed to execute query: %w", err)
	}
	defer rows.Close()

	// Convert rows to maps
	results, err := rowsToMaps(rows)
	if err != nil {
		return err
	}

	// Always rollback (read-only mode)
	if err := tx.Rollback(ctx); err != nil {
		return fmt.Errorf("failed to rollback transaction: %w", err)
	}

	// Print results
	return printResults(results, format)
}
