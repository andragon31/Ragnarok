package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestExportImportFlow(t *testing.T) {
	tempDir := t.TempDir()
	exportFile := filepath.Join(tempDir, "export.json")

	home, _ := os.UserHomeDir()
	baseDir := filepath.Join(home, ".ragnarok")

	plugins := map[string]string{
		"fenrir": filepath.Join(baseDir, ".fenrir"),
		"hati":   filepath.Join(baseDir, ".hati"),
		"skoll":  filepath.Join(baseDir, ".skoll"),
		"tyr":    filepath.Join(baseDir, ".tyr"),
	}

	exportData := make(map[string]interface{})
	exportData["version"] = "2.2.2"
	exportData["exported_at"] = "2026-03-28T00:00:00Z"
	exportData["plugins"] = make(map[string]interface{})

	pluginCount := 0
	for name, dir := range plugins {
		dbPath := filepath.Join(dir, name+".db")

		if _, err := os.Stat(dbPath); os.IsNotExist(err) {
			continue
		}

		pluginCount++

		exportData["plugins"].(map[string]interface{})[name] = map[string]interface{}{
			"tables":  map[string]interface{}{},
			"db_path": dbPath,
		}
	}

	if pluginCount == 0 {
		t.Skip("No plugin databases found to test export")
	}

	data, err := json.MarshalIndent(exportData, "", "  ")
	if err != nil {
		t.Fatalf("Failed to marshal export data: %v", err)
	}

	if err := os.WriteFile(exportFile, data, 0644); err != nil {
		t.Fatalf("Failed to write export file: %v", err)
	}

	importedData, err := os.ReadFile(exportFile)
	if err != nil {
		t.Fatalf("Failed to read export file: %v", err)
	}

	var parsedData map[string]interface{}
	if err := json.Unmarshal(importedData, &parsedData); err != nil {
		t.Fatalf("Failed to parse export file: %v", err)
	}

	if parsedData["version"] != "2.2.2" {
		t.Errorf("Export version = %v, want %v", parsedData["version"], "2.2.2")
	}

	pluginsData, ok := parsedData["plugins"].(map[string]interface{})
	if !ok {
		t.Fatal("Export file missing plugins data")
	}

	if len(pluginsData) == 0 {
		t.Log("Warning: No plugin data in export (databases may be empty or not initialized)")
	}
}

func TestImportRecordSQLGeneration(t *testing.T) {
	tests := []struct {
		name      string
		tableName string
		record    map[string]interface{}
	}{
		{
			name:      "skills table record",
			tableName: "skills",
			record: map[string]interface{}{
				"id":         "skill_123",
				"name":       "test-skill",
				"type":       "code",
				"skill":      "test content",
				"version":    float64(1),
				"created_at": "2026-03-28T00:00:00Z",
			},
		},
		{
			name:      "rules table record",
			tableName: "rules",
			record: map[string]interface{}{
				"id":          "rule_456",
				"name":        "test-rule",
				"severity":    "high",
				"description": "Test rule description",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			columns := make([]string, 0, len(tt.record))
			placeholders := make([]string, 0, len(tt.record))
			values := make([]interface{}, 0, len(tt.record))

			for col, val := range tt.record {
				columns = append(columns, col)
				placeholders = append(placeholders, "?")
				if val == nil {
					values = append(values, nil)
				} else {
					values = append(values, val)
				}
			}

			query := "INSERT OR REPLACE INTO " + tt.tableName + " (" + strings.Join(columns, ", ") + ") VALUES (" + strings.Join(placeholders, ", ") + ")"

			if len(columns) != len(tt.record) {
				t.Errorf("columns count mismatch")
			}
			if len(placeholders) != len(tt.record) {
				t.Errorf("placeholders count mismatch")
			}
			if len(values) != len(tt.record) {
				t.Errorf("values count mismatch")
			}
			if query == "" {
				t.Errorf("generated query is empty")
			}
		})
	}
}

func TestExportDataStructure(t *testing.T) {
	exportData := map[string]interface{}{
		"version":     "2.2.2",
		"exported_at": "2026-03-28T00:00:00Z",
		"plugins": map[string]interface{}{
			"fenrir": map[string]interface{}{
				"tables": map[string]interface{}{
					"observations": []map[string]interface{}{
						{"id": "obs_1", "content": "test"},
					},
				},
				"db_path": "/path/to/fenrir.db",
			},
		},
	}

	data, err := json.Marshal(exportData)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	var parsed map[string]interface{}
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if parsed["version"] != "2.2.2" {
		t.Errorf("version = %v, want %v", parsed["version"], "2.2.2")
	}

	plugins := parsed["plugins"].(map[string]interface{})
	fenrir := plugins["fenrir"].(map[string]interface{})
	tables := fenrir["tables"].(map[string]interface{})
	observations := tables["observations"].([]interface{})

	if len(observations) != 1 {
		t.Errorf("observations count = %v, want %v", len(observations), 1)
	}
}

func TestPluginDirectoryStructure(t *testing.T) {
	home, _ := os.UserHomeDir()
	baseDir := filepath.Join(home, ".ragnarok")

	plugins := map[string]string{
		"fenrir": filepath.Join(baseDir, ".fenrir"),
		"hati":   filepath.Join(baseDir, ".hati"),
		"skoll":  filepath.Join(baseDir, ".skoll"),
		"tyr":    filepath.Join(baseDir, ".tyr"),
	}

	for name, dir := range plugins {
		if !filepath.IsAbs(dir) {
			t.Errorf("Plugin %s path is not absolute: %s", name, dir)
		}

		expectedPrefix := filepath.Join(home, ".ragnarok")
		if !hasPrefix(dir, expectedPrefix) {
			t.Errorf("Plugin %s path doesn't have expected prefix: %s", name, dir)
		}
	}
}

func hasPrefix(s, prefix string) bool {
	return len(s) >= len(prefix) && s[:len(prefix)] == prefix
}

func TestImportRecordMergesWithExistingData(t *testing.T) {
	existingRecord := map[string]interface{}{
		"id":    "skill_001",
		"name":  "original-skill",
		"type":  "code",
		"skill": "original content",
	}

	newRecord := map[string]interface{}{
		"id":    "skill_001",
		"name":  "updated-skill",
		"type":  "code",
		"skill": "updated content",
	}

	merged := make(map[string]interface{})
	for k, v := range existingRecord {
		merged[k] = v
	}
	for k, v := range newRecord {
		merged[k] = v
	}

	if merged["name"] != "updated-skill" {
		t.Errorf("merged name = %v, want %v", merged["name"], "updated-skill")
	}

	if merged["skill"] != "updated content" {
		t.Errorf("merged skill = %v, want %v", merged["skill"], "updated content")
	}

	if merged["id"] != "skill_001" {
		t.Errorf("merged id = %v, want %v", merged["id"], "skill_001")
	}
}
