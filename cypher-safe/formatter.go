package main

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
	"reflect"
	"strings"

	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
	"github.com/olekukonko/tablewriter"
)

func printResults(results []map[string]any, outputFormat string) error {
	switch outputFormat {
	case "json":
		return printJSON(results, true)
	case "compact":
		return printJSON(results, false)
	case "csv":
		return printCSV(results)
	case "table":
		return printTable(results)
	default:
		return fmt.Errorf("unknown format: %s", outputFormat)
	}
}

func printJSON(results []map[string]any, indent bool) error {
	var (
		jsonBytes []byte
		err       error
	)

	if indent {
		jsonBytes, err = json.MarshalIndent(results, "", "  ")
	} else {
		jsonBytes, err = json.Marshal(results)
	}
	if err != nil {
		return fmt.Errorf("failed to marshal results: %w", err)
	}
	fmt.Println(string(jsonBytes))
	return nil
}

func printCSV(results []map[string]any) error {
	if len(results) == 0 {
		return nil
	}

	writer := csv.NewWriter(os.Stdout)
	defer writer.Flush()

	headers := make([]string, 0, len(results[0]))
	for k := range results[0] {
		headers = append(headers, k)
	}
	if err := writer.Write(headers); err != nil {
		return fmt.Errorf("failed to write CSV headers: %w", err)
	}

	for _, result := range results {
		row := make([]string, len(headers))
		for i, h := range headers {
			row[i] = formatValue(result[h])
		}
		if err := writer.Write(row); err != nil {
			return fmt.Errorf("failed to write CSV row: %w", err)
		}
	}

	return nil
}

func printTable(results []map[string]any) error {
	if len(results) == 0 {
		return nil
	}

	table := tablewriter.NewWriter(os.Stdout)

	headers := make([]string, 0, len(results[0]))
	for k := range results[0] {
		headers = append(headers, k)
	}
	table.Header(headers)

	for _, result := range results {
		row := make([]string, len(headers))
		for i, h := range headers {
			row[i] = formatValue(result[h])
		}
		if err := table.Append(row); err != nil {
			return fmt.Errorf("failed to append table row: %w", err)
		}
	}

	if err := table.Render(); err != nil {
		return fmt.Errorf("failed to render table: %w", err)
	}
	return nil
}

func formatValue(val any) string {
	if val == nil {
		return ""
	}

	switch v := val.(type) {
	case string:
		return v
	case fmt.Stringer:
		return v.String()
	case neo4j.Node:
		return formatNode(v)
	case *neo4j.Node:
		return formatNode(*v)
	case neo4j.Relationship:
		return formatRelationship(v)
	case *neo4j.Relationship:
		return formatRelationship(*v)
	case neo4j.Path:
		return formatPath(v)
	case *neo4j.Path:
		return formatPath(*v)
	case map[string]any:
		return toPrettyJSON(v)
	case []any:
		return toPrettyJSON(v)
	default:
		valValue := reflect.ValueOf(val)
		if valValue.Kind() == reflect.Map {
			return toPrettyJSON(reflectMapToJSONReady(valValue))
		}
		return fmt.Sprintf("%v", v)
	}
}

func formatNode(node neo4j.Node) string {
	return toPrettyJSON(nodeData(node))
}

func formatRelationship(rel neo4j.Relationship) string {
	return toPrettyJSON(relationshipData(rel))
}

func formatPath(path neo4j.Path) string {
	return toPrettyJSON(pathData(path))
}

func nodeData(node neo4j.Node) map[string]any {
	return map[string]any{
		"id":         node.Id,
		"element_id": node.ElementId,
		"labels":     node.Labels,
		"properties": normalizeMap(node.Props),
	}
}

func relationshipData(rel neo4j.Relationship) map[string]any {
	return map[string]any{
		"id":            rel.Id,
		"element_id":    rel.ElementId,
		"type":          rel.Type,
		"start_id":      rel.StartId,
		"end_id":        rel.EndId,
		"start_element": rel.StartElementId,
		"end_element":   rel.EndElementId,
		"properties":    normalizeMap(rel.Props),
	}
}

