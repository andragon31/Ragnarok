package mcp

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"os"

	"github.com/ragnarok-ecosystem/fenrir/internal/config"
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
	s.handlers["mem_session_start"] = s.handleSessionStart
	s.handlers["mem_save"] = s.handleMemSave
	s.handlers["mem_find"] = s.handleMemFind
	s.handlers["mem_context"] = s.handleMemContext
	s.handlers["mem_timeline"] = s.handleMemTimeline
	s.handlers["mem_stats"] = s.handleStats
	s.handlers["mem_session_end"] = s.handleMemSessionEnd
	s.handlers["mem_save_prompt"] = s.handleMemSavePrompt
	s.handlers["mem_session_checkpoint"] = s.handleMemSessionCheckpoint
	s.handlers["mem_get_observation"] = s.handleMemGetObservation

	s.handlers["spec_save"] = s.handleSpecSave
	s.handlers["spec_list"] = s.handleSpecList
	s.handlers["spec_check"] = s.handleSpecCheck
	s.handlers["spec_delta"] = s.handleSpecDelta

	s.handlers["incident_log"] = s.handleIncidentLog
	s.handlers["incident_list"] = s.handleIncidentList
	s.handlers["incident_resolve"] = s.handleIncidentResolve

	s.handlers["conflict_list"] = s.handleConflictList
	s.handlers["conflict_resolve"] = s.handleConflictResolve

	s.handlers["intent_save"] = s.handleIntentSave
	s.handlers["intent_verify"] = s.handleIntentVerify
	s.handlers["intent_get"] = s.handleIntentGet

	s.handlers["bias_report"] = s.handleBiasReport

	s.handlers["project_scan"] = s.handleProjectScan
	s.handlers["project_bootstrap"] = s.handleProjectBootstrap
	s.handlers["skill_generate"] = s.handleSkillGenerate
	s.handlers["rules_generate"] = s.handleRulesGenerate
	s.handlers["standards_generate"] = s.handleStandardsGenerate
	s.handlers["prompt_analyze"] = s.handlePromptAnalyze
	s.handlers["agents_md_get"] = s.handleAgentsMdGet
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

func (s *Server) Run(ctx context.Context) error {
	addr := fmt.Sprintf(":%d", s.port)
	log.Printf("Fenrir MCP server running on %s", addr)

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
