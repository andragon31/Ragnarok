package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

type Config struct {
	Project         string             `json:"project"`
	Version         string             `json:"version"`
	Port            int                `json:"port"`
	DataDir         string             `json:"data_dir"`
	LogLevel        string             `json:"log_level"`
	AutoInject      bool               `json:"auto_inject_on_session_start"`
	AutoInjectLimit int                `json:"auto_inject_limit"`
	SemanticSearch  bool               `json:"semantic_search_enabled"`
	CacheEnabled    bool               `json:"cache_enabled"`
	CacheTTLHours   int                `json:"cache_ttl_hours"`
	Specs           *SpecsConfig       `json:"specs,omitempty"`
	Bootstrap       *BootstrapConfig   `json:"bootstrap,omitempty"`
	Integrations    *IntegrationConfig `json:"integrations,omitempty"`
}

type SpecsConfig struct {
	CheckOnPlanCreate       bool `json:"check_on_plan_create"`
	ShowInPlanCheckpoint    bool `json:"show_in_plan_checkpoint"`
	GenerateDeltaOnComplete bool `json:"generate_delta_on_complete"`
}

type BootstrapConfig struct {
	SkipOnExisting bool `json:"skip_on_existing"`
}

type IntegrationConfig struct {
	Hati  *HatiIntegration  `json:"hati,omitempty"`
	Skoll *SkollIntegration `json:"skoll,omitempty"`
	Tyr   *TyrIntegration   `json:"tyr,omitempty"`
}

type HatiIntegration struct {
	Enabled             bool `json:"enabled"`
	ContextOnPlanCreate bool `json:"get_context_on_plan_create"`
}

type SkollIntegration struct {
	Enabled          bool `json:"enabled"`
	SkillsOnActivate bool `json:"skills_on_agent_activate"`
}

type TyrIntegration struct {
	Enabled         bool `json:"enabled"`
	FindingsNotify  bool `json:"findings_notify"`
	QualitySnapshot bool `json:"quality_snapshot"`
}

func DefaultConfig() *Config {
	home, _ := os.UserHomeDir()
	return &Config{
		Project:         "unnamed-project",
		Version:         "1.0.0",
		Port:            7438,
		DataDir:         filepath.Join(home, ".fenrir"),
		LogLevel:        "info",
		AutoInject:      true,
		AutoInjectLimit: 5,
		SemanticSearch:  true,
		CacheEnabled:    true,
		CacheTTLHours:   4,
		Specs: &SpecsConfig{
			CheckOnPlanCreate:       true,
			ShowInPlanCheckpoint:    true,
			GenerateDeltaOnComplete: true,
		},
		Bootstrap: &BootstrapConfig{
			SkipOnExisting: false,
		},
	}
}

func LoadConfig(configPath string) (*Config, error) {
	cfg := DefaultConfig()

	if configPath != "" {
		data, err := os.ReadFile(configPath)
		if err != nil {
			if !os.IsNotExist(err) {
				return nil, fmt.Errorf("failed to read config: %w", err)
			}
		} else {
			if err := json.Unmarshal(data, cfg); err != nil {
				return nil, fmt.Errorf("failed to parse config: %w", err)
			}
		}
	}

	if err := os.MkdirAll(cfg.DataDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create data dir: %w", err)
	}

	return cfg, nil
}

func (c *Config) Save(path string) error {
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}
	return os.WriteFile(path, data, 0644)
}
