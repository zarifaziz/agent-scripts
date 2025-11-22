package core

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"os"
)

//go:embed fields.json
var defaultFieldsJSON []byte

// Config describes how to group well-known keys when presenting logs.
type Config struct {
	OTelKeys     []string `json:"otelKeys"`
	ProviderKeys []string `json:"providerKeys"`
}

// LoadConfig loads configuration from the given path. If path is empty,
// the embedded default configuration is returned.
func LoadConfig(path string) (*Config, error) {
	if path == "" {
		cfg := DefaultConfig()
		return &cfg, nil
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config: %w", err)
	}
	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}
	return &cfg, nil
}

// DefaultConfig returns the embedded grouping configuration.
func DefaultConfig() Config {
	var cfg Config
	if err := json.Unmarshal(defaultFieldsJSON, &cfg); err != nil {
		// The embedded file should always be valid; panic to surface build-time errors.
		panic(fmt.Errorf("invalid embedded config: %w", err))
	}
	return cfg
}
