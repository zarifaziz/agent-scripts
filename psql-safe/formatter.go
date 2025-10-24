package main

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
	"reflect"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
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
	// Ensure empty results show as [] not null
	if results == nil {
		results = []map[string]any{}
	}

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

	// Preserve column order from first row
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

	// Preserve column order from first row
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
		table.Append(row)
	}

	table.Render()
	return nil
}

func formatValue(val any) string {
	if val == nil {
		return ""
	}

	switch v := val.(type) {
	case string:
		return v
	case []byte:
		return string(v)
	case map[string]any:
		return toPrettyJSON(v)
	case []any:
		return toPrettyJSON(v)
	default:
		return fmt.Sprintf("%v", v)
	}
}

func toPrettyJSON(val any) string {
	jsonBytes, err := json.MarshalIndent(val, "", "  ")
	if err != nil {
		return fmt.Sprintf("%v", val)
	}
	return string(jsonBytes)
}

// rowsToMaps converts pgx.Rows to a slice of maps
func rowsToMaps(rows pgx.Rows) ([]map[string]any, error) {
	var results []map[string]any

	fieldDescs := rows.FieldDescriptions()
	for rows.Next() {
		values, err := rows.Values()
		if err != nil {
			return nil, fmt.Errorf("failed to get row values: %w", err)
		}

		row := make(map[string]any, len(fieldDescs))
		for i, desc := range fieldDescs {
			row[string(desc.Name)] = normalizeValue(values[i])
		}
		results = append(results, row)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating rows: %w", err)
	}

	return results, nil
}

// normalizeValue converts PostgreSQL-specific types to JSON-friendly types
func normalizeValue(val any) any {
	if val == nil {
		return nil
	}

	// Handle pgtype types
	switch v := val.(type) {
	case pgtype.Numeric:
		if !v.Valid {
			return nil
		}
		// Try converting to float64
		f64, err := v.Float64Value()
		if err == nil && f64.Valid {
			return f64.Float64
		}
		// Fallback to int64
		i64, err := v.Int64Value()
		if err == nil && i64.Valid {
			return i64.Int64
		}
		// Last resort: return as is
		return v
	case pgtype.Int8:
		if !v.Valid {
			return nil
		}
		return v.Int64
	case pgtype.Int4:
		if !v.Valid {
			return nil
		}
		return v.Int32
	case pgtype.Int2:
		if !v.Valid {
			return nil
		}
		return v.Int16
	case pgtype.Float8:
		if !v.Valid {
			return nil
		}
		return v.Float64
	case pgtype.Float4:
		if !v.Valid {
			return nil
		}
		return v.Float32
	case pgtype.Bool:
		if !v.Valid {
			return nil
		}
		return v.Bool
	case pgtype.Text:
		if !v.Valid {
			return nil
		}
		return v.String
	case pgtype.Timestamp:
		if !v.Valid {
			return nil
		}
		return v.Time.Format(time.RFC3339)
	case pgtype.Timestamptz:
		if !v.Valid {
			return nil
		}
		return v.Time.Format(time.RFC3339)
	case pgtype.Date:
		if !v.Valid {
			return nil
		}
		return v.Time.Format("2006-01-02")
	case pgtype.UUID:
		if !v.Valid {
			return nil
		}
		return fmt.Sprintf("%x-%x-%x-%x-%x",
			v.Bytes[0:4], v.Bytes[4:6], v.Bytes[6:8], v.Bytes[8:10], v.Bytes[10:16])
	case time.Time:
		return v.Format(time.RFC3339)
	case []byte:
		return string(v)
	case map[string]any:
		normalized := make(map[string]any, len(v))
		for k, val := range v {
			normalized[k] = normalizeValue(val)
		}
		return normalized
	case []any:
		normalized := make([]any, len(v))
		for i, val := range v {
			normalized[i] = normalizeValue(val)
		}
		return normalized
	default:
		// Handle other types through reflection
		rv := reflect.ValueOf(val)
		if rv.IsValid() && rv.Kind() == reflect.Map {
			result := make(map[string]any)
			iter := rv.MapRange()
			for iter.Next() {
				key := fmt.Sprint(iter.Key().Interface())
				result[key] = normalizeValue(iter.Value().Interface())
			}
			return result
		}
		return v
	}
}
