package main

import (
	"encoding/json"
	"strings"
)

// TransformFilter takes LLM-friendly JSON and normalizes it to Notion API format.
// Handles:
// - User name -> UUID resolution in people filters
// - Case-insensitive property name matching
// - Case-insensitive enum value matching (status, select, multi_select)
// - Shorthand syntax expansion: {"assign": "hemanta"} -> full filter
// Safe to call with nil schema (skips normalization, still resolves users from global map)
func TransformFilter(raw json.RawMessage, schema *DatabaseSchema) (json.RawMessage, error) {
	if len(raw) == 0 {
		return raw, nil
	}

	var data map[string]any
	if err := json.Unmarshal(raw, &data); err != nil {
		return nil, err
	}

	// If no schema, still do basic transforms (user resolution uses global map)
	transformed := transformNode(data, schema)
	return json.Marshal(transformed)
}

func transformNode(data map[string]any, schema *DatabaseSchema) map[string]any {
	// Handle compound filters
	if andFilters, ok := data["and"].([]any); ok {
		var transformed []any
		for _, f := range andFilters {
			if fm, ok := f.(map[string]any); ok {
				transformed = append(transformed, transformNode(fm, schema))
			}
		}
		return map[string]any{"and": transformed}
	}
	if orFilters, ok := data["or"].([]any); ok {
		var transformed []any
		for _, f := range orFilters {
			if fm, ok := f.(map[string]any); ok {
				transformed = append(transformed, transformNode(fm, schema))
			}
		}
		return map[string]any{"or": transformed}
	}

	// Check for shorthand syntax: {"assign": "hemanta", "status": "In progress"}
	if _, hasProperty := data["property"]; !hasProperty {
		return expandShorthand(data, schema)
	}

	// Full filter syntax - normalize it
	return normalizeFilter(data, schema)
}

// expandShorthand converts {"assign": "hemanta", "status": "Done"} to proper filters
func expandShorthand(data map[string]any, schema *DatabaseSchema) map[string]any {
	var filters []map[string]any

	for key, value := range data {
		propName := normalizePropertyName(key, schema)

		// If no schema, use heuristics based on known property names
		var propType string
		var propOptions []string
		if schema != nil {
			if prop, exists := schema.Properties[propName]; exists {
				propType = prop.Type
				propOptions = prop.Options
			}
		}

		// Fallback type inference for common properties when no schema
		if propType == "" {
			switch strings.ToLower(propName) {
			case "assign", "qa engineer", "discuss with", "reviewer", "created by":
				propType = "people"
			case "status", "design status", "priority approved":
				propType = "status"
			case "priority", "reported by":
				propType = "select"
			case "environment", "issue report type":
				propType = "multi_select"
			case "issue title":
				propType = "title"
			case "deploying", "communicated", "silent", "required for deployment":
				propType = "checkbox"
			default:
				propType = "rich_text" // default fallback
			}
		}

		strVal, isStr := value.(string)

		switch propType {
		case "people", "created_by", "last_edited_by":
			resolved := strVal
			if isStr {
				resolved = resolveUserName(strVal, schema)
			}
			filters = append(filters, map[string]any{
				"property": propName,
				"people":   map[string]any{"contains": resolved},
			})

		case "status":
			normalized := normalizeEnumValue(strVal, propOptions)
			filters = append(filters, map[string]any{
				"property": propName,
				"status":   map[string]any{"equals": normalized},
			})

		case "select":
			normalized := normalizeEnumValue(strVal, propOptions)
			filters = append(filters, map[string]any{
				"property": propName,
				"select":   map[string]any{"equals": normalized},
			})

		case "multi_select":
			normalized := normalizeEnumValue(strVal, propOptions)
			filters = append(filters, map[string]any{
				"property":     propName,
				"multi_select": map[string]any{"contains": normalized},
			})

		case "checkbox":
			boolVal, _ := value.(bool)
			filters = append(filters, map[string]any{
				"property": propName,
				"checkbox": map[string]any{"equals": boolVal},
			})

		case "rich_text", "title":
			filters = append(filters, map[string]any{
				"property":  propName,
				"rich_text": map[string]any{"contains": strVal},
			})

		case "number":
			filters = append(filters, map[string]any{
				"property": propName,
				"number":   map[string]any{"equals": value},
			})

		case "unique_id":
			filters = append(filters, map[string]any{
				"property":  propName,
				"unique_id": map[string]any{"equals": value},
			})
		}
	}

	if len(filters) == 0 {
		return data
	}
	if len(filters) == 1 {
		return filters[0]
	}

	// Multiple filters -> AND them
	result := make([]any, len(filters))
	for i, f := range filters {
		result[i] = f
	}
	return map[string]any{"and": result}
}

