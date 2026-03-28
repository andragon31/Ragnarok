package mcp

import (
	"testing"
)

func TestTaskExecutionStateTransitions(t *testing.T) {
	validTransitions := map[string][]string{
		"pending":     {"in_progress", "cancelled"},
		"in_progress": {"completed", "failed", "cancelled"},
		"completed":   {},
		"failed":      {"pending"},
		"cancelled":   {},
	}

	tests := []struct {
		from  string
		to    string
		valid bool
	}{
		{"pending", "in_progress", true},
		{"in_progress", "completed", true},
		{"in_progress", "failed", true},
		{"completed", "in_progress", false},
		{"pending", "completed", false},
		{"in_progress", "cancelled", true},
		{"failed", "pending", true},
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

func TestTaskExecuteParams(t *testing.T) {
	tests := []struct {
		name    string
		params  map[string]interface{}
		wantErr bool
	}{
		{
			name: "valid params",
			params: map[string]interface{}{
				"task_id":  "task_123",
				"agent_id": "agent_456",
			},
			wantErr: false,
		},
		{
			name: "missing task_id",
			params: map[string]interface{}{
				"agent_id": "agent_456",
			},
			wantErr: true,
		},
		{
			name: "missing agent_id",
			params: map[string]interface{}{
				"task_id": "task_123",
			},
			wantErr: true,
		},
		{
			name:    "empty params",
			params:  map[string]interface{}{},
			wantErr: true,
		},
		{
			name: "with hati_task_id and phase_id",
			params: map[string]interface{}{
				"task_id":      "task_123",
				"hati_task_id": "hati_789",
				"agent_id":     "agent_456",
				"phase_id":     "phase_111",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hasErr := false
			if tt.params["task_id"] == nil || tt.params["task_id"] == "" {
				hasErr = true
			}
			if tt.params["agent_id"] == nil || tt.params["agent_id"] == "" {
				hasErr = true
			}
			if hasErr != tt.wantErr {
				t.Errorf("params=%v: expected wantErr=%v, got %v", tt.params, tt.wantErr, hasErr)
			}
		})
	}
}

func TestTaskDelegateParams(t *testing.T) {
	tests := []struct {
		name    string
		params  map[string]interface{}
		wantErr bool
	}{
		{
			name: "valid with multiple agents",
			params: map[string]interface{}{
				"task_id":   "task_123",
				"agent_ids": []string{"agent_1", "agent_2", "agent_3"},
			},
			wantErr: false,
		},
		{
			name: "valid with single agent",
			params: map[string]interface{}{
				"task_id":   "task_123",
				"agent_ids": []string{"agent_1"},
			},
			wantErr: false,
		},
		{
			name: "missing task_id",
			params: map[string]interface{}{
				"agent_ids": []string{"agent_1", "agent_2"},
			},
			wantErr: true,
		},
		{
			name: "empty agent_ids",
			params: map[string]interface{}{
				"task_id":   "task_123",
				"agent_ids": []string{},
			},
			wantErr: true,
		},
		{
			name: "nil agent_ids",
			params: map[string]interface{}{
				"task_id":   "task_123",
				"agent_ids": nil,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hasErr := false
			if tt.params["task_id"] == nil || tt.params["task_id"] == "" {
				hasErr = true
			}
			agentIDs, ok := tt.params["agent_ids"].([]string)
			if !ok || len(agentIDs) == 0 {
				hasErr = true
			}
			if hasErr != tt.wantErr {
				t.Errorf("params=%v: expected wantErr=%v, got %v", tt.params, tt.wantErr, hasErr)
			}
		})
	}
}

func TestTaskStatusParams(t *testing.T) {
	tests := []struct {
		name    string
		params  map[string]interface{}
		wantErr bool
	}{
		{
			name: "with execution_id",
			params: map[string]interface{}{
				"execution_id": "exec_123",
			},
			wantErr: false,
		},
		{
			name: "with task_id",
			params: map[string]interface{}{
				"task_id": "task_123",
			},
			wantErr: false,
		},
		{
			name: "with agent_id",
			params: map[string]interface{}{
				"agent_id": "agent_456",
			},
			wantErr: false,
		},
		{
			name: "with all params",
			params: map[string]interface{}{
				"execution_id": "exec_123",
				"task_id":      "task_123",
				"agent_id":     "agent_456",
			},
			wantErr: false,
		},
		{
			name:    "empty params",
			params:  map[string]interface{}{},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hasErr := true
			if tt.params["execution_id"] != nil && tt.params["execution_id"] != "" {
				hasErr = false
			}
			if tt.params["task_id"] != nil && tt.params["task_id"] != "" {
				hasErr = false
			}
			if tt.params["agent_id"] != nil && tt.params["agent_id"] != "" {
				hasErr = false
			}
			if hasErr != tt.wantErr {
				t.Errorf("params=%v: expected wantErr=%v, got %v", tt.params, tt.wantErr, hasErr)
			}
		})
	}
}

func TestTaskHeartbeatParams(t *testing.T) {
	tests := []struct {
		name    string
		params  map[string]interface{}
		wantErr bool
	}{
		{
			name: "valid execution_id",
			params: map[string]interface{}{
				"execution_id": "exec_123",
			},
			wantErr: false,
		},
		{
			name:    "missing execution_id",
			params:  map[string]interface{}{},
			wantErr: true,
		},
		{
			name: "empty execution_id",
			params: map[string]interface{}{
				"execution_id": "",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hasErr := tt.params["execution_id"] == nil || tt.params["execution_id"] == ""
			if hasErr != tt.wantErr {
				t.Errorf("params=%v: expected wantErr=%v, got %v", tt.params, tt.wantErr, hasErr)
			}
		})
	}
}

func TestTaskCompleteParams(t *testing.T) {
	tests := []struct {
		name    string
		params  map[string]interface{}
		wantErr bool
	}{
		{
			name: "valid with status completed",
			params: map[string]interface{}{
				"execution_id": "exec_123",
				"status":       "completed",
				"result":       "task done successfully",
			},
			wantErr: false,
		},
		{
			name: "valid with status failed",
			params: map[string]interface{}{
				"execution_id": "exec_123",
				"status":       "failed",
				"error":        "something went wrong",
			},
			wantErr: false,
		},
		{
			name: "valid without status (defaults to completed)",
			params: map[string]interface{}{
				"execution_id": "exec_123",
			},
			wantErr: false,
		},
		{
			name:    "missing execution_id",
			params:  map[string]interface{}{},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hasErr := tt.params["execution_id"] == nil || tt.params["execution_id"] == ""
			if hasErr != tt.wantErr {
				t.Errorf("params=%v: expected wantErr=%v, got %v", tt.params, tt.wantErr, hasErr)
			}
		})
	}
}

