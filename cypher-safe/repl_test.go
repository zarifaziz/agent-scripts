package main

import (
	"bytes"
	"context"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestShouldExecuteBuffer(t *testing.T) {
	if shouldExecuteBuffer([]string{"MATCH (n) RETURN n"}) {
		t.Fatalf("expected false when no semicolon present")
	}

	if !shouldExecuteBuffer([]string{"MATCH (n) RETURN n;"}) {
		t.Fatalf("expected true when final line ends with semicolon")
	}

	if !shouldExecuteBuffer([]string{"MATCH (n)", "RETURN n ;   "}) {
		t.Fatalf("expected true when last non-empty line ends with semicolon (ignoring whitespace)")
	}
}

func TestHandleReplParamsCommands(t *testing.T) {
	cfg := replConfig{
		Params: map[string]any{"existing": "value"},
	}

	prevStdout := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe: %v", err)
	}
	os.Stdout = w

	action := handleReplCommand(context.Background(), nil, ":params {\"foo\": \"bar\"}", &cfg)
	if !action.handled {
		t.Fatalf("expected handled to be true")
	}
	if cfg.Params["foo"] != "bar" {
		t.Fatalf("expected params to include foo=bar, got %#v", cfg.Params)
	}
	if _, ok := cfg.Params["existing"]; ok {
		t.Fatalf("expected previous params to be replaced")
	}

	action = handleReplCommand(context.Background(), nil, ":params clear", &cfg)
	if !action.handled {
		t.Fatalf("expected handled to be true for clear")
	}
	if len(cfg.Params) != 0 {
		t.Fatalf("expected params to be cleared, found %#v", cfg.Params)
	}

	_ = w.Close()
	os.Stdout = prevStdout
	_, _ = io.ReadAll(r)
}

func TestPrintSessionDetailsAndCleanup(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)

	sessionDir := filepath.Join(tmp, ".cache", "scripts", "cypher-query", "sessions")
	if err := os.MkdirAll(sessionDir, 0o755); err != nil {
		t.Fatalf("failed to create session dir: %v", err)
	}

	stale := &SessionData{
		Name:      "stale",
		Database:  "neo4j",
		Queries:   []QueryWithParams{{Query: "RETURN 1"}},
		CreatedAt: time.Now().AddDate(0, 0, -90),
	}
	active := &SessionData{
		Name:      "active",
		Database:  "neo4j",
		Queries:   []QueryWithParams{{Query: "RETURN 2"}},
		CreatedAt: time.Now(),
	}

	stalePath, err := getSessionPath(stale.Name)
	if err != nil {
		t.Fatalf("getSessionPath stale: %v", err)
	}
	if err := saveSession(stalePath, stale); err != nil {
		t.Fatalf("saveSession stale: %v", err)
	}

	activePath, err := getSessionPath(active.Name)
	if err != nil {
		t.Fatalf("getSessionPath active: %v", err)
	}
	if err := saveSession(activePath, active); err != nil {
		t.Fatalf("saveSession active: %v", err)
	}

	if err := printSessionDetails(active.Name); err != nil {
		t.Fatalf("printSessionDetails: %v", err)
	}

	if err := cleanStaleSessionsIfNeeded(30); err != nil {
		t.Fatalf("cleanStaleSessionsIfNeeded: %v", err)
	}

	if _, err := os.Stat(activePath); err != nil {
		t.Fatalf("expected active session to remain: %v", err)
	}
	if _, err := os.Stat(stalePath); err == nil {
		t.Fatalf("expected stale session to be removed")
	}
}

func TestHandleParamsShowsCurrent(t *testing.T) {
	cfg := replConfig{
		Params: map[string]any{"foo": "bar"},
	}

	var buf bytes.Buffer
	prevStdout := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("Pipe: %v", err)
	}
	os.Stdout = w

	done := make(chan struct{})
	go func() {
		_, _ = io.Copy(&buf, r)
		close(done)
	}()

	_ = handleReplCommand(context.Background(), nil, ":params", &cfg)

	_ = w.Close()
	os.Stdout = prevStdout
	<-done

	out := strings.TrimSpace(buf.String())
	if !strings.Contains(out, `"foo": "bar"`) {
		t.Fatalf("expected params output to contain foo:bar, got %q", out)
	}
}