func pathData(path neo4j.Path) map[string]any {
	nodes := make([]map[string]any, len(path.Nodes))
	for i, node := range path.Nodes {
		nodes[i] = nodeData(node)
	}

	rels := make([]map[string]any, len(path.Relationships))
	for i, rel := range path.Relationships {
		rels[i] = relationshipData(rel)
	}

	return map[string]any{
		"nodes":         nodes,
		"relationships": rels,
	}
}

func normalizeMap(input map[string]any) map[string]any {
	normalized := make(map[string]any, len(input))
	for k, v := range input {
		normalized[k] = normalizeValue(v)
	}
	return normalized
}

func normalizeSlice(values []any) []any {
	out := make([]any, len(values))
	for i, v := range values {
		out[i] = normalizeValue(v)
	}
	return out
}

func reflectMapToJSONReady(val reflect.Value) map[string]any {
	result := make(map[string]any, val.Len())
	iter := val.MapRange()
	for iter.Next() {
		key := fmt.Sprint(iter.Key().Interface())
		result[key] = normalizeValue(iter.Value().Interface())
	}
	return result
}

func normalizeValue(val any) any {
	switch v := val.(type) {
	case map[string]any:
		return normalizeMap(v)
	case []any:
		return normalizeSlice(v)
	case neo4j.Node:
		return nodeData(v)
	case *neo4j.Node:
		return nodeData(*v)
	case neo4j.Relationship:
		return relationshipData(v)
	case *neo4j.Relationship:
		return relationshipData(*v)
	case neo4j.Path:
		return pathData(v)
	case *neo4j.Path:
		return pathData(*v)
	default:
		rv := reflect.ValueOf(val)
		if rv.IsValid() && rv.Kind() == reflect.Map {
			return reflectMapToJSONReady(rv)
		}
		return v
	}
}

func toPrettyJSON(val any) string {
	normalized := normalizeValue(val)
	jsonBytes, err := json.MarshalIndent(normalized, "", "  ")
	if err != nil {
		return fmt.Sprintf("%v", val)
	}
	return string(jsonBytes)
}

func printExecutionPlan(plan any, outputFormat string) error {
	switch outputFormat {
	case "json", "compact":
		return printPlanJSON(plan, outputFormat == "json")
	case "table", "csv":
		return printPlanTree(plan)
	default:
		return printPlanTree(plan)
	}
}

func printPlanJSON(plan any, indent bool) error {
	planData := planToMap(plan)
	var jsonBytes []byte
	var err error

	if indent {
		jsonBytes, err = json.MarshalIndent(planData, "", "  ")
	} else {
		jsonBytes, err = json.Marshal(planData)
	}

	if err != nil {
		return fmt.Errorf("failed to marshal execution plan: %w", err)
	}

	fmt.Println(string(jsonBytes))
	return nil
}

func planToMap(plan any) map[string]any {
	switch p := plan.(type) {
	case neo4j.Plan:
		result := map[string]any{
			"operator":  p.Operator(),
			"arguments": p.Arguments(),
		}

		if p.Identifiers() != nil && len(p.Identifiers()) > 0 {
			result["identifiers"] = p.Identifiers()
		}

		children := p.Children()
		if len(children) > 0 {
			childMaps := make([]map[string]any, len(children))
			for i, child := range children {
				childMaps[i] = planToMap(child)
			}
			result["children"] = childMaps
		}

		return result

	case neo4j.ProfiledPlan:
		result := map[string]any{
			"operator":  p.Operator(),
			"arguments": p.Arguments(),
		}

		if p.Identifiers() != nil && len(p.Identifiers()) > 0 {
			result["identifiers"] = p.Identifiers()
		}

		result["dbHits"] = p.DbHits()
		result["records"] = p.Records()
		result["pageCacheHits"] = p.PageCacheHits()
		result["pageCacheMisses"] = p.PageCacheMisses()
		result["pageCacheHitRatio"] = p.PageCacheHitRatio()
		result["time"] = p.Time()

		children := p.Children()
		if len(children) > 0 {
			childMaps := make([]map[string]any, len(children))
			for i, child := range children {
				childMaps[i] = planToMap(child)
			}
			result["children"] = childMaps
		}

		return result

	default:
		return map[string]any{"error": "unknown plan type"}
	}
}

