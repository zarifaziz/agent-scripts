package core

import (
	"bufio"
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// helper to run processor and return decoded records
func run(t *testing.T, file string, includeFX bool) []map[string]any {
	t.Helper()
	path := filepath.Join("..", file)
	f, err := os.Open(path)
	if err != nil {
		t.Fatalf("open %s: %v", path, err)
	}
	defer f.Close()

	var buf bytes.Buffer
	p := NewProcessor(DefaultConfig(), Options{IncludeFXLogs: includeFX})
	if err := p.ProcessStream(bufio.NewReader(f), &buf); err != nil {
		t.Fatalf("process: %v", err)
	}

	dec := json.NewDecoder(bytes.NewReader(buf.Bytes()))
	var out []map[string]any
	for dec.More() {
		var m map[string]any
		if err := dec.Decode(&m); err != nil {
			t.Fatalf("decode: %v", err)
		}
		out = append(out, m)
	}
	return out
}

func pathsOfBlocks(recs []map[string]any) []string {
	var paths []string
	for _, r := range recs {
		if r["type"] == "block" {
			if b, ok := r["block"].(map[string]any); ok {
				if p, ok := b["path"].(string); ok {
					paths = append(paths, p)
				}
			}
		}
	}
	return paths
}

func TestFXFiltering(t *testing.T) {
	recs := run(t, "server-logs/example3.json", false)
	if len(recs) != 0 {
		t.Fatalf("expected FX logs to be filtered, got %d records", len(recs))
	}
	recs = run(t, "server-logs/example3.json", true)
	if len(recs) == 0 {
		t.Fatalf("expected records when FX allowed")
	}
}

func TestBlocksExtractionExample1(t *testing.T) {
	recs := run(t, "server-logs/example.json", false)
	got := pathsOfBlocks(recs)
	want := []string{"/error/stacktrace", "/error/messages"}
	if len(got) != len(want) {
		t.Fatalf("block count mismatch got %v want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("block path mismatch idx %d got %s want %s", i, got[i], want[i])
		}
	}
}

func TestProviderGrouping(t *testing.T) {
	recs := run(t, "server-logs/example5.json", false)
	if len(recs) == 0 {
		t.Fatalf("no records returned")
	}
	first := recs[0]
	provider, ok := first["provider"].(map[string]any)
	if !ok {
		t.Fatalf("provider missing")
	}
	if provider["model"] != "gpt-oss-120b" {
		t.Fatalf("model grouping mismatch: %v", provider["model"])
	}
}

func TestInvalidLatexDetailsBecomeBlocks(t *testing.T) {
	recs := run(t, "server-logs/example5.json", false)
	paths := pathsOfBlocks(recs)
	found := false
	for _, p := range paths {
		if p == "/invalidLatexData/0/errorDetail/0" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected invalidLatexData errorDetail to be extracted as block; got paths %v", paths)
	}
}
