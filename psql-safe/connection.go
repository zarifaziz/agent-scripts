package main

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type ConnConfig struct {
	Host     string
	Port     string
	User     string
	Database string
	Password string
	SSLMode  string
	CertDir  string
}

type PresetConfig struct {
	Host     string `json:"host"`
	Port     int    `json:"port"`
	User     string `json:"user"`
	Database string `json:"database"`
	Password string `json:"password"`
	SSLMode  string `json:"ssl_mode"`
	CertDir  string `json:"cert_dir"`
}

type PresetsFile struct {
	Presets map[string]PresetConfig `json:"presets"`
}

func getPresetsPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	cacheDir := filepath.Join(home, ".cache", "psql-safe")
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

func buildConnectionString(cfg ConnConfig) (string, error) {
	// Build the connection URL
	connStr := fmt.Sprintf("postgres://%s:%s@%s:%s/%s",
		cfg.User,
		cfg.Password,
		cfg.Host,
		cfg.Port,
		cfg.Database,
	)

	// Add SSL mode
	connStr += "?sslmode=" + cfg.SSLMode

	return connStr, nil
}

func createPool(ctx context.Context, cfg ConnConfig, timeoutSec int) (*pgxpool.Pool, error) {
	connStr, err := buildConnectionString(cfg)
	if err != nil {
		return nil, err
	}

	poolConfig, err := pgxpool.ParseConfig(connStr)
	if err != nil {
		return nil, fmt.Errorf("failed to parse connection string: %w", err)
	}

	// Set max connections to 1 for predictable transaction behavior
	poolConfig.MaxConns = 1

	// Configure TLS if needed
	if cfg.SSLMode == "verify-full" && cfg.CertDir != "" {
		tlsConfig, err := buildTLSConfig(cfg.CertDir, cfg.User, cfg.Host)
		if err != nil {
			return nil, fmt.Errorf("failed to build TLS config: %w", err)
		}
		poolConfig.ConnConfig.TLSConfig = tlsConfig
	} else if cfg.SSLMode == "disable" {
		fmt.Fprintf(os.Stderr, "Warning: SSL is disabled. Only use this for local development.\n")
	}

	// Apply timeout to pool creation and ping
	timeoutCtx, cancel := context.WithTimeout(ctx, time.Duration(timeoutSec)*time.Second)
	defer cancel()

	pool, err := pgxpool.NewWithConfig(timeoutCtx, poolConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create connection pool: %w", err)
	}

	// Test the connection with timeout
	if err := pool.Ping(timeoutCtx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	return pool, nil
}

func buildTLSConfig(certDir, user, host string) (*tls.Config, error) {
	// Load CA certificate
	caCertPath := filepath.Join(certDir, "ca.crt")
	caCert, err := os.ReadFile(caCertPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read CA certificate at %s: %w", caCertPath, err)
	}

	caCertPool := x509.NewCertPool()
	if !caCertPool.AppendCertsFromPEM(caCert) {
		return nil, fmt.Errorf("failed to parse CA certificate")
	}

	// Load client certificate and key
	clientCertPath := filepath.Join(certDir, fmt.Sprintf("client.%s.crt", user))
	clientKeyPath := filepath.Join(certDir, fmt.Sprintf("client.%s.key", user))

	clientCert, err := tls.LoadX509KeyPair(clientCertPath, clientKeyPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load client certificate from %s: %w", certDir, err)
	}

	return &tls.Config{
		ServerName:   host,
		RootCAs:      caCertPool,
		Certificates: []tls.Certificate{clientCert},
		MinVersion:   tls.VersionTLS12,
	}, nil
}

func expandPath(path string) string {
	if len(path) > 0 && path[0] == '~' {
		home, err := os.UserHomeDir()
		if err == nil {
			path = filepath.Join(home, path[1:])
		}
	}
	return path
}
