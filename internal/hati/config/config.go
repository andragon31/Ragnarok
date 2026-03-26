package config

import (
	"fmt"
	"os"
	"path/filepath"
)

type Config struct {
	Port                  int
	DataDir               string
	FenrirPort            int
	TyrPort               int
	SkollPort             int
	FastApprovalSec       int
	IncludePlanIDInCommit bool
}

func DefaultConfig() *Config {
	home, _ := os.UserHomeDir()
	dataDir := filepath.Join(home, ".hati")

	return &Config{
		Port:                  7439,
		DataDir:               dataDir,
		FenrirPort:            7437,
		TyrPort:               7440,
		SkollPort:             7438,
		FastApprovalSec:       5,
		IncludePlanIDInCommit: true,
	}
}

func (c *Config) DBPath() string {
	return filepath.Join(c.DataDir, "hati.db")
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
