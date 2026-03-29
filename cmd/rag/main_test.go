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

func TestContains(t *testing.T) {
	tests := []struct {
		name   string
		s      string
		substr string
		want   bool
	}{
		{"empty substr", "hello world", "", true},
		{"full match", "hello", "hello", true},
		{"partial match", "hello world", "world", true},
		{"no match", "hello", "xyz", false},
		{"case sensitive", "Hello", "hello", false},
		{"substr at start", "hello world", "hello", true},
		{"substr at end", "hello world", "world", true},
		{"empty string", "", "", true},
		{"substr longer", "hi", "hello", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := contains(tt.s, tt.substr)
			if got != tt.want {
				t.Errorf("contains(%q, %q) = %v, want %v", tt.s, tt.substr, got, tt.want)
			}
		})
	}
}

func TestGetPluginPort(t *testing.T) {
	tests := []struct {
		plugin string
		want   int
	}{
		{"fenrir", 7437},
		{"hati", 7439},
		{"skoll", 7438},
		{"tyr", 7440},
		{"unknown", 0},
		{"", 0},
	}

	for _, tt := range tests {
		t.Run(tt.plugin, func(t *testing.T) {
			got := getPluginPort(tt.plugin)
			if got != tt.want {
				t.Errorf("getPluginPort(%q) = %v, want %v", tt.plugin, got, tt.want)
			}
		})
	}
}

func TestGetPlugin(t *testing.T) {
	tests := []struct {
		name string
		want *Plugin
	}{
		{"fenrir", &Plugin{Name: "fenrir", Port: 7437, DataDir: "~/.fenrir", BinName: "fenrir"}},
		{"hati", &Plugin{Name: "hati", Port: 7439, DataDir: "~/.hati", BinName: "hati"}},
		{"skoll", &Plugin{Name: "skoll", Port: 7438, DataDir: "~/.skoll", BinName: "skoll"}},
		{"tyr", &Plugin{Name: "tyr", Port: 7440, DataDir: "~/.tyr", BinName: "tyr"}},
		{"unknown", nil},
		{"", nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := getPlugin(tt.name)
			if tt.want == nil {
				if got != nil {
					t.Errorf("getPlugin(%q) = %v, want nil", tt.name, got)
				}
			} else {
				if got == nil {
					t.Errorf("getPlugin(%q) = nil, want %v", tt.name, tt.want)
				} else if got.Name != tt.want.Name {
					t.Errorf("getPlugin(%q).Name = %v, want %v", tt.name, got.Name, tt.want.Name)
				}
			}
		})
	}
}

func TestGetStringResult(t *testing.T) {
	tests := []struct {
		name   string
		result interface{}
		key    string
		want   string
	}{
		{
			name:   "valid string value",
			result: map[string]interface{}{"name": "test_value"},
			key:    "name",
			want:   "test_value",
		},
		{
			name:   "missing key",
			result: map[string]interface{}{"other": "value"},
			key:    "name",
			want:   "",
		},
		{
			name:   "nil result",
			result: nil,
			key:    "name",
			want:   "",
		},
		{
			name:   "empty map",
			result: map[string]interface{}{},
			key:    "name",
			want:   "",
		},
		{
			name:   "wrong type for key",
			result: map[string]interface{}{"name": 123},
			key:    "name",
			want:   "",
		},
		{
			name:   "non-map result",
			result: "string",
			key:    "name",
			want:   "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := getStringResult(tt.result, tt.key)
			if got != tt.want {
				t.Errorf("getStringResult(%v, %q) = %v, want %v", tt.result, tt.key, got, tt.want)
			}
		})
	}
}

