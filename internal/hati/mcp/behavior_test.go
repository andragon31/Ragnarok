package mcp

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/andragon31/Ragnarok/internal/hati/config"
	"github.com/andragon31/Ragnarok/internal/hati/database"
	_ "modernc.org/sqlite"
)

func setupTestServer(t *testing.T) (*Server, func()) {
	tmpDir := filepath.Join(os.TempDir(), "ragnarok_hati_mcp_test_"+randomID())
	os.MkdirAll(tmpDir, 0755)

	cfg := &config.Config{
		Port:    17439,
		DataDir: tmpDir,
	}

	dbPath := filepath.Join(tmpDir, "hati.db")
	db, err := database.NewDB(dbPath)
	if err != nil {
		os.RemoveAll(tmpDir)
		t.Fatalf("failed to open database: %v", err)
	}

	if err := database.InitSchema(db); err != nil {
		db.Close()
		os.RemoveAll(tmpDir)
		t.Fatalf("failed to init schema: %v", err)
	}

	srv := NewServer(cfg, db)

	cleanup := func() {
		db.Close()
		os.RemoveAll(tmpDir)
	}

	return srv, cleanup
}

func TestPlanCreateAndRetrieve(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()

	ctx := context.Background()

	params := map[string]interface{}{
		"title":       "Test Project Plan",
		"description": "A comprehensive test plan",
		"risk_level":  "medium",
	}
	paramsJSON := marshalParams(params)
	req := &Request{
		Method: "plan_create",
		Params: paramsJSON,
	}

	result, err := srv.handlers["plan_create"](ctx, req)
	if err != nil {
		t.Fatalf("plan_create failed: %v", err)
	}

	resultMap := result.Result.(map[string]interface{})

	planID, ok := resultMap["id"].(string)
	if !ok || planID == "" {
		t.Fatal("expected non-empty plan id")
	}

	if resultMap["title"] != "Test Project Plan" {
		t.Errorf("expected title 'Test Project Plan', got %v", resultMap["title"])
	}
	if resultMap["status"] != "draft" {
		t.Errorf("expected status 'draft', got %v", resultMap["status"])
	}

	getParams := map[string]interface{}{
		"plan_id": planID,
	}
	getParamsJSON := marshalParams(getParams)
	getReq := &Request{
		Method: "plan_get",
		Params: getParamsJSON,
	}

	getResult, err := srv.handlers["plan_get"](ctx, getReq)
	if err != nil {
		t.Fatalf("plan_get failed: %v", err)
	}

	getResultMap := resultToMap(getResult.Result)

	if getResultMap["title"] != "Test Project Plan" {
		t.Errorf("retrieved plan title mismatch: expected 'Test Project Plan', got %v", getResultMap["title"])
	}
	if getResultMap["description"] != "A comprehensive test plan" {
		t.Errorf("retrieved plan description mismatch")
	}
}

func TestPlanCreateWithPhasesViaRevise(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()

	ctx := context.Background()

	params := map[string]interface{}{
		"title":       "Multi-Phase Project",
		"description": "Project with multiple phases",
		"risk_level":  "high",
	}
	paramsJSON := marshalParams(params)
	req := &Request{
		Method: "plan_create",
		Params: paramsJSON,
	}

	result, err := srv.handlers["plan_create"](ctx, req)
	if err != nil {
		t.Fatalf("plan_create failed: %v", err)
	}

	resultMap := result.Result.(map[string]interface{})
	planID := resultMap["id"].(string)

	reviseParams := map[string]interface{}{
		"plan_id":    planID,
		"new_phases": []string{"Phase 1", "Phase 2", "Phase 3"},
		"notes":      "Adding phases via revise",
	}
	reviseParamsJSON := marshalParams(reviseParams)
	reviseReq := &Request{
		Method: "plan_revise",
		Params: reviseParamsJSON,
	}

	reviseResult, err := srv.handlers["plan_revise"](ctx, reviseReq)
	if err != nil {
		t.Fatalf("plan_revise failed: %v", err)
	}

	reviseResultMap := reviseResult.Result.(map[string]interface{})

	if reviseResultMap["status"] != "needs_revision" {
		t.Errorf("expected status 'needs_revision', got %v", reviseResultMap["status"])
	}

	planRevisionsQuery := `SELECT COUNT(*) FROM plan_revisions WHERE plan_id = ?`
	var count int
	if err := srv.db.QueryRow(planRevisionsQuery, planID).Scan(&count); err != nil {
		t.Fatalf("failed to count revisions: %v", err)
	}
	if count != 1 {
		t.Errorf("expected 1 revision, got %d", count)
	}
}

