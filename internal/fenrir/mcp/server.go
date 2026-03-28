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

	"github.com/andragon31/Ragnarok/internal/fenrir/config"
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

func (s *Server) RunStdio(ctx context.Context) error {
	log.Printf("Fenrir MCP server running on stdio")

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

	log.Printf("Fenrir MCP server listening on %s (TCP)", addr)

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
		"server": "fenrir",
		"port":   s.port,
	})
}

func (s *Server) handleRoot(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"name":    "Fenrir",
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
	log.Printf("Fenrir MCP server running on %s", addr)

	if os.Getenv("MCP_TRANSPORT") == "tcp" {
		return s.RunTCPServer(ctx)
	}

	return s.RunStdio(ctx)
}
