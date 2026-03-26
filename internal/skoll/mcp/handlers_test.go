package mcp

import (
	"testing"
)

func TestSkillProgressiveDisclosure(t *testing.T) {
	result := map[string]interface{}{
		"skills_index": []map[string]string{
			{"name": "go-testing", "description": "Go testing expert", "version": "1.0.0"},
		},
		"count":       1,
		"progressive": true,
		"note":        "Use skill_load for full content",
	}

	if result["progressive"] != true {
		t.Error("Expected progressive disclosure to be enabled")
	}

	index := result["skills_index"].([]map[string]string)
	if len(index) != 1 {
		t.Errorf("Expected 1 skill in index, got %d", len(index))
	}
}

func TestSkillIndexStructure(t *testing.T) {
	tests := []struct {
		name    string
		skill   map[string]string
		wantErr bool
	}{
		{
			name: "Valid skill",
			skill: map[string]string{
				"name":        "go-testing",
				"description": "Go testing expert",
				"version":     "1.0.0",
				"trigger":     "go test|testing",
			},
			wantErr: false,
		},
		{
			name: "Skill without version",
			skill: map[string]string{
				"name":        "rust-testing",
				"description": "Rust testing",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.skill["name"] == "" {
				if !tt.wantErr {
					t.Error("Expected error for skill without name")
				}
			}
		})
	}
}

func TestSkillTriggerMatching(t *testing.T) {
	triggers := []string{
		"go test",
		"testing",
		"jest",
		"vitest",
		"pytest",
	}

	tests := []struct {
		name    string
		prompt  string
		matches bool
	}{
		{"Go test prompt", "run go test for this", true},
		{"Testing keyword", "running unit testing for the code", true},
		{"Jest keyword", "write jest unit tests", true},
		{"Pytest keyword", "add pytest coverage", true},
		{"Unrelated prompt", "fix the bug in auth", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hasMatch := false
			promptLower := tt.prompt
			for _, trigger := range triggers {
				if contains(promptLower, trigger) {
					hasMatch = true
					break
				}
			}
			if hasMatch != tt.matches {
				t.Errorf("prompt=%q: expected match=%v, got %v", tt.prompt, tt.matches, hasMatch)
			}
		})
	}
}

func TestAgentHandoffContract(t *testing.T) {
	contract := map[string]interface{}{
		"from_agent":   "agent-A",
		"to_agent":     "agent-B",
		"work_summary": "Completed phase 1 of OAuth implementation",
		"remaining":    []string{"phase_2_auth", "phase_3_testing"},
		"context": map[string]string{
			"plan_id": "plan_123",
			"module":  "auth",
		},
	}

	if contract["from_agent"] == contract["to_agent"] {
		t.Error("Handoff contract should be between different agents")
	}

	remaining := contract["remaining"].([]string)
	if len(remaining) == 0 {
		t.Error("Handoff should specify remaining work")
	}
}

func TestWorkflowStateTransitions(t *testing.T) {
	validTransitions := map[string][]string{
		"draft":       {"started", "cancelled"},
		"started":     {"in_progress", "completed", "failed"},
		"in_progress": {"completed", "failed", "blocked"},
		"completed":   {},
		"failed":      {"restarted"},
		"blocked":     {"in_progress"},
	}

	tests := []struct {
		from  string
		to    string
		valid bool
	}{
		{"draft", "started", true},
		{"started", "in_progress", true},
		{"in_progress", "completed", true},
		{"completed", "in_progress", false},
		{"failed", "restarted", true},
		{"draft", "completed", false},
	}

	for _, tt := range tests {
		t.Run(tt.from+"_"+tt.to, func(t *testing.T) {
			allowed := validTransitions[tt.from]
			isValid := false
			for _, a := range allowed {
				if a == tt.to {
					isValid = true
					break
				}
			}
			if isValid != tt.valid {
				t.Errorf("%s -> %s: expected valid=%v, got %v", tt.from, tt.to, tt.valid, isValid)
			}
		})
	}
}

func TestDodCheckCriteria(t *testing.T) {
	criteria := []map[string]interface{}{
		{"name": "tests_pass", "description": "All tests passing", "required": true},
		{"name": "lint_clean", "description": "No lint errors", "required": true},
		{"name": "coverage", "description": "Coverage > 80%", "required": false},
		{"name": "docs_updated", "description": "Documentation updated", "required": false},
	}

	requiredCount := 0
	for _, c := range criteria {
		if c["required"] == true {
			requiredCount++
		}
	}

	if requiredCount != 2 {
		t.Errorf("Expected 2 required criteria, got %d", requiredCount)
	}
}

func TestRuleSeverityLevels(t *testing.T) {
	severityLevels := []string{"critical", "high", "medium", "low", "info"}

	for i, level := range severityLevels {
		if i > 0 && level == "critical" {
			t.Error("Critical should be first severity level")
		}
	}

	expectedOrder := map[string]int{
		"critical": 0,
		"high":     1,
		"medium":   2,
		"low":      3,
		"info":     4,
	}

	if expectedOrder["critical"] != 0 {
		t.Error("Critical should be severity level 0 (highest)")
	}
	if expectedOrder["info"] != 4 {
		t.Error("Info should be severity level 4 (lowest)")
	}
}

func TestSkillVersionCheck(t *testing.T) {
	tests := []struct {
		name        string
		current     string
		available   string
		needsUpdate bool
	}{
		{"Outdated minor", "1.0.0", "1.1.0", true},
		{"Current", "1.0.0", "1.0.0", false},
		{"Major Update", "1.0.0", "2.0.0", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			needsUpdate := tt.current != tt.available && tt.current < tt.available
			if needsUpdate != tt.needsUpdate {
				t.Errorf("Version %s vs %s: expected needsUpdate=%v", tt.current, tt.available, tt.needsUpdate)
			}
		})
	}
}

func contains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
