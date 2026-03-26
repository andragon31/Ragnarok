package mcp

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/ragnarok-ecosystem/hati/internal/config"
)

type Server struct {
	port     int
	config   *config.Config
	db       *sql.DB
	handlers map[string]ToolHandler
}

type ToolHandler func(ctx context.Context, req *Request) (*Response, error)

type Request struct {
	Method string          `json:"method"`
	Params json.RawMessage `json:"params,omitempty"`
	ID     string          `json:"id,omitempty"`
}

type Response struct {
	Result interface{} `json:"result,omitempty"`
	Error  *Error      `json:"error,omitempty"`
	ID     string      `json:"id,omitempty"`
}

type Error struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

func NewServer(cfg *config.Config, db *sql.DB) *Server {
	s := &Server{
		port:     cfg.Port,
		config:   cfg,
		db:       db,
		handlers: make(map[string]ToolHandler),
	}
	s.registerHandlers()
	return s
}

func (s *Server) registerHandlers() {
	s.handlers["plan_create"] = s.handlePlanCreate
	s.handlers["plan_get"] = s.handlePlanGet
	s.handlers["plan_list"] = s.handlePlanList
	s.handlers["plan_revise"] = s.handlePlanRevise
	s.handlers["plan_abandon"] = s.handlePlanAbandon
	s.handlers["plan_complete"] = s.handlePlanComplete
	s.handlers["plan_completeness"] = s.handlePlanCompleteness
	s.handlers["plan_quality"] = s.handlePlanQuality
	s.handlers["plan_restart"] = s.handlePlanRestart
	s.handlers["plan_resume"] = s.handlePlanResume
	s.handlers["plan_blockers"] = s.handlePlanBlockers

	s.handlers["checkpoint_open"] = s.handleCheckpointOpen
	s.handlers["checkpoint_decide"] = s.handleCheckpointDecide
	s.handlers["checkpoint_status"] = s.handleCheckpointStatus
	s.handlers["checkpoint_approve"] = s.handleCheckpointApprove

	s.handlers["phase_start"] = s.handlePhaseStart
	s.handlers["phase_report"] = s.handlePhaseReport

	s.handlers["feedback_request"] = s.handleFeedbackRequest
	s.handlers["feedback_receive"] = s.handleFeedbackReceive
	s.handlers["feedback_escalate"] = s.handleFeedbackEscalate

	s.handlers["record_list"] = s.handleRecordList
	s.handlers["record_get"] = s.handleRecordGet
	s.handlers["record_export"] = s.handleRecordExport

	s.handlers["module_hints"] = s.handleModuleHints
	s.handlers["spec_impact"] = s.handleSpecImpact

	s.handlers["quality_snapshot"] = s.handleQualitySnapshot
	s.handlers["learning_answer"] = s.handleLearningAnswer

	s.handlers["hati_status"] = s.handleHatiStatus
	s.handlers["hati_stats"] = s.handleHatiStats
	s.handlers["hati_commit_info"] = s.handleHatiCommitInfo
	s.handlers["hati_register_commit"] = s.handleHatiRegisterCommit
}

func (s *Server) HandleRequest(ctx context.Context, raw []byte) ([]byte, error) {
	var req Request
	if err := json.Unmarshal(raw, &req); err != nil {
		return s.errorResponse(req.ID, -32700, "Parse error: "+err.Error())
	}

	handler, ok := s.handlers[req.Method]
	if !ok {
		return s.errorResponse(req.ID, -32601, fmt.Sprintf("Method not found: %s", req.Method))
	}

	result, err := handler(ctx, &req)
	if err != nil {
		return s.errorResponse(req.ID, -32603, "Internal error: "+err.Error())
	}

	resp := &Response{
		Result: result,
		ID:     req.ID,
	}
	return json.Marshal(resp)
}

func (s *Server) errorResponse(id string, code int, msg string) ([]byte, error) {
	resp := &Response{
		Error: &Error{Code: code, Message: msg},
		ID:    id,
	}
	return json.Marshal(resp)
}

func (s *Server) RunStdio(ctx context.Context) error {
	log.Printf("Hati MCP server running on stdio")

	stdin := os.NewFile(uintptr(os.Stdin.Fd()), "stdin")
	stdout := os.NewFile(uintptr(os.Stdout.Fd()), "stdout")
	decoder := json.NewDecoder(stdin)
	encoder := json.NewEncoder(stdout)

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			var raw json.RawMessage
			if err := decoder.Decode(&raw); err != nil {
				log.Printf("Decode error: %v", err)
				continue
			}

			resp, err := s.HandleRequest(ctx, raw)
			if err != nil {
				log.Printf("Handle error: %v", err)
				continue
			}

			if err := encoder.Encode(resp); err != nil {
				log.Printf("Encode error: %v", err)
			}
		}
	}
}