// normalizeFilter handles full Notion filter syntax with normalization
func normalizeFilter(data map[string]any, schema *DatabaseSchema) map[string]any {
	result := make(map[string]any)

	// Normalize property name
	propName, _ := data["property"].(string)
	normalizedProp := normalizePropertyName(propName, schema)
	result["property"] = normalizedProp

	var prop SchemaProperty
	var exists bool
	if schema != nil {
		prop, exists = schema.Properties[normalizedProp]
	}

	// Copy and normalize filter conditions
	for key, value := range data {
		if key == "property" {
			continue
		}

		switch key {
		case "people", "created_by", "last_edited_by":
			if cond, ok := value.(map[string]any); ok {
				normalized := make(map[string]any)
				for ck, cv := range cond {
					if strVal, ok := cv.(string); ok {
						normalized[ck] = resolveUserName(strVal, schema)
					} else {
						normalized[ck] = cv
					}
				}
				result[key] = normalized
			} else {
				result[key] = value
			}

		case "status", "select":
			if cond, ok := value.(map[string]any); ok && exists {
				normalized := make(map[string]any)
				for ck, cv := range cond {
					if strVal, ok := cv.(string); ok && (ck == "equals" || ck == "does_not_equal") {
						normalized[ck] = normalizeEnumValue(strVal, prop.Options)
					} else {
						normalized[ck] = cv
					}
				}
				result[key] = normalized
			} else {
				result[key] = value
			}

		case "multi_select":
			if cond, ok := value.(map[string]any); ok && exists {
				normalized := make(map[string]any)
				for ck, cv := range cond {
					if strVal, ok := cv.(string); ok && (ck == "contains" || ck == "does_not_contain") {
						normalized[ck] = normalizeEnumValue(strVal, prop.Options)
					} else {
						normalized[ck] = cv
					}
				}
				result[key] = normalized
			} else {
				result[key] = value
			}

		default:
			result[key] = value
		}
	}

	return result
}

// normalizePropertyName finds the correct casing for a property name
func normalizePropertyName(name string, schema *DatabaseSchema) string {
	if schema == nil {
		return name
	}

	// Exact match
	if _, exists := schema.Properties[name]; exists {
		return name
	}

	// Case-insensitive match
	lowerName := strings.ToLower(name)
	for propName := range schema.Properties {
		if strings.ToLower(propName) == lowerName {
			return propName
		}
	}

	// Common aliases
	aliases := map[string]string{
		"assignee":    "Assign",
		"assigned":    "Assign",
		"qa":          "QA Engineer",
		"title":       "Issue Title",
		"name":        "Issue Title",
		"discuss":     "Discuss With",
		"discusswith": "Discuss With",
		"created":     "Created by",
		"createdby":   "Created by",
		"author":      "Created by",
	}

	if mapped, ok := aliases[lowerName]; ok {
		return mapped
	}

	return name
}

// normalizeEnumValue finds the correct casing for enum values
func normalizeEnumValue(value string, options []string) string {
	if len(options) == 0 {
		return value
	}

	// Exact match
	for _, opt := range options {
		if opt == value {
			return value
		}
	}

	// Case-insensitive match
	lowerVal := strings.ToLower(value)
	for _, opt := range options {
		if strings.ToLower(opt) == lowerVal {
			return opt
		}
	}

	// Partial match (prefix)
	for _, opt := range options {
		if strings.HasPrefix(strings.ToLower(opt), lowerVal) {
			return opt
		}
	}

	return value
}

// resolveUserName converts user names/aliases to UUIDs
func resolveUserName(name string, schema *DatabaseSchema) string {
	if isUUID(name) {
		return name
	}

	if schema != nil && schema.Users != nil {
		if id, ok := schema.Users[strings.ToLower(name)]; ok {
			return id
		}
	}

	// Fallback to global map
	if id, ok := userIDMap[strings.ToLower(name)]; ok {
		return id
	}

	return name
}
