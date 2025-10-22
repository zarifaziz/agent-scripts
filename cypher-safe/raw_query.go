package main

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
	"github.com/rlch/neogo"
)

func executeRawQuery(ctx context.Context, client neogo.Driver, cypherQuery string, params map[string]any, outputFormat string, readOnly bool) error {
	trimmedQuery := strings.TrimSpace(cypherQuery)

	rawDriver := client.DB()

	accessMode := neo4j.AccessModeWrite
	if readOnly {
		accessMode = neo4j.AccessModeRead
	}

	session := rawDriver.NewSession(ctx, neo4j.SessionConfig{
		AccessMode: accessMode,
	})
	defer func() {
		_ = session.Close(ctx)
	}()

	var results []map[string]any
	var executeErr error

	if readOnly {
		_, executeErr = session.ExecuteRead(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
			result, err := tx.Run(ctx, trimmedQuery, params)
			if err != nil {
				return nil, err
			}

			records, err := result.Collect(ctx)
			if err != nil {
				return nil, err
			}

			for _, record := range records {
				results = append(results, record.AsMap())
			}

			return nil, nil
		})
	} else {
		_, executeErr = session.ExecuteWrite(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
			result, err := tx.Run(ctx, trimmedQuery, params)
			if err != nil {
				return nil, err
			}

			records, err := result.Collect(ctx)
			if err != nil {
				return nil, err
			}

			for _, record := range records {
				results = append(results, record.AsMap())
			}

			return nil, nil
		})
	}

	if executeErr != nil {
		if readOnly {
			errMsg := executeErr.Error()
			if strings.Contains(errMsg, "Writing in read access mode not allowed") ||
				strings.Contains(errMsg, "Write queries cannot be performed") {
				return fmt.Errorf("write operation detected in read-only mode\n\nHint: Use --session mode to safely test write queries:\n  cypher-query --session test-name \"YOUR_QUERY\"\n\nOriginal error: %w", executeErr)
			}
		}
		return fmt.Errorf("failed to execute query: %w", executeErr)
	}

	return printResults(results, outputFormat)
}

func executeRawQueryInSession(ctx context.Context, client neogo.Driver, sessionName, database, cypherQuery string, params map[string]any, outputFormat string) error {
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
			Queries:   []QueryWithParams{},
			CreatedAt: time.Now(),
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

	trimmedQuery := strings.TrimSpace(cypherQuery)
	newQuery := QueryWithParams{
		Query:  trimmedQuery,
		Params: params,
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
		return err
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	var lastResults []map[string]any

	for i, qp := range sessionData.Queries {
		result, err := tx.Run(ctx, qp.Query, qp.Params)
		if err != nil {
			return fmt.Errorf("query %d failed: %w", i+1, err)
		}

		_, err = result.Collect(ctx)
		if err != nil {
			return fmt.Errorf("query %d failed to collect results: %w", i+1, err)
		}
	}

	result, err := tx.Run(ctx, newQuery.Query, newQuery.Params)
	if err != nil {
		return fmt.Errorf("query %d failed: %w", len(sessionData.Queries)+1, err)
	}

	records, err := result.Collect(ctx)
	if err != nil {
		return fmt.Errorf("query %d failed to collect results: %w", len(sessionData.Queries)+1, err)
	}

	for _, record := range records {
		lastResults = append(lastResults, record.AsMap())
	}

	sessionData.Queries = append(sessionData.Queries, newQuery)

	if err := saveSession(sessionPath, sessionData); err != nil {
		return err
	}

	return printResults(lastResults, outputFormat)
}
