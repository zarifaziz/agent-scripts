package core

import (
	"bufio"
	"encoding/json"
	"errors"
	"io"
	"regexp"
	"strconv"
	"strings"
)

// Options control how raw log lines are transformed.
type Options struct {
	IncludeFXLogs      bool
	MinMultilineLines  int
	MaxScannerBufBytes int // optional override; defaults to 1MiB if zero
}

// OutputRecord represents one item ready for UI/CLI consumption.
// Type "log" contains structured data; "text" is passthrough.
type OutputRecord struct {
	Type     string         `json:"type"`               // "log", "block", or "text"
	Index    int            `json:"index"`              // stable per input line
	Text     string         `json:"text,omitempty"`     // for Type == "text"
	Summary  *Summary       `json:"summary,omitempty"`  // for Type == "log"
	OTel     map[string]any `json:"otel,omitempty"`     // grouped system fields
	Provider map[string]any `json:"provider,omitempty"` // grouped provider fields
	Data     map[string]any `json:"data,omitempty"`     // remaining payload fields
	JSON     map[string]any `json:"json,omitempty"`     // cleaned json without extracted blocks
	Block    *Block         `json:"block,omitempty"`    // for Type == "block"
}

// Summary captures the small preview shown in tables.
type Summary struct {
	Time     string `json:"time,omitempty"`
	Severity string `json:"severity,omitempty"`
	Message  string `json:"message,omitempty"`
	Caller   string `json:"caller,omitempty"`
}

// Block holds multiline content extracted from the log object.
type Block struct {
	Path  string `json:"path"`
	Value string `json:"value"`
}

var fxLine = regexp.MustCompile(`^\[F[0-9x]\]`)

// Processor turns raw log streams into structured records.
type Processor struct {
	cfg Config
	opt Options
}

// NewProcessor builds a Processor with defaults applied.
func NewProcessor(cfg Config, opt Options) *Processor {
	if opt.MinMultilineLines == 0 {
		opt.MinMultilineLines = 4
	}
	if opt.MaxScannerBufBytes == 0 {
		opt.MaxScannerBufBytes = 1024 * 1024
	}
	return &Processor{cfg: cfg, opt: opt}
}

// ProcessStream reads lines from r and writes NDJSON records to w.
func (p *Processor) ProcessStream(r io.Reader, w io.Writer) error {
	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 0, p.opt.MaxScannerBufBytes), p.opt.MaxScannerBufBytes)

	enc := json.NewEncoder(w)
	index := 0

	for scanner.Scan() {
		line := strings.TrimRight(scanner.Text(), "\r\n")
		if !p.opt.IncludeFXLogs && fxLine.MatchString(line) {
			continue
		}

		records, err := p.processLine(line, index)
		if err != nil {
			return err
		}
		for _, rec := range records {
			if err := enc.Encode(rec); err != nil {
				return err
			}
		}
		index++
	}

	if err := scanner.Err(); err != nil {
		return err
	}
	return nil
}

func (p *Processor) processLine(line string, idx int) ([]OutputRecord, error) {
	var jsonObj any
	if err := json.Unmarshal([]byte(line), &jsonObj); err != nil {
		return []OutputRecord{
			{Type: "text", Index: idx, Text: line},
		}, nil
	}

	decoded := decodeNestedJSON(jsonObj)

	blocks := make([]Block, 0)
	cleaned := stripMultiline(decoded, p.opt.MinMultilineLines, &blocks, nil)

	cleaned = extractErrorBlocks(cleaned, &blocks)

	objMap, ok := cleaned.(map[string]any)
	if !ok {
		return nil, errors.New("log JSON root must be object")
	}

	summary := buildSummary(objMap)
	otel, provider, remaining := groupFields(objMap, p.cfg)

	rec := OutputRecord{
		Type:     "log",
		Index:    idx,
		Summary:  &summary,
		OTel:     otel,
		Provider: provider,
		Data:     remaining,
		JSON:     objMap,
	}

	results := []OutputRecord{rec}
	for _, b := range blocks {
		bc := b // copy
		results = append(results, OutputRecord{
			Type:  "block",
			Index: idx,
			Block: &bc,
		})
	}

	return results, nil
}

func buildSummary(m map[string]any) Summary {
	return Summary{
		Time:     getString(m, "time"),
		Severity: firstNonEmpty(getString(m, "severity"), getString(m, "level")),
		Message:  getString(m, "message"),
		Caller:   getString(m, "caller"),
	}
}