func TestPlanRevise(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()

	ctx := context.Background()

	planParams := map[string]interface{}{
		"title": "Original Title",
	}
	planParamsJSON := marshalParams(planParams)
	planReq := &Request{
		Method: "plan_create",
		Params: planParamsJSON,
	}

	planResult, err := srv.handlers["plan_create"](ctx, planReq)
	if err != nil {
		t.Fatalf("plan_create failed: %v", err)
	}
	planResultMap := planResult.Result.(map[string]interface{})
	planID := planResultMap["id"].(string)

	reviseParams := map[string]interface{}{
		"plan_id": planID,
		"title":   "Revised Title",
		"notes":   "Updating the plan title",
	}
	reviseParamsJSON := marshalParams(reviseParams)
	reviseReq := &Request{
		Method: "plan_revise",
		Params: reviseParamsJSON,
	}

	reviseResult, err := srv.handlers["plan_revise"](ctx, reviseReq)
	if err != nil {
		t.Fatalf("plan_revise failed: %v", err)
	}

	reviseResultMap := reviseResult.Result.(map[string]interface{})

	if reviseResultMap["status"] != "needs_revision" {
		t.Errorf("expected status 'needs_revision', got %v", reviseResultMap["status"])
	}

	getParams := map[string]interface{}{
		"plan_id": planID,
	}
	getParamsJSON := marshalParams(getParams)
	getReq := &Request{
		Method: "plan_get",
		Params: getParamsJSON,
	}

	getResult, err := srv.handlers["plan_get"](ctx, getReq)
	if err != nil {
		t.Fatalf("plan_get failed: %v", err)
	}

	getResultMap := resultToMap(getResult.Result)

	if getResultMap["status"] != "needs_revision" {
		t.Errorf("plan status should be 'needs_revision' after revise, got %v", getResultMap["status"])
	}
}

func TestPlanAbandon(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()

	ctx := context.Background()

	planParams := map[string]interface{}{
		"title": "Abandon Test Plan",
	}
	planParamsJSON := marshalParams(planParams)
	planReq := &Request{
		Method: "plan_create",
		Params: planParamsJSON,
	}

	planResult, err := srv.handlers["plan_create"](ctx, planReq)
	if err != nil {
		t.Fatalf("plan_create failed: %v", err)
	}
	planResultMap := planResult.Result.(map[string]interface{})
	planID := planResultMap["id"].(string)

	abandonParams := map[string]interface{}{
		"plan_id": planID,
		"reason":  "Testing abandon functionality",
	}
	abandonParamsJSON := marshalParams(abandonParams)
	abandonReq := &Request{
		Method: "plan_abandon",
		Params: abandonParamsJSON,
	}

	abandonResult, err := srv.handlers["plan_abandon"](ctx, abandonReq)
	if err != nil {
		t.Fatalf("plan_abandon failed: %v", err)
	}

	abandonResultMap := abandonResult.Result.(map[string]interface{})

	if abandonResultMap["status"] != "abandoned" {
		t.Errorf("expected status 'abandoned', got %v", abandonResultMap["status"])
	}
}

func TestPlanComplete(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()

	ctx := context.Background()

	planParams := map[string]interface{}{
		"title": "Complete Test Plan",
	}
	planParamsJSON := marshalParams(planParams)
	planReq := &Request{
		Method: "plan_create",
		Params: planParamsJSON,
	}

	planResult, err := srv.handlers["plan_create"](ctx, planReq)
	if err != nil {
		t.Fatalf("plan_create failed: %v", err)
	}
	planResultMap := planResult.Result.(map[string]interface{})
	planID := planResultMap["id"].(string)

	completeParams := map[string]interface{}{
		"plan_id": planID,
	}
	completeParamsJSON := marshalParams(completeParams)
	completeReq := &Request{
		Method: "plan_complete",
		Params: completeParamsJSON,
	}

	completeResult, err := srv.handlers["plan_complete"](ctx, completeReq)
	if err != nil {
		t.Fatalf("plan_complete failed: %v", err)
	}

	completeResultMap := completeResult.Result.(map[string]interface{})

	if completeResultMap["status"] != "completed" {
		t.Errorf("expected status 'completed', got %v", completeResultMap["status"])
	}
	if completeResultMap["completed_at"] == nil {
		t.Error("expected completed_at to be set")
	}
}

func marshalParams(params map[string]interface{}) []byte {
	b, _ := json.Marshal(params)
	return b
}

func randomID() string {
	return itoa(time.Now().UnixNano())
}

func itoa(i int64) string {
	if i == 0 {
		return "0"
	}
	if i < 0 {
		return "-" + itoa(-i)
	}
	digits := ""
	for i > 0 {
		digits = string(rune('0'+i%10)) + digits
		i /= 10
	}
	return digits
}

func resultToMap(result any) map[string]any {
	if m, ok := result.(map[string]any); ok {
		return m
	}
	b, _ := json.Marshal(result)
	var m map[string]any
	json.Unmarshal(b, &m)
	return m
}
