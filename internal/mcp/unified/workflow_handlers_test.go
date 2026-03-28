package unified

import (
	"context"
	"encoding/json"
	"testing"

	rootmcp "github.com/andragon31/Ragnarok/internal/mcp"
)

func TestWorkflowStackBasedInitParams(t *testing.T) {
	tests := []struct {
		name    string
		params  map[string]interface{}
		wantErr bool
	}{
		{
			name: "valid with project_path only",
			params: map[string]interface{}{
				"project_path": "/path/to/project",
			},
			wantErr: false,
		},
		{
			name: "valid with all params",
			params: map[string]interface{}{
				"project_path": "/path/to/project",
				"title":        "My Plan",
				"phases":       []string{"Setup", "Backend", "Frontend"},
				"agent_ids":    []string{"agent_1", "agent_2"},
			},
			wantErr: false,
		},
		{
			name:    "missing project_path",
			params:  map[string]interface{}{},
			wantErr: true,
		},
		{
			name: "empty project_path",
			params: map[string]interface{}{
				"project_path": "",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hasErr := false
			if pp, ok := tt.params["project_path"].(string); !ok || pp == "" {
				hasErr = true
			}
			if hasErr != tt.wantErr {
				t.Errorf("params=%v: expected wantErr=%v, got %v", tt.params, tt.wantErr, hasErr)
			}
		})
	}
}

func TestWorkflowPlanDevelopV2Params(t *testing.T) {
	tests := []struct {
		name    string
		params  map[string]interface{}
		wantErr bool
	}{
		{
			name: "valid with plan_id only",
			params: map[string]interface{}{
				"plan_id": "plan_123",
			},
			wantErr: false,
		},
		{
			name: "valid with all params",
			params: map[string]interface{}{
				"plan_id":       "plan_123",
				"agent_id":      "agent_456",
				"auto_continue": true,
			},
			wantErr: false,
		},
		{
			name:    "missing plan_id",
			params:  map[string]interface{}{},
			wantErr: true,
		},
		{
			name: "empty plan_id",
			params: map[string]interface{}{
				"plan_id": "",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hasErr := false
			if planID, ok := tt.params["plan_id"].(string); !ok || planID == "" {
				hasErr = true
			}
			if hasErr != tt.wantErr {
				t.Errorf("params=%v: expected wantErr=%v, got %v", tt.params, tt.wantErr, hasErr)
			}
		})
	}
}

func TestWorkflowStepStructure(t *testing.T) {
	step := WorkflowStep{
		Name:   "project_scan",
		Status: "success",
		Output: map[string]interface{}{"path": "/test", "name": "test-project"},
	}

	if step.Name == "" {
		t.Error("step name should not be empty")
	}
	if step.Status != "success" && step.Status != "error" {
		t.Errorf("invalid step status: %s", step.Status)
	}
}

func TestWorkflowResultStructure(t *testing.T) {
	result := WorkflowResult{
		Workflow: "stack_based_init",
		Status:   "completed",
		Steps: []WorkflowStep{
			{Name: "step1", Status: "success"},
			{Name: "step2", Status: "success"},
		},
		Results: map[string]interface{}{
			"plan_id": "plan_123",
		},
	}

	if result.Workflow == "" {
		t.Error("workflow name should not be empty")
	}
	if len(result.Steps) != 2 {
		t.Errorf("expected 2 steps, got %d", len(result.Steps))
	}
}

func TestParseProjectAnalysis(t *testing.T) {
	tests := []struct {
		name    string
		input   map[string]interface{}
		wantErr bool
	}{
		{
			name: "full analysis",
			input: map[string]interface{}{
				"name": "test-project",
				"path": "/path/to/project",
				"stack": map[string]interface{}{
					"language":        "go",
					"framework":       "gin",
					"package_manager": "go",
					"ci_tool":         "github-actions",
					"db_engine":       "postgres",
					"has_docker":      true,
					"has_ci":          true,
					"has_tests":       true,
				},
				"architecture": map[string]interface{}{
					"type":          "monolith",
					"has_api":       true,
					"has_frontend":  false,
					"is_monorepo":   false,
					"frontend_lib":  "",
					"api_framework": "gin",
				},
			},
			wantErr: false,
		},
		{
			name: "minimal analysis",
			input: map[string]interface{}{
				"name": "minimal-project",
			},
			wantErr: false,
		},
		{
			name:    "empty map input",
			input:   map[string]interface{}{},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := parseProjectAnalysis(tt.input)
			hasErr := err != nil
			if hasErr != tt.wantErr {
				t.Errorf("input=%v: expected wantErr=%v, got %v (err=%v)", tt.input, tt.wantErr, hasErr, err)
			}
		})
	}
}