func (s *Server) RunTCPServer(ctx context.Context) error {
	addr := fmt.Sprintf(":%d", s.port)
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("failed to listen on %s: %w", addr, err)
	}
	defer listener.Close()

	log.Printf("Hati MCP server listening on %s (TCP)", addr)

	mux := http.NewServeMux()
	mux.HandleFunc("/mcp", s.handleHTTPRequest)
	mux.HandleFunc("/health", s.handleHealth)
	mux.HandleFunc("/", s.handleRoot)

	server := &http.Server{
		Handler: mux,
	}

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		<-ctx.Done()
		server.Close()
	}()

	err = server.Serve(listener)
	if err != nil && !strings.Contains(err.Error(), "Server closed") {
		return err
	}
	wg.Wait()
	return nil
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status": "healthy",
		"server": "hati",
		"port":   s.port,
	})
}

func (s *Server) handleRoot(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"name":    "Hati",
		"version": "1.1.0",
		"status":  "running",
		"mcp":     "/mcp",
	})
}

func (s *Server) handleHTTPRequest(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

	if r.Method == "OPTIONS" {
		return
	}

	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req Request
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		errResp, _ := s.errorResponse("", -32700, "Parse error: "+err.Error())
		w.Write(errResp)
		return
	}

	ctx := context.Background()
	resp, err := s.HandleRequest(ctx, req.Params)
	if err != nil {
		errResp, _ := s.errorResponse(req.ID, -32603, "Internal error: "+err.Error())
		w.Write(errResp)
		return
	}

	w.Write(resp)
}

func (s *Server) Run(ctx context.Context) error {
	addr := fmt.Sprintf(":%d", s.port)
	log.Printf("Hati MCP server running on %s", addr)

	if os.Getenv("MCP_TRANSPORT") == "tcp" {
		return s.RunTCPServer(ctx)
	}

	return s.RunStdio(ctx)
}

type Plan struct {
	ID              string    `json:"id"`
	SessionID       string    `json:"session_id,omitempty"`
	Title           string    `json:"title"`
	Description     string    `json:"description,omitempty"`
	Status          string    `json:"status"`
	RiskLevel       string    `json:"risk_level"`
	SpecImpact      string    `json:"spec_impact,omitempty"`
	ModuleHintsUsed string    `json:"module_hints_used,omitempty"`
	QualitySource   string    `json:"quality_source"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
	CompletedAt     time.Time `json:"completed_at,omitempty"`
}

type Phase struct {
	ID              string    `json:"id"`
	PlanID          string    `json:"plan_id"`
	Name            string    `json:"name"`
	Description     string    `json:"description,omitempty"`
	RiskLevel       string    `json:"risk_level"`
	Status          string    `json:"status"`
	OrderNum        int       `json:"order_num"`
	AgentsMdHints   string    `json:"agents_md_hints,omitempty"`
	SpecIDsAffected string    `json:"spec_ids_affected,omitempty"`
	Module          string    `json:"module,omitempty"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

type Checkpoint struct {
	ID              string    `json:"id"`
	PlanID          string    `json:"plan_id,omitempty"`
	PhaseID         string    `json:"phase_id,omitempty"`
	Type            string    `json:"type"`
	Status          string    `json:"status"`
	CanContinue     bool      `json:"can_continue"`
	RiskLevel       string    `json:"risk_level,omitempty"`
	SpecDelta       string    `json:"spec_delta,omitempty"`
	QualitySnapshot string    `json:"quality_snapshot,omitempty"`
	CreatedAt       time.Time `json:"created_at"`
	DecidedAt       time.Time `json:"decided_at,omitempty"`
	DecidedBy       string    `json:"decided_by,omitempty"`
	Feedback        string    `json:"feedback,omitempty"`
}

type ApprovalRecord struct {
	ID         string    `json:"id"`
	PlanID     string    `json:"plan_id"`
	Decision   string    `json:"decision"`
	Approver   string    `json:"approver,omitempty"`
	Notes      string    `json:"notes,omitempty"`
	SpecDeltas string    `json:"spec_deltas,omitempty"`
	CreatedAt  time.Time `json:"created_at"`
}

type Feedback struct {
	ID           string    `json:"id"`
	CheckpointID string    `json:"checkpoint_id"`
	Type         string    `json:"type"`
	Content      string    `json:"content"`
	Author       string    `json:"author,omitempty"`
	CreatedAt    time.Time `json:"created_at"`
}

var idCounter = 0

func generateID(prefix string) string {
	idCounter++
	return fmt.Sprintf("%s_%d_%d", prefix, time.Now().UnixNano(), idCounter)
}
