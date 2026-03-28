package unified

import (
	"testing"
)

func TestWorkflowResultStruct(t *testing.T) {
	result := WorkflowResult{
		Workflow: "test_workflow",
		Status:   "completed",
		Steps: []WorkflowStep{
			{Name: "step1", Status: "success"},
			{Name: "step2", Status: "error", Error: "failed"},
		},
		Results: map[string]interface{}{
			"plan_id": "plan_123",
		},
	}

	if result.Workflow != "test_workflow" {
		t.Errorf("Workflow mismatch: got %s, want test_workflow", result.Workflow)
	}
	if result.Status != "completed" {
		t.Errorf("Status mismatch: got %s, want completed", result.Status)
	}
	if len(result.Steps) != 2 {
		t.Errorf("Steps length mismatch: got %d, want 2", len(result.Steps))
	}
}

func TestWorkflowStepStruct(t *testing.T) {
	step := WorkflowStep{
		Name:   "test_step",
		Status: "success",
		Output: map[string]interface{}{"key": "value"},
		Error:  "",
	}

	if step.Name != "test_step" {
		t.Errorf("Name mismatch: got %s, want test_step", step.Name)
	}
	if step.Status != "success" {
		t.Errorf("Status mismatch: got %s, want success", step.Status)
	}
	if step.Output == nil {
		t.Error("Output should not be nil")
	}
}

func TestWorkflowStepWithError(t *testing.T) {
	step := WorkflowStep{
		Name:   "failed_step",
		Status: "error",
		Error:  "something went wrong",
	}

	if step.Status != "error" {
		t.Errorf("Status mismatch: got %s, want error", step.Status)
	}
	if step.Error != "something went wrong" {
		t.Errorf("Error mismatch: got %s, want 'something went wrong'", step.Error)
	}
}

func TestWorkflowResultWithNilResults(t *testing.T) {
	result := WorkflowResult{
		Workflow: "minimal_workflow",
		Status:   "success",
		Steps:    []WorkflowStep{},
		Results:  nil,
	}

	if result.Results != nil {
		t.Errorf("Results should be nil, got %v", result.Results)
	}
	if len(result.Steps) != 0 {
		t.Errorf("Steps should be empty, got %d", len(result.Steps))
	}
}

func TestWorkflowStepEmptyOutput(t *testing.T) {
	step := WorkflowStep{
		Name:   "step_no_output",
		Status: "success",
		Output: nil,
	}

	if step.Output != nil {
		t.Errorf("Output should be nil, got %v", step.Output)
	}
}

func TestWorkflowResultError(t *testing.T) {
	result := WorkflowResult{
		Workflow: "failed_workflow",
		Status:   "error",
		Error:    "workflow failed",
	}

	if result.Error != "workflow failed" {
		t.Errorf("Error mismatch: got %s, want 'workflow failed'", result.Error)
	}
}

func TestParseProjectAnalysisHelper(t *testing.T) {
	input := map[string]interface{}{
		"name": "test-project",
		"path": "/path/to/project",
		"stack": map[string]interface{}{
			"language":        "go",
			"framework":       "gin",
			"package_manager": "go",
		},
	}

	analysis, err := parseProjectAnalysis(input)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	if analysis.Name != "test-project" {
		t.Errorf("Name mismatch: got %s, want test-project", analysis.Name)
	}

	if analysis.Stack == nil {
		t.Fatal("Stack should not be nil")
	}

	if analysis.Stack.Language != "go" {
		t.Errorf("Language mismatch: got %s, want go", analysis.Stack.Language)
	}
}

func TestParseProjectAnalysisMinimal(t *testing.T) {
	input := map[string]interface{}{
		"name": "minimal-project",
	}

	analysis, err := parseProjectAnalysis(input)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	if analysis == nil {
		t.Fatal("Analysis should not be nil")
	}

	if analysis.Name != "minimal-project" {
		t.Errorf("Name mismatch: got %s, want minimal-project", analysis.Name)
	}
}

func TestParseProjectAnalysisEmpty(t *testing.T) {
	input := map[string]interface{}{}

	analysis, err := parseProjectAnalysis(input)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	if analysis == nil {
		t.Fatal("Analysis should not be nil")
	}
}

func TestParseProjectAnalysisWithArchitecture(t *testing.T) {
	input := map[string]interface{}{
		"name": "full-project",
		"architecture": map[string]interface{}{
			"type":          "api",
			"has_api":       true,
			"has_frontend":  false,
			"is_monorepo":   false,
			"api_framework": "gin",
		},
	}

	analysis, err := parseProjectAnalysis(input)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	if analysis.Architecture == nil {
		t.Fatal("Architecture should not be nil")
	}

	if analysis.Architecture.Type != "api" {
		t.Errorf("Architecture type mismatch: got %s, want api", analysis.Architecture.Type)
	}

	if !analysis.Architecture.HasAPI {
		t.Error("HasAPI should be true")
	}

	if analysis.Architecture.HasFrontend {
		t.Error("HasFrontend should be false")
	}
}

func TestFindAgentByTypeWithEmptyList(t *testing.T) {
	agents := []map[string]string{}

	result := findAgentByType("backend", agents)
	if result != "" {
		t.Errorf("Expected empty string for empty agent list, got %s", result)
	}
}

func TestFindAgentByTypeWithMatching(t *testing.T) {
	agents := []map[string]string{
		{"name": "agent-1", "type": "backend"},
		{"name": "agent-2", "type": "frontend"},
	}

	result := findAgentByType("backend", agents)
	if result != "agent-1" {
		t.Errorf("Expected agent-1, got %s", result)
	}
}

func TestFindAgentByTypeWithNoMatch(t *testing.T) {
	agents := []map[string]string{
		{"name": "agent-1", "type": "backend"},
	}

	result := findAgentByType("devops", agents)
	if result != "" {
		t.Errorf("Expected empty string for no match, got %s", result)
	}
}