func printPlanTree(plan any) error {
	fmt.Println("Execution Plan:")
	fmt.Println(strings.Repeat("=", 80))
	printPlanNode(plan, "", true, true)
	return nil
}

func printPlanNode(plan any, prefix string, isLast bool, isRoot bool) {
	var connector string
	if isRoot {
		connector = ""
	} else if isLast {
		connector = "└── "
	} else {
		connector = "├── "
	}

	var operator string
	var args map[string]any
	var childPrefix string

	if isRoot {
		childPrefix = "    "
	} else if isLast {
		childPrefix = prefix + "    "
	} else {
		childPrefix = prefix + "│   "
	}

	switch p := plan.(type) {
	case neo4j.Plan:
		operator = p.Operator()
		args = p.Arguments()
		fmt.Printf("%s%s%s\n", prefix, connector, operator)

		if len(args) > 0 {
			for key, value := range args {
				valueStr := formatPlanValue(value)
				fmt.Printf("%s• %s: %s\n", childPrefix, key, valueStr)
			}
		}

		children := p.Children()
		for i, child := range children {
			var nextPrefix string
			if isRoot {
				nextPrefix = ""
			} else if isLast {
				nextPrefix = prefix + "    "
			} else {
				nextPrefix = prefix + "│   "
			}
			printPlanNode(child, nextPrefix, i == len(children)-1, false)
		}

	case neo4j.ProfiledPlan:
		operator = p.Operator()
		args = p.Arguments()
		
		dbHits := p.DbHits()
		records := p.Records()
		time := p.Time()
		
		fmt.Printf("%s%s%s (rows=%d, dbHits=%d, time=%dµs)\n", prefix, connector, operator, records, dbHits, time)

		if len(args) > 0 {
			for key, value := range args {
				valueStr := formatPlanValue(value)
				fmt.Printf("%s• %s: %s\n", childPrefix, key, valueStr)
			}
		}

		pageCacheHits := p.PageCacheHits()
		pageCacheMisses := p.PageCacheMisses()
		if pageCacheHits > 0 || pageCacheMisses > 0 {
			fmt.Printf("%s• Page Cache: %d hits, %d misses (%.2f%% hit ratio)\n", 
				childPrefix, pageCacheHits, pageCacheMisses, p.PageCacheHitRatio()*100)
		}

		children := p.Children()
		for i, child := range children {
			var nextPrefix string
			if isRoot {
				nextPrefix = ""
			} else if isLast {
				nextPrefix = prefix + "    "
			} else {
				nextPrefix = prefix + "│   "
			}
			printPlanNode(child, nextPrefix, i == len(children)-1, false)
		}

	default:
		fmt.Printf("%s%sUnknown plan type\n", prefix, connector)
	}
}

func formatPlanValue(value any) string {
	switch v := value.(type) {
	case string:
		return v
	case []any:
		if len(v) == 0 {
			return "[]"
		}
		parts := make([]string, len(v))
		for i, item := range v {
			parts[i] = formatPlanValue(item)
		}
		return "[" + strings.Join(parts, ", ") + "]"
	case map[string]any:
		if len(v) == 0 {
			return "{}"
		}
		jsonBytes, err := json.Marshal(v)
		if err != nil {
			return fmt.Sprintf("%v", v)
		}
		return string(jsonBytes)
	default:
		return fmt.Sprintf("%v", v)
	}
}
