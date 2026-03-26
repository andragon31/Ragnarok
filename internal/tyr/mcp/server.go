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

	"github.com/andragon31/Ragnarok/internal/tyr/config"
	rootmcp "github.com/andragon31/Ragnarok/internal/mcp"
)

type Request = rootmcp.Request
type Response = rootmcp.Response
type Error = rootmcp.Error
type ToolHandler = rootmcp.ToolHandler

type Server struct {
	port     int
	config   *config.Config
	db       *sql.DB
	handlers map[string]ToolHandler
}

func (s *Server) Handlers() map[string]ToolHandler {
	return s.handlers
}

type PkgCheckResult struct {
	Name              string    `json:"name"`
	Ecosystem         string    `json:"ecosystem"`
	Version           string    `json:"version,omitempty"`
	Exists            bool      `json:"exists"`
	Trusted           bool      `json:"trusted"`
	CVECount          int       `json:"cve_count"`
	AgeDays           int       `json:"age_days"`
	DownloadsMonthly  int       `json:"downloads_monthly"`
	TyposquattingRisk bool      `json:"typosquatting_risk"`
	LastChecked       time.Time `json:"last_checked"`
}

type SASTFinding struct {
	ID        string    `json:"id"`
	RuleID    string    `json:"rule_id"`
	Severity  string    `json:"severity"`
	File      string    `json:"file"`
	Line      int       `json:"line"`
	Message   string    `json:"message"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
}

type AuditEntry struct {
	ID         string    `json:"id"`
	SessionID  string    `json:"session_id,omitempty"`
	Tool       string    `json:"tool"`
	ActionType string    `json:"action_type"`
	Target     string    `json:"target"`
	RiskLevel  string    `json:"risk_level"`
	Result     string    `json:"result"`
	CreatedAt  time.Time `json:"created_at"`
}

type Standard struct {
	ID          string  `json:"id"`
	Name        string  `json:"name"`
	Description string  `json:"description"`
	Category    string  `json:"category"`
	LastResult  string  `json:"last_result,omitempty"`
	PassRate    float64 `json:"pass_rate,omitempty"`
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
	s.handlers["pkg_check"] = s.handlePkgCheck
	s.handlers["pkg_license"] = s.handlePkgLicense
	s.handlers["pkg_audit"] = s.handlePkgAudit
	s.handlers["pkg_audit_snapshot"] = s.handlePkgAuditSnapshot
	s.handlers["pkg_audit_continuous"] = s.handlePkgAuditContinuous

	s.handlers["sast_run"] = s.handleSastRun
	s.handlers["sast_findings"] = s.handleSastFindings
	s.handlers["sast_resolve"] = s.handleSastResolve

	s.handlers["audit_log"] = s.handleAuditLog
	s.handlers["session_audit"] = s.handleSessionAudit
	s.handlers["inject_guard"] = s.handleInjectGuard
	s.handlers["proactive_scan"] = s.handleProactiveScan
	s.handlers["sanitize"] = s.handleSanitize

	s.handlers["standard_run"] = s.handleStandardRun
	s.handlers["standard_run_all"] = s.handleStandardRunAll
	s.handlers["standard_list"] = s.handleStandardList
	s.handlers["quality_snapshot"] = s.handleQualitySnapshot

	s.handlers["scope_violations"] = s.handleScopeViolations
	s.handlers["tyr_stats"] = s.handleTyrStats

	s.handlers["precommit_validate"] = s.handlePrecommitValidate
	s.handlers["precommit_autofix"] = s.handlePrecommitAutofix
	s.handlers["bootstrap_import"] = s.handleBootstrapImport
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
	log.Printf("Tyr MCP server running on stdio")

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

	log.Printf("Tyr MCP server listening on %s (TCP)", addr)

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
		"server": "tyr",
		"port":   s.port,
	})
}

func (s *Server) handleRoot(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"name":    "Tyr",
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
	log.Printf("Tyr MCP server running on %s", addr)

	if os.Getenv("MCP_TRANSPORT") == "tcp" {
		return s.RunTCPServer(ctx)
	}

	return s.RunStdio(ctx)
}

var idCounter = 0

func generateID(prefix string) string {
	idCounter++
	return fmt.Sprintf("%s_%d_%d", prefix, time.Now().UnixNano(), idCounter)
}