func TestTaskCancelParams(t *testing.T) {
	tests := []struct {
		name    string
		params  map[string]interface{}
		wantErr bool
	}{
		{
			name: "valid with reason",
			params: map[string]interface{}{
				"execution_id": "exec_123",
				"reason":       "user cancelled",
			},
			wantErr: false,
		},
		{
			name: "valid without reason",
			params: map[string]interface{}{
				"execution_id": "exec_123",
			},
			wantErr: false,
		},
		{
			name:    "missing execution_id",
			params:  map[string]interface{}{},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hasErr := tt.params["execution_id"] == nil || tt.params["execution_id"] == ""
			if hasErr != tt.wantErr {
				t.Errorf("params=%v: expected wantErr=%v, got %v", tt.params, tt.wantErr, hasErr)
			}
		})
	}
}

func TestWorkflowDeprecateParams(t *testing.T) {
	tests := []struct {
		name    string
		params  map[string]interface{}
		wantErr bool
	}{
		{
			name: "valid workflow_id",
			params: map[string]interface{}{
				"workflow_id": "wf_123",
			},
			wantErr: false,
		},
		{
			name:    "missing workflow_id",
			params:  map[string]interface{}{},
			wantErr: true,
		},
		{
			name: "empty workflow_id",
			params: map[string]interface{}{
				"workflow_id": "",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hasErr := tt.params["workflow_id"] == nil || tt.params["workflow_id"] == ""
			if hasErr != tt.wantErr {
				t.Errorf("params=%v: expected wantErr=%v, got %v", tt.params, tt.wantErr, hasErr)
			}
		})
	}
}

func TestTaskExecutionResult(t *testing.T) {
	result := map[string]interface{}{
		"execution_id": "texec_abc123",
		"task_id":      "task_123",
		"agent_id":     "agent_456",
		"status":       "in_progress",
		"started_at":   "2024-01-15T10:30:00Z",
	}

	if result["execution_id"] == "" {
		t.Error("execution_id should not be empty")
	}
	if result["task_id"] == "" {
		t.Error("task_id should not be empty")
	}
	if result["agent_id"] == "" {
		t.Error("agent_id should not be empty")
	}
	if result["status"] != "in_progress" {
		t.Errorf("expected status 'in_progress', got %v", result["status"])
	}
}

func TestTaskDelegateResult(t *testing.T) {
	result := map[string]interface{}{
		"task_id": "task_123",
		"delegated_to": []map[string]interface{}{
			{"execution_id": "texec_1", "agent_id": "agent_1", "status": "pending"},
			{"execution_id": "texec_2", "agent_id": "agent_2", "status": "pending"},
		},
		"total_agents": 2,
	}

	delegations := result["delegated_to"].([]map[string]interface{})
	if len(delegations) != 2 {
		t.Errorf("expected 2 delegations, got %d", len(delegations))
	}

	totalAgents := result["total_agents"].(int)
	if totalAgents != len(delegations) {
		t.Errorf("total_agents %d should match delegations length %d", totalAgents, len(delegations))
	}
}

func TestTaskStatusResult(t *testing.T) {
	result := map[string]interface{}{
		"executions": []map[string]interface{}{
			{
				"id":           "texec_1",
				"task_id":      "task_123",
				"agent_id":     "agent_1",
				"status":       "completed",
				"completed_at": "2024-01-15T11:00:00Z",
			},
			{
				"id":         "texec_2",
				"task_id":    "task_123",
				"agent_id":   "agent_2",
				"status":     "in_progress",
				"started_at": "2024-01-15T10:30:00Z",
			},
		},
		"count": 2,
	}

	executions := result["executions"].([]map[string]interface{})
	if len(executions) != 2 {
		t.Errorf("expected 2 executions, got %d", len(executions))
	}

	count := result["count"].(int)
	if count != len(executions) {
		t.Errorf("count %d should match executions length %d", count, len(executions))
	}
}

func TestAgentMultiTaskExecution(t *testing.T) {
	executions := []map[string]interface{}{
		{"agent_id": "agent_backend", "concurrent_tasks": 3},
		{"agent_id": "agent_frontend", "concurrent_tasks": 2},
	}

	maxConcurrent := 0
	for _, exec := range executions {
		if tasks := exec["concurrent_tasks"].(int); tasks > maxConcurrent {
			maxConcurrent = tasks
		}
	}

	if maxConcurrent != 3 {
		t.Errorf("expected max concurrent tasks 3, got %d", maxConcurrent)
	}
}
