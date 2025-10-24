package main

import (
	"fmt"
	"strings"
)

// allowDDL controls whether DDL statements are permitted.
// Set to true and recompile to enable DDL (not recommended for safe operation).
const allowDDL = false

// enforceDDLPolicy checks if SQL contains DDL statements and blocks them if allowDDL is false
func enforceDDLPolicy(sql string) error {
	if allowDDL {
		return nil // DDL is enabled
	}

	// Strip comments and strings to avoid false positives and ensure we catch DDL after comments
	cleaned := removeCommentsAndStrings(sql)
	upperSQL := strings.ToUpper(strings.TrimSpace(cleaned))

	ddlKeywords := []string{
		"CREATE",      // Create database objects (tables, indexes, views, databases)
		"ALTER",       // Modify existing database objects
		"DROP",        // Remove database objects
		"TRUNCATE",    // Quickly remove all rows from a table
		"RENAME",      // Rename tables or other objects
		"COMMENT",     // Add comments for object documentation
		"GRANT",       // Grant privileges
		"REVOKE",      // Revoke privileges
		"SET CLUSTER", // Cluster-level settings
	}

	for _, keyword := range ddlKeywords {
		// Check if keyword appears anywhere as a separate word
		// Look for: start of string, or preceded by whitespace
		if strings.HasPrefix(upperSQL, keyword+" ") ||
			strings.HasPrefix(upperSQL, keyword+"\t") ||
			strings.HasPrefix(upperSQL, keyword+"\n") ||
			strings.HasPrefix(upperSQL, keyword+"(") ||
			upperSQL == keyword ||
			strings.Contains(upperSQL, " "+keyword+" ") ||
			strings.Contains(upperSQL, "\t"+keyword+" ") ||
			strings.Contains(upperSQL, "\n"+keyword+" ") ||
			strings.Contains(upperSQL, " "+keyword+"\t") ||
			strings.Contains(upperSQL, " "+keyword+"\n") ||
			strings.Contains(upperSQL, " "+keyword+"(") {
			return fmt.Errorf("DDL statements are disabled in psql-safe for safety.\n\nDetected: %s\n\nReason: CockroachDB schema changes may auto-commit even inside transactions,\nbreaking the rollback guarantee. This prevents accidental schema modifications.\n\nIf you absolutely need DDL:\n1. Use the native cockroach/psql CLI\n2. Or rebuild psql-safe with allowDDL=true in statement_guard.go", keyword)
		}
	}

	return nil
}

// rejectMultiStatement checks if SQL contains multiple statements (multiple semicolons)
// and returns an error if found
func rejectMultiStatement(sql string) error {
	// Remove comments and string literals to avoid false positives
	cleaned := removeCommentsAndStrings(sql)

	// Count non-trailing semicolons
	trimmed := strings.TrimSpace(cleaned)
	semicolonCount := strings.Count(trimmed, ";")

	// Allow one trailing semicolon
	if strings.HasSuffix(trimmed, ";") {
		semicolonCount--
	}

	if semicolonCount > 0 {
		return fmt.Errorf("multi-statement queries are not allowed\n\nFound %d statement separator(s). psql-safe only accepts single statements.\n\nTo run multiple statements, use --session mode and issue them one at a time.", semicolonCount)
	}

	return nil
}

// removeCommentsAndStrings removes SQL comments and string literals to avoid
// counting semicolons inside them
func removeCommentsAndStrings(sql string) string {
	var result strings.Builder
	inString := false
	inLineComment := false
	inBlockComment := false
	stringChar := rune(0)

	runes := []rune(sql)
	for i := 0; i < len(runes); i++ {
		ch := runes[i]

		// Handle block comments /* */
		if !inString && !inLineComment && i < len(runes)-1 {
			if ch == '/' && runes[i+1] == '*' {
				inBlockComment = true
				i++
				continue
			}
		}
		if inBlockComment {
			if ch == '*' && i < len(runes)-1 && runes[i+1] == '/' {
				inBlockComment = false
				i++
			}
			continue
		}

		// Handle line comments --
		if !inString && !inBlockComment && i < len(runes)-1 {
			if ch == '-' && runes[i+1] == '-' {
				inLineComment = true
				i++
				continue
			}
		}
		if inLineComment {
			if ch == '\n' {
				inLineComment = false
			}
			continue
		}

		// Handle strings
		if !inBlockComment && !inLineComment {
			if ch == '\'' || ch == '"' {
				if !inString {
					inString = true
					stringChar = ch
					continue
				} else if ch == stringChar {
					// Check for escaped quote
					if i < len(runes)-1 && runes[i+1] == stringChar {
						i++ // Skip escaped quote
						continue
					}
					inString = false
					continue
				}
			}
			if inString {
				continue
			}
		}

		result.WriteRune(ch)
	}

	return result.String()
}
