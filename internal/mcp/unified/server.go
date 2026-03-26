package unified

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"

	fenrirconfig "github.com/andragon31/Ragnarok/internal/fenrir/config"
	fenrirdb "github.com/andragon31/Ragnarok/internal/fenrir/database"
	fenrirmcp "github.com/andragon31/Ragnarok/internal/fenrir/mcp"
	haticonfig "github.com/andragon31/Ragnarok/internal/hati/config"
	hatidb "github.com/andragon31/Ragnarok/internal/hati/database"
	hatimcp "github.com/andragon31/Ragnarok/internal/hati/mcp"
	"github.com/andragon31/Ragnarok/internal/mcp"
	skollconfig "github.com/andragon31/Ragnarok/internal/skoll/config"
	skolldb "github.com/andragon31/Ragnarok/internal/skoll/database"
	skollmcp "github.com/andragon31/Ragnarok/internal/skoll/mcp"
	tyrconfig "github.com/andragon31/Ragnarok/internal/tyr/config"
	tyrdb "github.com/andragon31/Ragnarok/internal/tyr/database"
	tyrmcp "github.com/andragon31/Ragnarok/internal/tyr/mcp"
)

type Server struct {
	handlers      map[string]mcp.ToolHandler
	tools         []Tool
	serverName    string
	serverVersion string
}

type Tool struct {
	Name        string          `json:"name"`
	Description string          `json:"description"`
	InputSchema json.RawMessage `json:"inputSchema"`
}

type InitializeResult struct {
	ProtocolVersion string                 `json:"protocolVersion"`
	Capabilities    map[string]interface{} `json:"capabilities"`
	ServerInfo      ServerInfo             `json:"serverInfo"`
}

type ServerInfo struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

func NewServer(dataDir string) (*Server, error) {
	if dataDir == "" {
		home, _ := os.UserHomeDir()
		dataDir = filepath.Join(home, ".ragnarok")
	}

	s := &Server{
		handlers:      make(map[string]mcp.ToolHandler),
		tools:         []Tool{},
		serverName:    "ragnarok",
		serverVersion: "1.2.0",
	}

	s.registerHandlers(dataDir)

	return s, nil
}

func (s *Server) registerHandlers(dataDir string) {
	fCfg := &fenrirconfig.Config{DataDir: filepath.Join(dataDir, ".fenrir")}
	fDB, err := fenrirdb.NewDB(filepath.Join(fCfg.DataDir, "fenrir.db"))
	if err == nil {
		fenrirdb.InitSchema(fDB)
		fSrv := fenrirmcp.NewServer(fCfg, fDB)
		for k, v := range fSrv.Handlers() {
			s.handlers[k] = v
			s.tools = append(s.tools, Tool{
				Name:        k,
				Description: getToolDescription(k),
				InputSchema: json.RawMessage(`{"type":"object","properties":{}}`),
			})
		}
	}

	hCfg, _ := haticonfig.LoadConfig(filepath.Join(dataDir, ".hati"))
	hDB, err := hatidb.NewDB(hCfg.DBPath())
	if err == nil {
		hatidb.InitSchema(hDB)
		hSrv := hatimcp.NewServer(hCfg, hDB)
		for k, v := range hSrv.Handlers() {
			s.handlers[k] = v
			s.tools = append(s.tools, Tool{
				Name:        k,
				Description: getToolDescription(k),
				InputSchema: json.RawMessage(`{"type":"object","properties":{}}`),
			})
		}
	}

	skCfg, _ := skollconfig.LoadConfig(filepath.Join(dataDir, ".skoll"))
	skDB, err := skolldb.NewDB(skCfg.DBPath())
	if err == nil {
		skolldb.InitSchema(skDB)
		skSrv := skollmcp.NewServer(skCfg, skDB)
		for k, v := range skSrv.Handlers() {
			s.handlers[k] = v
			s.tools = append(s.tools, Tool{
				Name:        k,
				Description: getToolDescription(k),
				InputSchema: json.RawMessage(`{"type":"object","properties":{}}`),
			})
		}
	}

	tCfg, _ := tyrconfig.LoadConfig(filepath.Join(dataDir, ".tyr"))
	tDB, err := tyrdb.NewDB(tCfg.DBPath())
	if err == nil {
		tyrdb.InitSchema(tDB)
		tSrv := tyrmcp.NewServer(tCfg, tDB)
		for k, v := range tSrv.Handlers() {
			s.handlers[k] = v
			s.tools = append(s.tools, Tool{
				Name:        k,
				Description: getToolDescription(k),
				InputSchema: json.RawMessage(`{"type":"object","properties":{}}`),
			})
		}
	}
}