func groupFields(m map[string]any, cfg Config) (map[string]any, map[string]any, map[string]any) {
	system := map[string]any{}
	provider := map[string]any{}
	remaining := cloneMap(m)

	for _, k := range cfg.OTelKeys {
		if v, ok := remaining[k]; ok {
			system[k] = v
			delete(remaining, k)
		}
	}

	for _, k := range cfg.ProviderKeys {
		if v, ok := remaining[k]; ok {
			provider[k] = v
			delete(remaining, k)
		}
	}

	return system, provider, remaining
}

func cloneMap(src map[string]any) map[string]any {
	dst := make(map[string]any, len(src))
	for k, v := range src {
		dst[k] = v
	}
	return dst
}

func getString(m map[string]any, key string) string {
	if val, ok := m[key]; ok {
		if s, ok := val.(string); ok {
			return s
		}
	}
	return ""
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if v != "" {
			return v
		}
	}
	return ""
}

func decodeNestedJSON(v any) any {
	switch t := v.(type) {
	case string:
		trimmed := strings.TrimSpace(t)
		if len(trimmed) == 0 {
			return t
		}
		if (strings.HasPrefix(trimmed, "{") && strings.HasSuffix(trimmed, "}")) ||
			(strings.HasPrefix(trimmed, "[") && strings.HasSuffix(trimmed, "]")) {
			var decoded any
			if err := json.Unmarshal([]byte(trimmed), &decoded); err == nil {
				return decodeNestedJSON(decoded)
			}
		}
		return t
	case []any:
		for i, item := range t {
			t[i] = decodeNestedJSON(item)
		}
		return t
	case map[string]any:
		for k, item := range t {
			t[k] = decodeNestedJSON(item)
		}
		return t
	default:
		return v
	}
}

func stripMultiline(v any, minLines int, blocks *[]Block, path []string) any {
	switch t := v.(type) {
	case string:
		lines := strings.Count(t, "\n") + 1
		if lines >= minLines {
			*blocks = append(*blocks, Block{Path: buildPath(path), Value: t})
			return nil
		}
		return t
	case []any:
		result := make([]any, 0, len(t))
		for idx, item := range t {
			segment := append(path, intToStr(idx))
			cleaned := stripMultiline(item, minLines, blocks, segment)
			if cleaned != nil {
				result = append(result, cleaned)
			}
		}
		return result
	case map[string]any:
		copy := make(map[string]any, len(t))
		for k, item := range t {
			segment := append(path, k)
			cleaned := stripMultiline(item, minLines, blocks, segment)
			if cleaned != nil {
				copy[k] = cleaned
			}
		}
		return copy
	default:
		return v
	}
}

func intToStr(i int) string {
	return strconv.Itoa(i)
}

func buildPath(parts []string) string {
	if len(parts) == 0 {
		return "/"
	}
	return "/" + strings.Join(parts, "/")
}

func indentLines(s string) string {
	lines := strings.Split(s, "\n")
	for i, l := range lines {
		lines[i] = "  " + l
	}
	return strings.Join(lines, "\n")
}

func extractErrorBlocks(v any, blocks *[]Block) any {
	obj, ok := v.(map[string]any)
	if !ok {
		return v
	}
	errVal, ok := obj["error"]
	if !ok {
		return obj
	}
	errMap, ok := errVal.(map[string]any)
	if !ok {
		return obj
	}

	if stack, ok := errMap["stacktrace"]; ok {
		if stackStr, ok := stack.(string); ok {
			*blocks = append(*blocks, Block{
				Path:  "/error/stacktrace",
				Value: stackStr,
			})
			delete(errMap, "stacktrace")
		}
	}

	if msgs, ok := errMap["messages"]; ok {
		switch mt := msgs.(type) {
		case string:
			*blocks = append(*blocks, Block{
				Path:  "/error/messages",
				Value: indentLines(mt),
			})
			delete(errMap, "messages")
		case []any:
			parts := make([]string, 0, len(mt))
			for _, v := range mt {
				if s, ok := v.(string); ok {
					parts = append(parts, "  "+s)
				}
			}
			if len(parts) > 0 {
				*blocks = append(*blocks, Block{
					Path:  "/error/messages",
					Value: strings.Join(parts, "\n"),
				})
				delete(errMap, "messages")
			}
		}
	}

	if len(errMap) == 0 {
		delete(obj, "error")
	}
	return obj
}
