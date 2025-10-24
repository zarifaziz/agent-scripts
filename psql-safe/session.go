package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type QueryEntry struct {
	SQL    string         `json:"sql"`
	Params map[string]any `json:"params,omitempty"`
}

type TxOptions struct {
	IsoLevel string `json:"iso_level"`
	ReadOnly bool   `json:"read_only"`
}

type SessionData struct {
	Name          string         `json:"name"`
	Database      string         `json:"database"`
	Queries       []QueryEntry   `json:"queries"`
	CreatedAt     time.Time      `json:"created_at"`
	DefaultFormat string         `json:"default_format,omitempty"`
	TxOptions     TxOptions      `json:"tx_options"`
}

func getSessionDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	sessionDir := filepath.Join(home, ".cache", "scripts", "sql-query", "sessions")
	if err := os.MkdirAll(sessionDir, 0o755); err != nil {
		return "", err
	}
	return sessionDir, nil
}

func getSessionPath(sessionName string) (string, error) {
	sessionDir, err := getSessionDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(sessionDir, sessionName+".json"), nil
}

func loadSession(sessionPath string) (*SessionData, error) {
	data, err := os.ReadFile(sessionPath)
	if err != nil {
		return nil, err
	}
	var session SessionData
	if err := json.Unmarshal(data, &session); err != nil {
		return nil, err
	}
	return &session, nil
}

func saveSession(sessionPath string, session *SessionData) error {
	data, err := json.MarshalIndent(session, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(sessionPath, data, 0o644)
}

func sessionExists(sessionName string) (bool, error) {
	sessionPath, err := getSessionPath(sessionName)
	if err != nil {
		return false, err
	}
	_, err = os.Stat(sessionPath)
	if os.IsNotExist(err) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

func deleteSession(sessionName string) error {
	sessionPath, err := getSessionPath(sessionName)
	if err != nil {
		return err
	}
	return os.Remove(sessionPath)
}

func getSessions() ([]SessionData, error) {
	sessionDir, err := getSessionDir()
	if err != nil {
		return nil, err
	}

	entries, err := os.ReadDir(sessionDir)
	if err != nil {
		return nil, err
	}

	var sessions []SessionData
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}
		sessionPath := filepath.Join(sessionDir, entry.Name())
		session, err := loadSession(sessionPath)
		if err != nil {
			continue
		}
		sessions = append(sessions, *session)
	}
	return sessions, nil
}

func cleanStaleSessionsIfNeeded(maxAgeDays int) error {
	sessionDir, err := getSessionDir()
	if err != nil {
		return err
	}

	entries, err := os.ReadDir(sessionDir)
	if err != nil {
		return err
	}

	cutoffTime := time.Now().AddDate(0, 0, -maxAgeDays)
	var removed int

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}
		sessionPath := filepath.Join(sessionDir, entry.Name())
		session, err := loadSession(sessionPath)
		if err != nil {
			continue
		}
		if session.CreatedAt.Before(cutoffTime) {
			if err := os.Remove(sessionPath); err == nil {
				removed++
			}
		}
	}

	if removed > 0 {
		fmt.Fprintf(os.Stderr, "Cleaned up %d stale session(s) older than %d days\n", removed, maxAgeDays)
	}

	return nil
}