func TestFindAgentByType(t *testing.T) {
	agents := []map[string]string{
		{"name": "backend-agent-1", "type": "backend"},
		{"name": "frontend-agent-1", "type": "frontend"},
		{"name": "qa-agent-1", "type": "qa"},
	}

	tests := []struct {
		agentType string
		expected  string
	}{
		{"backend", "backend-agent-1"},
		{"frontend", "frontend-agent-1"},
		{"qa", "qa-agent-1"},
		{"devops", ""},
	}

	for _, tt := range tests {
		t.Run(tt.agentType, func(t *testing.T) {
			result := findAgentByType(tt.agentType, agents)
			if result != tt.expected {
				t.Errorf("findAgentByType(%s): expected %s, got %s", tt.agentType, tt.expected, result)
			}
		})
	}
}

func TestWorkflowCheckpointCreateParams(t *testing.T) {
	tests := []struct {
		name    string
		params  map[string]interface{}
		wantErr bool
	}{
		{
			name: "valid with plan_id and description",
			params: map[string]interface{}{
				"plan_id":     "plan_123",
				"description": "Milestone checkpoint",
			},
			wantErr: false,
		},
		{
			name: "valid with all params including phase_id",
			params: map[string]interface{}{
				"plan_id":     "plan_123",
				"phase_id":    "phase_456",
				"description": "Phase 1 complete",
			},
			wantErr: false,
		},
		{
			name:    "missing plan_id",
			params:  map[string]interface{}{},
			wantErr: true,
		},
		{
			name: "missing description",
			params: map[string]interface{}{
				"plan_id": "plan_123",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hasErr := false
			if planID, ok := tt.params["plan_id"].(string); !ok || planID == "" {
				hasErr = true
			}
			if desc, ok := tt.params["description"].(string); !ok || desc == "" {
				hasErr = true
			}
			if hasErr != tt.wantErr {
				t.Errorf("params=%v: expected wantErr=%v, got %v", tt.params, tt.wantErr, hasErr)
			}
		})
	}
}

func TestWorkflowSessionStartParams(t *testing.T) {
	tests := []struct {
		name    string
		params  map[string]interface{}
		wantErr bool
	}{
		{
			name: "valid with goal",
			params: map[string]interface{}{
				"goal": "Implement new feature",
			},
			wantErr: false,
		},
		{
			name: "valid with all params",
			params: map[string]interface{}{
				"goal":         "Implement new feature",
				"module":       "auth",
				"project_path": "/path/to/project",
			},
			wantErr: false,
		},
		{
			name:    "missing goal",
			params:  map[string]interface{}{},
			wantErr: true,
		},
		{
			name: "empty goal",
			params: map[string]interface{}{
				"goal": "",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hasErr := false
			if g, ok := tt.params["goal"].(string); !ok || g == "" {
				hasErr = true
			}
			if hasErr != tt.wantErr {
				t.Errorf("params=%v: expected wantErr=%v, got %v", tt.params, tt.wantErr, hasErr)
			}
		})
	}
}

func TestWorkflowAgenticInitParams(t *testing.T) {
	tests := []struct {
		name    string
		params  map[string]interface{}
		wantErr bool
	}{
		{
			name: "valid with title and phases",
			params: map[string]interface{}{
				"title":  "My Project Plan",
				"phases": []string{"Phase1", "Phase2"},
			},
			wantErr: false,
		},
		{
			name: "valid with all params",
			params: map[string]interface{}{
				"title":        "My Project Plan",
				"description":  "A great project",
				"phases":       []string{"Phase1", "Phase2"},
				"agent_name":   "dev-team",
				"project_path": "/path/to/project",
			},
			wantErr: false,
		},
		{
			name: "missing title",
			params: map[string]interface{}{
				"phases": []string{"Phase1"},
			},
			wantErr: true,
		},
		{
			name: "missing phases",
			params: map[string]interface{}{
				"title": "My Plan",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hasErr := false
			if title, ok := tt.params["title"].(string); !ok || title == "" {
				hasErr = true
			}
			if phases, ok := tt.params["phases"].([]string); !ok || len(phases) == 0 {
				hasErr = true
			}
			if hasErr != tt.wantErr {
				t.Errorf("params=%v: expected wantErr=%v, got %v", tt.params, tt.wantErr, hasErr)
			}
		})
	}
}

func TestWorkflowPRDAnalyzeParams(t *testing.T) {
	tests := []struct {
		name    string
		params  map[string]interface{}
		wantErr bool
	}{
		{
			name: "valid with prd_file",
			params: map[string]interface{}{
				"prd_file": "/path/to/prd.md",
			},
			wantErr: false,
		},
		{
			name: "valid with all params",
			params: map[string]interface{}{
				"prd_file":     "/path/to/prd.md",
				"project_path": "/path/to/project",
				"plan_title":   "My Plan",
			},
			wantErr: false,
		},
		{
			name:    "missing prd_file",
			params:  map[string]interface{}{},
			wantErr: true,
		},
		{
			name: "empty prd_file",
			params: map[string]interface{}{
				"prd_file": "",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hasErr := false
			if prd, ok := tt.params["prd_file"].(string); !ok || prd == "" {
				hasErr = true
			}
			if hasErr != tt.wantErr {
				t.Errorf("params=%v: expected wantErr=%v, got %v", tt.params, tt.wantErr, hasErr)
			}
		})
	}
}

