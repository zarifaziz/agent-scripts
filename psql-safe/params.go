package main

import (
	"fmt"
	"regexp"
	"strings"
)

// rewriteNamedParams converts named parameters (:param) to positional parameters ($1, $2, ...)
// and returns the rewritten SQL and the ordered parameter values
func rewriteNamedParams(sql string, params map[string]any) (string, []any, error) {
	if len(params) == 0 {
		return sql, nil, nil
	}

	// Find all named parameters in the SQL
	re := regexp.MustCompile(`:([a-zA-Z_][a-zA-Z0-9_]*)`)
	matches := re.FindAllStringSubmatch(sql, -1)

	if len(matches) == 0 {
		// No named parameters found, assume positional parameters
		// Convert map to slice based on numeric keys if possible
		return sql, nil, nil
	}

	// Build a map of parameter names to their position (first-appearance order)
	paramNames := make(map[string]int)
	paramList := []string{}
	for _, match := range matches {
		paramName := match[1]
		if _, exists := paramNames[paramName]; !exists {
			paramList = append(paramList, paramName)
			paramNames[paramName] = len(paramList) // Position based on first appearance
		}
	}

	// Rewrite the SQL
	rewrittenSQL := re.ReplaceAllStringFunc(sql, func(match string) string {
		paramName := strings.TrimPrefix(match, ":")
		if pos, ok := paramNames[paramName]; ok {
			return fmt.Sprintf("$%d", pos)
		}
		return match
	})

	// Build the ordered parameter slice
	orderedParams := make([]any, len(paramList))
	for i, name := range paramList {
		val, ok := params[name]
		if !ok {
			return "", nil, fmt.Errorf("parameter :%s not found in provided parameters", name)
		}
		orderedParams[i] = val
	}

	return rewrittenSQL, orderedParams, nil
}

// convertMapToSlice tries to convert a map[string]any to []any for positional parameters
// Expects keys to be "1", "2", "3", etc.
func convertMapToSlice(params map[string]any) ([]any, error) {
	if len(params) == 0 {
		return nil, nil
	}

	// Check if all keys are numeric strings
	maxIdx := 0
	for key := range params {
		var idx int
		if _, err := fmt.Sscanf(key, "%d", &idx); err != nil {
			// Not a numeric key, return as-is (will be handled by named params)
			return nil, nil
		}
		if idx > maxIdx {
			maxIdx = idx
		}
	}

	// Build slice
	result := make([]any, maxIdx)
	for i := 1; i <= maxIdx; i++ {
		key := fmt.Sprintf("%d", i)
		val, ok := params[key]
		if !ok {
			return nil, fmt.Errorf("missing parameter at position %d", i)
		}
		result[i-1] = val
	}

	return result, nil
}
