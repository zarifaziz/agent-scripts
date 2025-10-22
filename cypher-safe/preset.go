package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

type PresetConfig struct {
	URI      string `json:"uri"`
	Username string `json:"username"`
	Password string `json:"password"`
	Database string `json:"database"`
}

type PresetsFile struct {
	Presets map[string]PresetConfig `json:"presets"`
}

func getPresetsPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	cacheDir := filepath.Join(home, ".cache", "scripts", "cypher-query")
	if err := os.MkdirAll(cacheDir, 0o755); err != nil {
		return "", err
	}
	return filepath.Join(cacheDir, "presets.json"), nil
}

func loadPresets() (*PresetsFile, error) {
	presetsPath, err := getPresetsPath()
	if err != nil {
		return nil, err
	}

	if _, err := os.Stat(presetsPath); os.IsNotExist(err) {
		return &PresetsFile{Presets: make(map[string]PresetConfig)}, nil
	}

	data, err := os.ReadFile(presetsPath)
	if err != nil {
		return nil, err
	}

	var presets PresetsFile
	if err := json.Unmarshal(data, &presets); err != nil {
		return nil, err
	}

	if presets.Presets == nil {
		presets.Presets = make(map[string]PresetConfig)
	}

	return &presets, nil
}

func savePresets(presets *PresetsFile) error {
	presetsPath, err := getPresetsPath()
	if err != nil {
		return err
	}

	data, err := json.MarshalIndent(presets, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(presetsPath, data, 0o644)
}

func getPreset(name string) (*PresetConfig, error) {
	presets, err := loadPresets()
	if err != nil {
		return nil, err
	}

	preset, exists := presets.Presets[name]
	if !exists {
		return nil, fmt.Errorf("preset '%s' not found", name)
	}

	return &preset, nil
}

func ensureDefaultPresets() error {
	presets, err := loadPresets()
	if err != nil {
		return err
	}

	if _, exists := presets.Presets["resources-dev"]; !exists {
		presets.Presets["resources-dev"] = PresetConfig{
			URI:      "neo4j://resources-neo4j-db-lb-neo4j.learning.svc.cluster.local:7687",
			Username: "neo4j",
			Password: "De0YFd4XG239RCoP",
			Database: "neo4j",
		}
		if err := savePresets(presets); err != nil {
			return err
		}
	}

	return nil
}