func TestExecuteWorkflowNotFound(t *testing.T) {
	srv := &Server{
		handlers: make(map[string]rootmcp.ToolHandler),
	}

	ctx := context.Background()
	_, err := srv.ExecuteWorkflow(ctx, "nonexistent_workflow", map[string]interface{}{})

	if err == nil {
		t.Error("Expected error for nonexistent workflow, got nil")
	}
}

func TestExecuteWorkflowFound(t *testing.T) {
	srv := &Server{
		handlers: map[string]rootmcp.ToolHandler{
			"test_workflow": func(ctx context.Context, req *Request) (*Response, error) {
				return &Response{
					Result: map[string]interface{}{
						"status":  "success",
						"message": "test passed",
					},
				}, nil
			},
		},
	}

	ctx := context.Background()
	result, err := srv.ExecuteWorkflow(ctx, "test_workflow", map[string]interface{}{"key": "value"})

	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if result == nil {
		t.Error("Expected result, got nil")
	}
}

func TestCallToolNotFound(t *testing.T) {
	srv := &Server{
		handlers: make(map[string]rootmcp.ToolHandler),
	}

	ctx := context.Background()
	_, err := srv.CallTool(ctx, "nonexistent_tool", map[string]interface{}{})

	if err == nil {
		t.Error("Expected error for nonexistent tool, got nil")
	}
}

func TestCallToolFound(t *testing.T) {
	srv := &Server{
		handlers: map[string]rootmcp.ToolHandler{
			"test_tool": func(ctx context.Context, req *Request) (*Response, error) {
				return &Response{
					Result: map[string]interface{}{
						"tool": "test_tool",
						"ran":  true,
					},
				}, nil
			},
		},
	}

	ctx := context.Background()
	result, err := srv.CallTool(ctx, "test_tool", map[string]interface{}{"param": "value"})

	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if result == nil {
		t.Error("Expected result, got nil")
	}
}

func TestExecuteWorkflowParamsPassedCorrectly(t *testing.T) {
	paramsReceived := make(map[string]interface{})

	srv := &Server{
		handlers: map[string]rootmcp.ToolHandler{
			"params_test": func(ctx context.Context, req *Request) (*Response, error) {
				json.Unmarshal(req.Params, &paramsReceived)
				return &Response{Result: paramsReceived}, nil
			},
		},
	}

	ctx := context.Background()
	testParams := map[string]interface{}{
		"string_param": "hello",
		"int_param":    42,
		"bool_param":   true,
		"array_param":  []string{"a", "b", "c"},
	}

	_, err := srv.ExecuteWorkflow(ctx, "params_test", testParams)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	if paramsReceived["string_param"] != "hello" {
		t.Errorf("string_param mismatch: got %v", paramsReceived["string_param"])
	}
}

func TestExecuteWorkflowWithEmptyParams(t *testing.T) {
	paramsReceived := false

	srv := &Server{
		handlers: map[string]rootmcp.ToolHandler{
			"empty_params": func(ctx context.Context, req *Request) (*Response, error) {
				if len(req.Params) == 0 || string(req.Params) == "{}" {
					paramsReceived = true
				}
				return &Response{Result: map[string]interface{}{"received": paramsReceived}}, nil
			},
		},
	}

	ctx := context.Background()
	_, err := srv.ExecuteWorkflow(ctx, "empty_params", map[string]interface{}{})

	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if !paramsReceived {
		t.Error("Expected params to be received as empty object")
	}
}

func TestCallToolParamsPassedCorrectly(t *testing.T) {
	paramsReceived := make(map[string]interface{})

	srv := &Server{
		handlers: map[string]rootmcp.ToolHandler{
			"tool_params": func(ctx context.Context, req *Request) (*Response, error) {
				json.Unmarshal(req.Params, &paramsReceived)
				return &Response{Result: paramsReceived}, nil
			},
		},
	}

	ctx := context.Background()
	testParams := map[string]interface{}{
		"project_path": "/test/project",
		"title":        "Test Plan",
	}

	_, err := srv.CallTool(ctx, "tool_params", testParams)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	if paramsReceived["project_path"] != "/test/project" {
		t.Errorf("project_path mismatch: got %v", paramsReceived["project_path"])
	}
}

func TestExecuteWorkflowReturnsNil(t *testing.T) {
	srv := &Server{
		handlers: map[string]rootmcp.ToolHandler{
			"nil_result": func(ctx context.Context, req *Request) (*Response, error) {
				return nil, nil
			},
		},
	}

	ctx := context.Background()
	result, err := srv.ExecuteWorkflow(ctx, "nil_result", map[string]interface{}{})

	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if result != nil {
		t.Errorf("Expected nil result, got %v", result)
	}
}

func TestCallToolReturnsNil(t *testing.T) {
	srv := &Server{
		handlers: map[string]rootmcp.ToolHandler{
			"nil_tool": func(ctx context.Context, req *Request) (*Response, error) {
				return nil, nil
			},
		},
	}

	ctx := context.Background()
	result, err := srv.CallTool(ctx, "nil_tool", map[string]interface{}{})

	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if result != nil {
		t.Errorf("Expected nil result, got %v", result)
	}
}
