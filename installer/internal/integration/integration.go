package integration

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

type BootstrapData struct {
	Skills    []map[string]string `json:"suggested_skills"`
	Rules     []map[string]string `json:"rules"`
	Standards []map[string]string `json:"standards"`
}

func LoadBootstrapData(projectPath string) (*BootstrapData, error) {
	ragnarokDir := filepath.Join(projectPath, ".ragnarok")

	data := &BootstrapData{}

	skillsFile := filepath.Join(ragnarokDir, "skills.json")
	if f, err := os.ReadFile(skillsFile); err == nil {
		var parsed map[string]interface{}
		if err := json.Unmarshal(f, &parsed); err == nil {
			if skills, ok := parsed["suggested_skills"].([]interface{}); ok {
				for _, s := range skills {
					if skill, ok := s.(map[string]interface{}); ok {
						m := make(map[string]string)
						for k, v := range skill {
							if vs, ok := v.(string); ok {
								m[k] = vs
							}
						}
						data.Skills = append(data.Skills, m)
					}
				}
			}
		}
	}

	rulesFile := filepath.Join(ragnarokDir, "rules.json")
	if f, err := os.ReadFile(rulesFile); err == nil {
		if err := json.Unmarshal(f, &data.Rules); err != nil {
			return nil, fmt.Errorf("failed to parse rules.json: %w", err)
		}
	}

	standardsFile := filepath.Join(ragnarokDir, "standards.json")
	if f, err := os.ReadFile(standardsFile); err == nil {
		if err := json.Unmarshal(f, &data.Standards); err != nil {
			return nil, fmt.Errorf("failed to parse standards.json: %w", err)
		}
	}

	return data, nil
}

func (d *BootstrapData) HasData() bool {
	return len(d.Skills) > 0 || len(d.Rules) > 0 || len(d.Standards) > 0
}

func (d *BootstrapData) Summary() string {
	var lines []string
	lines = append(lines, fmt.Sprintf("Skills: %d", len(d.Skills)))
	lines = append(lines, fmt.Sprintf("Rules: %d", len(d.Rules)))
	lines = append(lines, fmt.Sprintf("Standards: %d", len(d.Standards)))
	return fmt.Sprintf("%s\n", lines)
}
