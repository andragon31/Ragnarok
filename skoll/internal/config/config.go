package config

import (
	"fmt"
	"os"
	"path/filepath"
)

type Config struct {
	Port      int
	DataDir   string
	SkillsDir string
}

func DefaultConfig() *Config {
	home, _ := os.UserHomeDir()
	dataDir := filepath.Join(home, ".skoll")

	return &Config{
		Port:      7438,
		DataDir:   dataDir,
		SkillsDir: filepath.Join(dataDir, "skills"),
	}
}

func (c *Config) DBPath() string {
	return filepath.Join(c.DataDir, "skoll.db")
}

func LoadConfig(dataDir string) (*Config, error) {
	cfg := DefaultConfig()

	if dataDir != "" {
		cfg.DataDir = dataDir
		cfg.SkillsDir = filepath.Join(dataDir, "skills")
	}

	if err := os.MkdirAll(cfg.DataDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create data directory: %w", err)
	}

	return cfg, nil
}
