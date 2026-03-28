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
	"path/filepath"
	"strings"
	"sync"
	"time"

	rootmcp "github.com/andragon31/Ragnarok/internal/mcp"
	"github.com/andragon31/Ragnarok/internal/skoll/config"
	"github.com/andragon31/Ragnarok/internal/skoll/skills"
)

type Request = rootmcp.Request
type Response = rootmcp.Response
type Error = rootmcp.Error
type ToolHandler = rootmcp.ToolHandler

type Server struct {
	port        int
	config      *config.Config
	db          *sql.DB
	handlers    map[string]ToolHandler
	skillLoader *skills.SkillLoader
}

func (s *Server) Handlers() map[string]ToolHandler {
	return s.handlers
}

type Skill struct {
	ID           string    `json:"id"`
	Name         string    `json:"name"`
	Description  string    `json:"description"`
	Version      string    `json:"version,omitempty"`
	Trigger      string    `json:"trigger,omitempty"`
	AllowedTools []string  `json:"allowed_tools,omitempty"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

type Rule struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Category  string    `json:"category"`
	Content   string    `json:"content,omitempty"`
	Severity  string    `json:"severity"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
}

type Agent struct {
	ID           string   `json:"id"`
	Name         string   `json:"name"`
	Role         string   `json:"role,omitempty"`
	Skills       []string `json:"skills,omitempty"`
	Scope        string   `json:"scope,omitempty"`
	AllowedTools []string `json:"allowed_tools,omitempty"`
	IsActive     int      `json:"is_active"`
}

type Workflow struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
}

func NewServer(cfg *config.Config, db *sql.DB) *Server {
	skillsDir := filepath.Join(cfg.DataDir, "skills")
	s := &Server{
		port:        cfg.Port,
		config:      cfg,
		db:          db,
		handlers:    make(map[string]ToolHandler),
		skillLoader: skills.NewSkillLoader(skillsDir),
	}
	s.registerHandlers()
	return s
}

func (s *Server) registerHandlers() {
	s.handlers["rule_list"] = s.handleRuleList
	s.handlers["rule_check"] = s.handleRuleCheck
	s.handlers["rule_get"] = s.handleRuleGet

	s.handlers["skill_list"] = s.handleSkillList
	s.handlers["skill_load"] = s.handleSkillLoad
	s.handlers["skill_search"] = s.handleSkillSearch
	s.handlers["skill_version_check"] = s.handleSkillVersionCheck
	s.handlers["skill_verify"] = s.handleSkillVerify
	s.handlers["skill_read_file"] = s.handleSkillReadFile

	s.handlers["agent_list"] = s.handleAgentList
	s.handlers["agent_activate"] = s.handleAgentActivate
	s.handlers["agent_context"] = s.handleAgentContext
	s.handlers["agent_handoff"] = s.handleAgentHandoff

	s.handlers["agent_create"] = s.handleAgentCreate
	s.handlers["agent_get"] = s.handleAgentGet
	s.handlers["agent_specialized_list"] = s.handleSpecializedAgentList
	s.handlers["agent_assign_task"] = s.handleAgentAssignTask
	s.handlers["agent_complete_task"] = s.handleAgentCompleteTask
	s.handlers["agent_heartbeat"] = s.handleAgentHeartbeat
	s.handlers["agent_skills_get"] = s.handleAgentSkillsGet
	s.handlers["team_create"] = s.handleTeamCreate
	s.handlers["team_get"] = s.handleTeamGet

	s.handlers["workflow_start"] = s.handleWorkflowStart
	s.handlers["workflow_step"] = s.handleWorkflowStep
	s.handlers["workflow_status"] = s.handleWorkflowStatus
	s.handlers["workflow_complete"] = s.handleWorkflowComplete
	s.handlers["workflow_deprecate"] = s.handleWorkflowDeprecate

	s.handlers["task_execute"] = s.handleTaskExecute
	s.handlers["task_delegate"] = s.handleTaskDelegate
	s.handlers["task_status"] = s.handleTaskStatus
	s.handlers["task_heartbeat"] = s.handleTaskHeartbeat
	s.handlers["task_complete"] = s.handleTaskComplete
	s.handlers["task_cancel"] = s.handleTaskCancel

	s.handlers["skoll_status"] = s.handleSkollStatus
	s.handlers["skoll_validate"] = s.handleSkollValidate

	s.handlers["rule_pending"] = s.handleRulePending
	s.handlers["rule_promote"] = s.handleRulePromote

	s.handlers["team_status"] = s.handleTeamStatus
	s.handlers["team_register"] = s.handleTeamRegister

	s.handlers["dod_check"] = s.handleDodCheck

	s.handlers["skills_import"] = s.handleSkillsImport
	s.handlers["skills_update"] = s.handleSkillsUpdate
	s.handlers["api_docs_check"] = s.handleApiDocsCheck
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
	log.Printf("Skoll MCP server running on stdio")

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

	log.Printf("Skoll MCP server listening on %s (TCP)", addr)

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
		"server": "skoll",
		"port":   s.port,
	})
}

func (s *Server) handleRoot(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"name":    "Skoll",
		"version": "1.4.0",
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
	log.Printf("Skoll MCP server running on %s", addr)

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
