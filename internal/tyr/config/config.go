package config

import (
	"fmt"
	"os"
	"path/filepath"
)

type Config struct {
	Port             int
	DataDir          string
	SemgrepPath      string
	CacheTTLHours    int
	CVECacheTTLHours int
}

func DefaultConfig() *Config {
	home, _ := os.UserHomeDir()
	dataDir := filepath.Join(home, ".tyr")

	return &Config{
		Port:             7440,
		DataDir:          dataDir,
		SemgrepPath:      "semgrep",
		CacheTTLHours:    1,
		CVECacheTTLHours: 6,
	}
}

func (c *Config) DBPath() string {
	return filepath.Join(c.DataDir, "tyr.db")
}

func LoadConfig(dataDir string) (*Config, error) {
	cfg := DefaultConfig()

	if dataDir != "" {
		cfg.DataDir = dataDir
	}

	if err := os.MkdirAll(cfg.DataDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create data directory: %w", err)
	}

	return cfg, nil
}
