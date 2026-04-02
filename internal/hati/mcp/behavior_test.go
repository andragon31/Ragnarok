package mcp

import (
	"context"
	"encoding/json"
	"fmt"
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

func TestPhaseCreateAndList(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()

	ctx := context.Background()

	planParams := map[string]interface{}{
		"title": "Multi-Phase Project",
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

	phaseParams := map[string]interface{}{
		"plan_id": planID,
		"title":   "Phase 1 - Setup",
	}
	phaseParamsJSON := marshalParams(phaseParams)
	phaseReq := &Request{
		Method: "phase_create",
		Params: phaseParamsJSON,
	}

	phaseResult, err := srv.handlers["phase_create"](ctx, phaseReq)
	if err != nil {
		t.Fatalf("phase_create failed: %v", err)
	}

	phaseResultMap := phaseResult.Result.(map[string]interface{})
	phaseID, ok := phaseResultMap["id"].(string)
	if !ok || phaseID == "" {
		t.Fatal("expected non-empty phase id")
	}

	if phaseResultMap["title"] != "Phase 1 - Setup" {
		t.Errorf("expected title 'Phase 1 - Setup', got %v", phaseResultMap["title"])
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

func TestPlanList(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()

	ctx := context.Background()

	for i := 1; i <= 3; i++ {
		planParams := map[string]interface{}{
			"title": fmt.Sprintf("Test Plan %d", i),
		}
		planParamsJSON := marshalParams(planParams)
		planReq := &Request{
			Method: "plan_create",
			Params: planParamsJSON,
		}
		_, err := srv.handlers["plan_create"](ctx, planReq)
		if err != nil {
			t.Fatalf("plan_create %d failed: %v", i, err)
		}
	}

	listParams := map[string]interface{}{}
	listParamsJSON := marshalParams(listParams)
	listReq := &Request{
		Method: "plan_list",
		Params: listParamsJSON,
	}

	listResult, err := srv.handlers["plan_list"](ctx, listReq)
	if err != nil {
		t.Fatalf("plan_list failed: %v", err)
	}

	listResultMap := listResult.Result.(map[string]interface{})
	plansRaw := listResultMap["plans"]
	plans, ok := plansRaw.([]*Plan)
	if !ok {
		t.Fatalf("plans type mismatch: %T", plansRaw)
	}

	if len(plans) != 3 {
		t.Errorf("expected 3 plans, got %d", len(plans))
	}
}

func TestTaskCreateAndList(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()

	ctx := context.Background()

	planParams := map[string]interface{}{
		"title": "Task Test Plan",
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

	phaseParams := map[string]interface{}{
		"plan_id":   planID,
		"title":     "Phase 1",
		"order_num": 1,
	}
	phaseParamsJSON := marshalParams(phaseParams)
	phaseReq := &Request{
		Method: "phase_create",
		Params: phaseParamsJSON,
	}
	phaseResult, err := srv.handlers["phase_create"](ctx, phaseReq)
	if err != nil {
		t.Fatalf("phase_create failed: %v", err)
	}
	phaseResultMap := phaseResult.Result.(map[string]interface{})
	phaseID := phaseResultMap["id"].(string)

	taskParams := map[string]interface{}{
		"phase_id": phaseID,
		"title":    "Test Task 1",
		"priority": 1,
	}
	taskParamsJSON := marshalParams(taskParams)
	taskReq := &Request{
		Method: "task_create",
		Params: taskParamsJSON,
	}
	taskResult, err := srv.handlers["task_create"](ctx, taskReq)
	if err != nil {
		t.Fatalf("task_create failed: %v", err)
	}
	taskResultMap := taskResult.Result.(map[string]interface{})

	if taskResultMap["title"] != "Test Task 1" {
		t.Errorf("expected title 'Test Task 1', got %v", taskResultMap["title"])
	}

	listParams := map[string]interface{}{
		"plan_id": planID,
	}
	listParamsJSON := marshalParams(listParams)
	listReq := &Request{
		Method: "task_list",
		Params: listParamsJSON,
	}

	_, err = srv.handlers["task_list"](ctx, listReq)
	if err != nil {
		t.Fatalf("task_list failed: %v", err)
	}
}

func TestTaskGetNext(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()

	ctx := context.Background()

	planParams := map[string]interface{}{
		"title": "Task GetNext Test Plan",
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

	phaseParams := map[string]interface{}{
		"plan_id":   planID,
		"title":     "Phase 1",
		"order_num": 1,
	}
	phaseParamsJSON := marshalParams(phaseParams)
	phaseReq := &Request{
		Method: "phase_create",
		Params: phaseParamsJSON,
	}
	_, err = srv.handlers["phase_create"](ctx, phaseReq)
	if err != nil {
		t.Fatalf("phase_create failed: %v", err)
	}

	getNextParams := map[string]interface{}{
		"plan_id": planID,
	}
	getNextParamsJSON := marshalParams(getNextParams)
	getNextReq := &Request{
		Method: "task_get_next",
		Params: getNextParamsJSON,
	}

	_, err = srv.handlers["task_get_next"](ctx, getNextReq)
	if err != nil {
		t.Fatalf("task_get_next failed: %v", err)
	}
}

func TestTaskUpdate(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()

	ctx := context.Background()

	planParams := map[string]interface{}{
		"title": "Task Update Test Plan",
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

	phaseParams := map[string]interface{}{
		"plan_id":   planID,
		"title":     "Phase 1",
		"order_num": 1,
	}
	phaseParamsJSON := marshalParams(phaseParams)
	phaseReq := &Request{
		Method: "phase_create",
		Params: phaseParamsJSON,
	}
	phaseResult, err := srv.handlers["phase_create"](ctx, phaseReq)
	if err != nil {
		t.Fatalf("phase_create failed: %v", err)
	}
	phaseResultMap := phaseResult.Result.(map[string]interface{})
	phaseID := phaseResultMap["id"].(string)

	taskParams := map[string]interface{}{
		"phase_id": phaseID,
		"title":    "Test Task",
		"priority": 1,
	}
	taskParamsJSON := marshalParams(taskParams)
	taskReq := &Request{
		Method: "task_create",
		Params: taskParamsJSON,
	}
	taskResult, err := srv.handlers["task_create"](ctx, taskReq)
	if err != nil {
		t.Fatalf("task_create failed: %v", err)
	}
	taskResultMap := taskResult.Result.(map[string]interface{})
	taskID := taskResultMap["id"].(string)

	updateParams := map[string]interface{}{
		"task_id": taskID,
		"status":  "in_progress",
	}
	updateParamsJSON := marshalParams(updateParams)
	updateReq := &Request{
		Method: "task_update",
		Params: updateParamsJSON,
	}

	updateResult, err := srv.handlers["task_update"](ctx, updateReq)
	if err != nil {
		t.Fatalf("task_update failed: %v", err)
	}

	updateResultMap := updateResult.Result.(map[string]interface{})
	if updateResultMap["status"] != "in_progress" {
		t.Errorf("expected status 'in_progress', got %v", updateResultMap["status"])
	}
}

func TestPlanProgress(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()

	ctx := context.Background()

	planParams := map[string]interface{}{
		"title": "Progress Test Plan",
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

	progressParams := map[string]interface{}{
		"plan_id": planID,
	}
	progressParamsJSON := marshalParams(progressParams)
	progressReq := &Request{
		Method: "plan_progress",
		Params: progressParamsJSON,
	}

	progressResult, err := srv.handlers["plan_progress"](ctx, progressReq)
	if err != nil {
		t.Fatalf("plan_progress failed: %v", err)
	}

	progressResultMap := progressResult.Result.(map[string]interface{})
	if progressResultMap["plan_id"] != planID {
		t.Errorf("expected plan_id %s, got %v", planID, progressResultMap["plan_id"])
	}
}

func TestPlanDependencies(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()

	ctx := context.Background()

	planParams := map[string]interface{}{
		"title": "Dependencies Test Plan",
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

	depsParams := map[string]interface{}{
		"plan_id": planID,
	}
	depsParamsJSON := marshalParams(depsParams)
	depsReq := &Request{
		Method: "plan_dependencies",
		Params: depsParamsJSON,
	}

	depsResult, err := srv.handlers["plan_dependencies"](ctx, depsReq)
	if err != nil {
		t.Fatalf("plan_dependencies failed: %v", err)
	}

	depsResultMap := depsResult.Result.(map[string]interface{})
	if depsResultMap["plan_id"] != planID {
		t.Errorf("expected plan_id %s, got %v", planID, depsResultMap["plan_id"])
	}
}

func TestQualitySnapshot(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()

	ctx := context.Background()

	snapshotParams := map[string]interface{}{}
	snapshotParamsJSON := marshalParams(snapshotParams)
	snapshotReq := &Request{
		Method: "quality_snapshot",
		Params: snapshotParamsJSON,
	}

	snapshotResult, err := srv.handlers["quality_snapshot"](ctx, snapshotReq)
	if err != nil {
		t.Fatalf("quality_snapshot failed: %v", err)
	}

	snapshotResultMap := snapshotResult.Result.(map[string]interface{})
	if snapshotResultMap["source"] != "tyr" {
		t.Errorf("expected source 'tyr', got %v", snapshotResultMap["source"])
	}
}

func TestSpecImpact(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()

	ctx := context.Background()

	planParams := map[string]interface{}{
		"title": "Spec Impact Test Plan",
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

	impactParams := map[string]interface{}{
		"plan_id": planID,
	}
	impactParamsJSON := marshalParams(impactParams)
	impactReq := &Request{
		Method: "spec_impact",
		Params: impactParamsJSON,
	}

	impactResult, err := srv.handlers["spec_impact"](ctx, impactReq)
	if err != nil {
		t.Fatalf("spec_impact failed: %v", err)
	}

	impactResultMap := impactResult.Result.(map[string]interface{})
	if impactResultMap["plan_id"] != planID {
		t.Errorf("expected plan_id %s, got %v", planID, impactResultMap["plan_id"])
	}
}

func TestPhaseUpdate(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()

	ctx := context.Background()

	planParams := map[string]interface{}{
		"title": "Phase Update Test Plan",
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

	phaseParams := map[string]interface{}{
		"plan_id":   planID,
		"title":     "Phase 1",
		"order_num": 1,
	}
	phaseParamsJSON := marshalParams(phaseParams)
	phaseReq := &Request{
		Method: "phase_create",
		Params: phaseParamsJSON,
	}
	phaseResult, err := srv.handlers["phase_create"](ctx, phaseReq)
	if err != nil {
		t.Fatalf("phase_create failed: %v", err)
	}
	phaseResultMap := phaseResult.Result.(map[string]interface{})
	phaseID := phaseResultMap["id"].(string)

	updateParams := map[string]interface{}{
		"phase_id": phaseID,
		"status":   "completed",
	}
	updateParamsJSON := marshalParams(updateParams)
	updateReq := &Request{
		Method: "phase_update",
		Params: updateParamsJSON,
	}

	updateResult, err := srv.handlers["phase_update"](ctx, updateReq)
	if err != nil {
		t.Fatalf("phase_update failed: %v", err)
	}

	updateResultMap := updateResult.Result.(map[string]interface{})
	if updateResultMap["status"] != "completed" {
		t.Errorf("expected status 'completed', got %v", updateResultMap["status"])
	}
}

func TestPlanActivate(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()

	ctx := context.Background()

	planParams := map[string]interface{}{
		"title": "Activate Test Plan",
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

	activateParams := map[string]interface{}{
		"plan_id": planID,
	}
	activateParamsJSON := marshalParams(activateParams)
	activateReq := &Request{
		Method: "plan_activate",
		Params: activateParamsJSON,
	}

	activateResult, err := srv.handlers["plan_activate"](ctx, activateReq)
	if err != nil {
		t.Fatalf("plan_activate failed: %v", err)
	}

	activateResultMap := activateResult.Result.(map[string]interface{})
	if activateResultMap["status"] != "active" {
		t.Errorf("expected status 'active', got %v", activateResultMap["status"])
	}
}