func getToolDescription(name string) string {
	descriptions := map[string]string{
		"mem_save":           "Save an observation to memory",
		"mem_find":           "Search memories",
		"mem_context":        "Get context for a module",
		"mem_timeline":       "Get recent memories",
		"mem_stats":          "Get memory statistics",
		"spec_save":          "Save a specification",
		"spec_list":          "List specifications",
		"plan_create":        "Create a plan",
		"plan_list":          "List plans",
		"checkpoint_open":    "Open a checkpoint",
		"skill_list":         "List skills",
		"skill_load":         "Load a skill",
		"pkg_check":          "Check a package",
		"sast_run":           "Run SAST scan",
		"precommit_validate": "Validate pre-commit",
	}
	if desc, ok := descriptions[name]; ok {
		return desc
	}
	return fmt.Sprintf("Ragnarok tool: %s", name)
}

func (s *Server) Run(ctx context.Context) error {
	log.Printf("Ragnarok Unified MCP server running on stdio")

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
				return err
			}

			var baseReq struct {
				Method string      `json:"method"`
				ID     interface{} `json:"id"`
			}
			if err := json.Unmarshal(raw, &baseReq); err != nil {
				continue
			}

			var resp interface{}
			switch baseReq.Method {
			case "initialize":
				resp = s.handleInitialize(baseReq.ID)
			case "tools/list":
				resp = s.handleToolsList(baseReq.ID)
			case "tools/call":
				resp = s.handleToolsCall(ctx, raw, baseReq.ID)
			default:
				handler, ok := s.handlers[baseReq.Method]
				if !ok {
					continue
				}

				var req mcp.Request
				json.Unmarshal(raw, &req)
				result, err := handler(ctx, &req)
				if err != nil {
					continue
				}
				resp = map[string]interface{}{
					"jsonrpc": "2.0",
					"id":      baseReq.ID,
					"result":  result,
				}
			}

			if resp != nil {
				encoder.Encode(resp)
			}
		}
	}
}

func (s *Server) handleInitialize(id interface{}) map[string]interface{} {
	return map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      id,
		"result": InitializeResult{
			ProtocolVersion: "2024-11-05",
			Capabilities: map[string]interface{}{
				"tools": map[string]interface{}{},
			},
			ServerInfo: ServerInfo{
				Name:    s.serverName,
				Version: s.serverVersion,
			},
		},
	}
}

func (s *Server) handleToolsList(id interface{}) map[string]interface{} {
	return map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      id,
		"result": map[string]interface{}{
			"tools": s.tools,
		},
	}
}

func (s *Server) handleToolsCall(ctx context.Context, raw json.RawMessage, id interface{}) map[string]interface{} {
	var req struct {
		Params struct {
			Name      string          `json:"name"`
			Arguments json.RawMessage `json:"arguments"`
		} `json:"params"`
	}
	json.Unmarshal(raw, &req)

	handler, ok := s.handlers[req.Params.Name]
	if !ok {
		return map[string]interface{}{
			"jsonrpc": "2.0",
			"id":      id,
			"error": map[string]interface{}{
				"code":    -32601,
				"message": fmt.Sprintf("Tool not found: %s", req.Params.Name),
			},
		}
	}

	mcpReq := &mcp.Request{
		Method: req.Params.Name,
		Params: req.Params.Arguments,
	}

	result, err := handler(ctx, mcpReq)
	if err != nil {
		return map[string]interface{}{
			"jsonrpc": "2.0",
			"id":      id,
			"error": map[string]interface{}{
				"code":    -32603,
				"message": err.Error(),
			},
		}
	}

	return map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      id,
		"result": map[string]interface{}{
			"content": []map[string]interface{}{
				{"type": "text", "text": formatResult(result)},
			},
		},
	}
}

func formatResult(result interface{}) string {
	if result == nil {
		return "{}"
	}
	data, err := json.Marshal(result)
	if err != nil {
		return fmt.Sprintf("%v", result)
	}
	return string(data)
}
