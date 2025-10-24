package main

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// executeInSession runs a query within a named session
func executeInSession(ctx context.Context, pool *pgxpool.Pool, sessionName, database, sql string, params map[string]any, format string, shouldCommit bool) error {
	sessionPath, err := getSessionPath(sessionName)
	if err != nil {
		return err
	}

	exists, err := sessionExists(sessionName)
	if err != nil {
		return err
	}

	var sessionData *SessionData
	if !exists {
		sessionData = &SessionData{
			Name:      sessionName,
			Database:  database,
			Queries:   []QueryEntry{},
			CreatedAt: time.Now(),
			TxOptions: TxOptions{
				IsoLevel: "SERIALIZABLE",
				ReadOnly: false,
			},
		}
	} else {
		sessionData, err = loadSession(sessionPath)
		if err != nil {
			return err
		}
		if sessionData.Database != database {
			return fmt.Errorf("session %s exists with different database %s (requested: %s)", sessionName, sessionData.Database, database)
		}
	}

	// Enforce DDL policy
	if err := enforceDDLPolicy(sql); err != nil {
		return err
	}

	// Reject multi-statement queries
	if err := rejectMultiStatement(sql); err != nil {
		return err
	}

	trimmedSQL := strings.TrimSpace(sql)
	trimmedSQL = strings.TrimSuffix(trimmedSQL, ";")

	newQuery := QueryEntry{
		SQL:    trimmedSQL,
		Params: params,
	}

	// Begin a transaction
	tx, err := pool.BeginTx(ctx, pgx.TxOptions{
		IsoLevel:   pgx.Serializable,
		AccessMode: pgx.ReadWrite,
	})
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}

	var committed bool
	defer func() {
		if !committed {
			_ = tx.Rollback(ctx)
		}
	}()

	// Replay all stored queries
	for i, qp := range sessionData.Queries {
		var queryParams []any
		rewrittenSQL := qp.SQL

		if len(qp.Params) > 0 {
			if strings.Contains(qp.SQL, ":") {
				var err error
				rewrittenSQL, queryParams, err = rewriteNamedParams(qp.SQL, qp.Params)
				if err != nil {
					return fmt.Errorf("query %d failed (parameter rewriting): %w", i+1, err)
				}
			} else {
				var err error
				queryParams, err = convertMapToSlice(qp.Params)
				if err != nil {
					return fmt.Errorf("query %d failed (parameter conversion): %w", i+1, err)
				}
			}
		}

		rows, err := tx.Query(ctx, rewrittenSQL, queryParams...)
		if err != nil {
			return fmt.Errorf("query %d failed during replay: %w\nSQL: %.80s", i+1, err, qp.SQL)
		}
		rows.Close()
	}

	// Execute the new query
	var queryParams []any
	rewrittenSQL := newQuery.SQL

	if len(newQuery.Params) > 0 {
		if strings.Contains(newQuery.SQL, ":") {
			var err error
			rewrittenSQL, queryParams, err = rewriteNamedParams(newQuery.SQL, newQuery.Params)
			if err != nil {
				return fmt.Errorf("failed to rewrite named parameters: %w", err)
			}
		} else {
			var err error
			queryParams, err = convertMapToSlice(newQuery.Params)
			if err != nil {
				return fmt.Errorf("failed to convert parameters: %w", err)
			}
		}
	}

	rows, err := tx.Query(ctx, rewrittenSQL, queryParams...)
	if err != nil {
		return fmt.Errorf("query %d failed: %w", len(sessionData.Queries)+1, err)
	}
	defer rows.Close()

	// Collect results
	results, err := rowsToMaps(rows)
	if err != nil {
		return fmt.Errorf("failed to collect results: %w", err)
	}
	rows.Close()

	// Handle commit or rollback
	if shouldCommit {
		if err := tx.Commit(ctx); err != nil {
			return fmt.Errorf("failed to commit transaction: %w", err)
		}
		committed = true

		// Delete session file after successful commit
		if err := deleteSession(sessionName); err != nil {
			return fmt.Errorf("transaction committed but failed to delete session: %w", err)
		}

		fmt.Fprintf(os.Stderr, "✓ Session '%s' committed (%d queries applied)\n", sessionName, len(sessionData.Queries)+1)
	} else {
		// Rollback (default behavior)
		if err := tx.Rollback(ctx); err != nil {
			return fmt.Errorf("failed to rollback transaction: %w", err)
		}

		// Save the session with the new query appended
		sessionData.Queries = append(sessionData.Queries, newQuery)
		if err := saveSession(sessionPath, sessionData); err != nil {
			return fmt.Errorf("failed to save session: %w", err)
		}
	}

	// Print results
	return printResults(results, format)
}

// commitSession commits all queries in a session
func commitSession(ctx context.Context, pool *pgxpool.Pool, sessionName, database string) error {
	sessionPath, err := getSessionPath(sessionName)
	if err != nil {
		return err
	}

	sessionData, err := loadSession(sessionPath)
	if err != nil {
		if os.IsNotExist(err) {
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

	// Begin transaction
	tx, err := pool.BeginTx(ctx, pgx.TxOptions{
		IsoLevel:   pgx.Serializable,
		AccessMode: pgx.ReadWrite,
	})
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}

	var committed bool
	defer func() {
		if !committed {
			_ = tx.Rollback(ctx)
		}
	}()

	// Execute all queries
	for i, qp := range sessionData.Queries {
		var queryParams []any
		rewrittenSQL := qp.SQL

		if len(qp.Params) > 0 {
			if strings.Contains(qp.SQL, ":") {
				var err error
				rewrittenSQL, queryParams, err = rewriteNamedParams(qp.SQL, qp.Params)
				if err != nil {
					return fmt.Errorf("query %d failed (parameter rewriting): %w", i+1, err)
				}
			} else {
				var err error
				queryParams, err = convertMapToSlice(qp.Params)
				if err != nil {
					return fmt.Errorf("query %d failed (parameter conversion): %w", i+1, err)
				}
			}
		}

		rows, err := tx.Query(ctx, rewrittenSQL, queryParams...)
		if err != nil {
			return fmt.Errorf("commit aborted at query %d: %w\nSQL: %.80s", i+1, err, qp.SQL)
		}
		rows.Close()
	}

	// Commit the transaction
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit session: %w", err)
	}
	committed = true

	// Delete the session file
	if err := deleteSession(sessionName); err != nil {
		return fmt.Errorf("session committed but cleanup failed: %w", err)
	}

	fmt.Fprintf(os.Stderr, "✓ Session '%s' committed (%d queries applied)\n", sessionName, len(sessionData.Queries))
	return nil
}
