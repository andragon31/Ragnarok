package main

import (
	"testing"
)

func TestGetDefaultPhases(t *testing.T) {
	tests := []struct {
		stack   string
		wantLen int
	}{
		{"go", 7},
		{"node", 7},
		{"python", 6},
		{"rust", 6},
		{"dotnet", 6},
		{"java", 6},
		{"unknown", 5},
	}

	for _, tt := range tests {
		t.Run(tt.stack, func(t *testing.T) {
			got := getDefaultPhases(tt.stack)
			if len(got) != tt.wantLen {
				t.Errorf("getDefaultPhases(%s) returned %d phases, want %d", tt.stack, len(got), tt.wantLen)
			}
			if len(got) == 0 {
				t.Errorf("getDefaultPhases(%s) returned empty slice", tt.stack)
			}
		})
	}
}

func TestGetPlanID(t *testing.T) {
	tests := []struct {
		name   string
		result interface{}
		want   string
	}{
		{
			name:   "valid plan_id",
			result: map[string]interface{}{"plan_id": "plan_123"},
			want:   "plan_123",
		},
		{
			name:   "nil result",
			result: nil,
			want:   "",
		},
		{
			name:   "empty map",
			result: map[string]interface{}{},
			want:   "",
		},
		{
			name:   "wrong type",
			result: "string instead of map",
			want:   "",
		},
		{
			name:   "plan_id missing",
			result: map[string]interface{}{"title": "My Plan"},
			want:   "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := getPlanID(tt.result)
			if got != tt.want {
				t.Errorf("getPlanID() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetPhaseID(t *testing.T) {
	tests := []struct {
		name   string
		result interface{}
		want   string
	}{
		{
			name:   "valid id",
			result: map[string]interface{}{"id": "phase_456"},
			want:   "phase_456",
		},
		{
			name:   "nil result",
			result: nil,
			want:   "",
		},
		{
			name:   "empty map",
			result: map[string]interface{}{},
			want:   "",
		},
		{
			name:   "wrong type",
			result: 123,
			want:   "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := getPhaseID(tt.result)
			if got != tt.want {
				t.Errorf("getPhaseID() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetTaskID(t *testing.T) {
	tests := []struct {
		name   string
		result interface{}
		want   string
	}{
		{
			name:   "valid task id",
			result: map[string]interface{}{"id": "task_789"},
			want:   "task_789",
		},
		{
			name:   "nil result",
			result: nil,
			want:   "",
		},
		{
			name:   "empty map",
			result: map[string]interface{}{},
			want:   "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := getTaskID(tt.result)
			if got != tt.want {
				t.Errorf("getTaskID() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetPlanProgress(t *testing.T) {
	tests := []struct {
		name   string
		result interface{}
		want   string
	}{
		{
			name: "valid progress with percent",
			result: map[string]interface{}{
				"progress": map[string]interface{}{
					"percent": 75.5,
				},
			},
			want: "76%",
		},
		{
			name: "zero percent",
			result: map[string]interface{}{
				"progress": map[string]interface{}{
					"percent": 0.0,
				},
			},
			want: "0%",
		},
		{
			name:   "nil result",
			result: nil,
			want:   "unknown",
		},
		{
			name:   "empty map",
			result: map[string]interface{}{},
			want:   "unknown",
		},
		{
			name: "progress missing",
			result: map[string]interface{}{
				"status": "active",
			},
			want: "unknown",
		},
		{
			name: "progress without percent",
			result: map[string]interface{}{
				"progress": map[string]interface{}{},
			},
			want: "unknown",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := getPlanProgress(tt.result)
			if got != tt.want {
				t.Errorf("getPlanProgress() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetActivePlanIDEmpty(t *testing.T) {
	got := getActivePlanID()
	if got != "" {
		t.Errorf("getActivePlanID() = %v, want empty string (no server)", got)
	}
}

func TestGetPlanIDFromResult(t *testing.T) {
	result := map[string]interface{}{
		"plan_id": "test_plan_123",
		"status":  "active",
	}
	got := getPlanID(result)
	want := "test_plan_123"
	if got != want {
		t.Errorf("getPlanID() = %v, want %v", got, want)
	}
}

func TestGetPhaseIDFromResult(t *testing.T) {
	result := map[string]interface{}{
		"id":     "phase_abc",
		"status": "pending",
	}
	got := getPhaseID(result)
	want := "phase_abc"
	if got != want {
		t.Errorf("getPhaseID() = %v, want %v", got, want)
	}
}

func TestGetTaskIDFromResult(t *testing.T) {
	result := map[string]interface{}{
		"id":     "task_xyz",
		"status": "completed",
	}
	got := getTaskID(result)
	want := "task_xyz"
	if got != want {
		t.Errorf("getTaskID() = %v, want %v", got, want)
	}
}