func TestPrintWorkflowResult(t *testing.T) {
	tests := []struct {
		name     string
		workflow string
		result   interface{}
	}{
		{
			name:     "nil result does nothing",
			workflow: "test",
			result:   nil,
		},
		{
			name:     "empty map does nothing",
			workflow: "test",
			result:   map[string]interface{}{},
		},
		{
			name:     "valid result with status",
			workflow: "test",
			result: map[string]interface{}{
				"status":  "completed",
				"message": "done",
				"steps":   []interface{}{"step1", "step2"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			printWorkflowResult(tt.workflow, tt.result)
		})
	}
}

func TestGetStringResultFromMap(t *testing.T) {
	result := map[string]interface{}{
		"id":     "test_id_123",
		"name":   "Test Plan",
		"status": "active",
	}

	if got := getStringResult(result, "id"); got != "test_id_123" {
		t.Errorf("getStringResult() id = %v, want %v", got, "test_id_123")
	}
	if got := getStringResult(result, "name"); got != "Test Plan" {
		t.Errorf("getStringResult() name = %v, want %v", got, "Test Plan")
	}
	if got := getStringResult(result, "status"); got != "active" {
		t.Errorf("getStringResult() status = %v, want %v", got, "active")
	}
	if got := getStringResult(result, "nonexistent"); got != "" {
		t.Errorf("getStringResult() nonexistent = %v, want empty", got)
	}
}

func TestPluginStats(t *testing.T) {
	stats := PluginStats{
		Name:      "fenrir",
		Status:    "online",
		Port:      7437,
		LatencyMs: 5,
		Data:      map[string]interface{}{"total_observations": 42},
	}

	if stats.Name != "fenrir" {
		t.Errorf("PluginStats.Name = %v, want %v", stats.Name, "fenrir")
	}
	if stats.Status != "online" {
		t.Errorf("PluginStats.Status = %v, want %v", stats.Status, "online")
	}
	if stats.Port != 7437 {
		t.Errorf("PluginStats.Port = %v, want %v", stats.Port, 7437)
	}
	if stats.LatencyMs != 5 {
		t.Errorf("PluginStats.LatencyMs = %v, want %v", stats.LatencyMs, 5)
	}
	if stats.Data["total_observations"] != 42 {
		t.Errorf("PluginStats.Data[total_observations] = %v, want %v", stats.Data["total_observations"], 42)
	}
}

func TestEcosystemStats(t *testing.T) {
	stats := EcosystemStats{
		Fenrir: &PluginStats{Name: "fenrir", Status: "online"},
		Hati:   &PluginStats{Name: "hati", Status: "offline"},
		Skoll:  &PluginStats{Name: "skoll", Status: "online"},
		Tyr:    &PluginStats{Name: "tyr", Status: "online"},
	}

	if stats.Fenrir.Status != "online" {
		t.Errorf("EcosystemStats.Fenrir.Status = %v, want %v", stats.Fenrir.Status, "online")
	}
	if stats.Hati.Status != "offline" {
		t.Errorf("EcosystemStats.Hati.Status = %v, want %v", stats.Hati.Status, "offline")
	}
	if stats.Skoll.Status != "online" {
		t.Errorf("EcosystemStats.Skoll.Status = %v, want %v", stats.Skoll.Status, "online")
	}
	if stats.Tyr.Status != "online" {
		t.Errorf("EcosystemStats.Tyr.Status = %v, want %v", stats.Tyr.Status, "online")
	}
}

func TestVersion(t *testing.T) {
	if version == "" {
		t.Error("version should not be empty")
	}
	if version != "2.4.6" {
		t.Errorf("version = %v, want %v", version, "2.4.6")
	}
}

func TestAllPlugins(t *testing.T) {
	expectedPlugins := []string{"fenrir", "hati", "skoll", "tyr"}
	if len(allPlugins) != len(expectedPlugins) {
		t.Errorf("len(allPlugins) = %v, want %v", len(allPlugins), len(expectedPlugins))
	}

	for i, expected := range expectedPlugins {
		if allPlugins[i].Name != expected {
			t.Errorf("allPlugins[%d].Name = %v, want %v", i, allPlugins[i].Name, expected)
		}
	}

	for _, plugin := range allPlugins {
		if plugin.Port == 0 {
			t.Errorf("Plugin %s has Port 0", plugin.Name)
		}
		if plugin.DataDir == "" {
			t.Errorf("Plugin %s has empty DataDir", plugin.Name)
		}
		if plugin.BinName == "" {
			t.Errorf("Plugin %s has empty BinName", plugin.Name)
		}
	}
}

func TestMCPJson(t *testing.T) {
	mcpJson := MCPJson{
		MCPServers: map[string]MCPServer{
			"fenrir": {
				Command: "fenrir",
				Args:    []string{"serve"},
				Env:     map[string]string{"KEY": "value"},
			},
		},
	}

	if len(mcpJson.MCPServers) != 1 {
		t.Errorf("len(MCPJson.MCPServers) = %v, want %v", len(mcpJson.MCPServers), 1)
	}

	fenrirServer := mcpJson.MCPServers["fenrir"]
	if fenrirServer.Command != "fenrir" {
		t.Errorf("MCPServer.Command = %v, want %v", fenrirServer.Command, "fenrir")
	}
	if len(fenrirServer.Args) != 1 || fenrirServer.Args[0] != "serve" {
		t.Errorf("MCPServer.Args = %v, want %v", fenrirServer.Args, []string{"serve"})
	}
	if fenrirServer.Env["KEY"] != "value" {
		t.Errorf("MCPServer.Env[KEY] = %v, want %v", fenrirServer.Env["KEY"], "value")
	}
}

func TestImportRecordQueryGeneration(t *testing.T) {
	tests := []struct {
		name      string
		tableName string
		record    map[string]interface{}
		wantCols  int
		wantErr   bool
	}{
		{
			name:      "single column",
			tableName: "test_table",
			record: map[string]interface{}{
				"id":   1,
				"name": "test",
			},
			wantCols: 2,
			wantErr:  false,
		},
		{
			name:      "empty record",
			tableName: "test_table",
			record:    map[string]interface{}{},
			wantCols:  0,
			wantErr:   true,
		},
		{
			name:      "record with nil values",
			tableName: "test_table",
			record: map[string]interface{}{
				"id":   1,
				"name": nil,
			},
			wantCols: 2,
			wantErr:  false,
		},
		{
			name:      "record with various types",
			tableName: "test_table",
			record: map[string]interface{}{
				"id":        1,
				"name":      "test",
				"count":     float64(42),
				"is_active": true,
			},
			wantCols: 4,
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.wantErr && len(tt.record) == 0 {
				return
			}

			columns := make([]string, 0, len(tt.record))
			placeholders := make([]string, 0, len(tt.record))

			for col := range tt.record {
				columns = append(columns, col)
				placeholders = append(placeholders, "?")
			}

			if len(columns) != tt.wantCols {
				t.Errorf("columns count = %v, want %v", len(columns), tt.wantCols)
			}

			if len(placeholders) != tt.wantCols {
				t.Errorf("placeholders count = %v, want %v", len(placeholders), tt.wantCols)
			}
		})
	}
}

func TestFetchPluginStats(t *testing.T) {
	stats := fetchPluginStats("unknown_plugin", 0)
	if stats.Status != "unknown" {
		t.Errorf("fetchPluginStats with port 0: Status = %v, want %v", stats.Status, "unknown")
	}
	if stats.Port != 0 {
		t.Errorf("fetchPluginStats with port 0: Port = %v, want %v", stats.Port, 0)
	}
}

func TestPrintPluginStats(t *testing.T) {
	stats := &PluginStats{
		Name:      "test",
		Status:    "online",
		LatencyMs: 10,
		Data:      map[string]interface{}{"key": "value"},
	}
	printPluginStats(stats)
}

func TestPrintUnifiedStats(t *testing.T) {
	stats := &EcosystemStats{
		Fenrir: &PluginStats{Name: "fenrir", Status: "online", LatencyMs: 5},
		Hati:   &PluginStats{Name: "hati", Status: "offline"},
		Skoll:  &PluginStats{Name: "skoll", Status: "online", LatencyMs: 3},
		Tyr:    &PluginStats{Name: "tyr", Status: "online", LatencyMs: 7},
	}
	printUnifiedStats(stats)
}

func TestPluginStruct(t *testing.T) {
	p := Plugin{
		Name:    "test",
		Port:    1234,
		DataDir: "/tmp/data",
		BinName: "test.bin",
	}

	if p.Name != "test" {
		t.Errorf("Plugin.Name = %v, want %v", p.Name, "test")
	}
	if p.Port != 1234 {
		t.Errorf("Plugin.Port = %v, want %v", p.Port, 1234)
	}
	if p.DataDir != "/tmp/data" {
		t.Errorf("Plugin.DataDir = %v, want %v", p.DataDir, "/tmp/data")
	}
	if p.BinName != "test.bin" {
		t.Errorf("Plugin.BinName = %v, want %v", p.BinName, "test.bin")
	}
}
