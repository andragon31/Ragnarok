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
	"github.com/andragon31/Ragnarok/internal/version"
)

type Server struct {
	handlers      map[string]mcp.ToolHandler
	tools         []Tool
	dbPaths       map[string]string
	serverVersion string
}

type Tool struct {
	Name        string          `json:"name"`
	Description string          `json:"description"`
	InputSchema json.RawMessage `json:"inputSchema"`
}

func NewServer(optDataDir string) (*Server, error) {
	dataDir := optDataDir
	if dataDir == "" {
		dataDir = os.Getenv("RAGNAROK_DATA_DIR")
	}
	if dataDir == "" {
		homeDir, _ := os.UserHomeDir()
		dataDir = filepath.Join(homeDir, ".ragnarok")
	}

	s := &Server{
		handlers:      make(map[string]mcp.ToolHandler),
		tools:         []Tool{},
		dbPaths:       make(map[string]string),
		serverVersion: version.Version,
	}

	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create data directory: %w", err)
	}

	if err := s.registerHandlers(dataDir); err != nil {
		return nil, err
	}

	return s, nil
}

func (s *Server) registerHandlers(dataDir string) error {
	var errs []error

	log.Printf("Initializing Ragnarok modules...")

	s.initFenrir(dataDir, &errs)
	s.initHati(dataDir, &errs)
	s.initSkoll(dataDir, &errs)
	s.initTyr(dataDir, &errs)

	s.registerWorkflowHandlers()
	s.registerHelpHandlers()

	if len(errs) > 0 {
		return fmt.Errorf("initialization errors: %v", errs)
	}
	return nil
}

func (s *Server) initFenrir(dataDir string, errs *[]error) {
	log.Printf("Initializing Fenrir...")
	fCfg := &fenrirconfig.Config{DataDir: filepath.Join(dataDir, ".fenrir")}
	fDB, err := fenrirdb.NewDB(filepath.Join(fCfg.DataDir, dbFenrir))
	if err != nil {
		*errs = append(*errs, fmt.Errorf("fenrir: failed to open database: %w", err))
	} else {
		if err := fenrirdb.InitSchema(fDB); err != nil {
			*errs = append(*errs, fmt.Errorf("fenrir: failed to init schema: %w", err))
		} else {
			s.dbPaths["fenrir"] = filepath.Join(fCfg.DataDir, dbFenrir)
			fServer := fenrirmcp.NewServer(fCfg, fDB)
			s.addHandlers(fServer.Handlers())
			log.Printf("  Fenrir: ✅ ready")
		}
	}
}

func (s *Server) initHati(dataDir string, errs *[]error) {
	log.Printf("Initializing Hati...")
	hCfg := &haticonfig.Config{DataDir: filepath.Join(dataDir, ".hati")}
	hDB, err := hatidb.NewDB(filepath.Join(hCfg.DataDir, dbHati))
	if err != nil {
		*errs = append(*errs, fmt.Errorf("hati: failed to open database: %w", err))
	} else {
		if err := hatidb.InitSchema(hDB); err != nil {
			*errs = append(*errs, fmt.Errorf("hati: failed to init schema: %w", err))
		} else {
			s.dbPaths["hati"] = filepath.Join(hCfg.DataDir, dbHati)
			hServer := hatimcp.NewServer(hCfg, hDB)
			s.addHandlers(hServer.Handlers())
			log.Printf("  Hati: ✅ ready")
		}
	}
}

func (s *Server) initSkoll(dataDir string, errs *[]error) {
	log.Printf("Initializing Skoll...")
	sCfg := &skollconfig.Config{DataDir: filepath.Join(dataDir, ".skoll")}
	sDB, err := skolldb.NewDB(filepath.Join(sCfg.DataDir, dbSkoll))
	if err != nil {
		*errs = append(*errs, fmt.Errorf("skoll: failed to open database: %w", err))
	} else {
		if err := skolldb.InitSchema(sDB); err != nil {
			*errs = append(*errs, fmt.Errorf("skoll: failed to init schema: %w", err))
		} else {
			s.dbPaths["skoll"] = filepath.Join(sCfg.DataDir, dbSkoll)
			sServer := skollmcp.NewServer(sCfg, sDB)
			s.addHandlers(sServer.Handlers())
			log.Printf("  Skoll: ✅ ready")
		}
	}
}

func (s *Server) initTyr(dataDir string, errs *[]error) {
	log.Printf("Initializing Tyr...")
	tCfg := &tyrconfig.Config{DataDir: filepath.Join(dataDir, ".tyr")}
	tDB, err := tyrdb.NewDB(filepath.Join(tCfg.DataDir, dbTyr))
	if err != nil {
		*errs = append(*errs, fmt.Errorf("tyr: failed to open database: %w", err))
	} else {
		if err := tyrdb.InitSchema(tDB); err != nil {
			*errs = append(*errs, fmt.Errorf("tyr: failed to init schema: %w", err))
		} else {
			s.dbPaths["tyr"] = filepath.Join(tCfg.DataDir, dbTyr)
			tServer := tyrmcp.NewServer(tCfg, tDB)
			s.addHandlers(tServer.Handlers())
			log.Printf("  Tyr: ✅ ready")
		}
	}
}

func (s *Server) addHandlers(handlers map[string]mcp.ToolHandler) {
	for name, handler := range handlers {
		s.handlers[name] = handler
		s.tools = append(s.tools, Tool{
			Name:        name,
			Description: getToolDescription(name),
			InputSchema: json.RawMessage(getToolInputSchema(name)),
		})
	}
}

func (s *Server) registerWorkflowHandlers() {
	workflows := map[string]struct {
		desc   string
		schema string
		fn     func(context.Context, *mcp.Request) (*mcp.Response, error)
	}{
		"workflow_prd_analyze": {
			desc:   "Analyze PRD and create full development plan with stack detection",
			schema: `{"type":"object","properties":{"prd_file":{"type":"string"},"project_path":{"type":"string"},"plan_title":{"type":"string"}},"required":["prd_file"]}`,
			fn:     s.handleWorkflowPRDAnalyze,
		},
		"workflow_plan_develop_v2": {
			desc:   "Execute development with multi-agent task delegation",
			schema: `{"type":"object","properties":{"plan_id":{"type":"string"},"agent_id":{"type":"string"},"auto_continue":{"type":"boolean"}},"required":["plan_id"]}`,
			fn:     s.handleWorkflowPlanDevelopV2,
		},
		"workflow_stack_based_init": {
			desc:   "Initialize project with stack-based phases and tasks (Recommended)",
			schema: `{"type":"object","properties":{"project_path":{"type":"string"},"title":{"type":"string"},"phases":{"type":"array","items":{"type":"string"}},"agent_ids":{"type":"array","items":{"type":"string"}}},"required":["project_path"]}`,
			fn:     s.handleWorkflowStackBasedInit,
		},
		"workflow_session_start": {
			desc:   "Start a work session with full context",
			schema: `{"type":"object","properties":{"goal":{"type":"string"},"module":{"type":"string"},"project_path":{"type":"string"}},"required":["goal"]}`,
			fn:     s.handleWorkflowSessionStart,
		},
		"workflow_project_lifecycle": {
			desc:   "Initialize full project lifecycle: scan + PRD + design + plan + agents + quality (Recommended)",
			schema: `{"type":"object","properties":{"project_path":{"type":"string"},"prd_file":{"type":"string"},"title":{"type":"string"},"auto_start":{"type":"boolean"}},"required":["project_path"]}`,
			fn:     s.handleWorkflowProjectLifecycle,
		},
		"workflow_checkpoint_create": {
			desc:   "Create quality checkpoint with human approval",
			schema: `{"type":"object","properties":{"plan_id":{"type":"string"},"phase_id":{"type":"string"},"description":{"type":"string"}},"required":["plan_id","description"]}`,
			fn:     s.handleWorkflowCheckpointCreate,
		},
		"workflow_team_setup_from_prd": {
			desc:   "Create full agent structure and project team from PRD analysis (Skoll)",
			schema: `{"type":"object","properties":{"prd_file":{"type":"string"},"project_path":{"type":"string"},"team_name":{"type":"string"}},"required":["prd_file"]}`,
			fn:     s.handleWorkflowTeamSetupFromPRD,
		},
		"workflow_onboard_existing": {
			desc:   "Onboard an existing project: scan + summarize + spec + skills + rules + quality init (Recommended for existing codebases)",
			schema: `{"type":"object","properties":{"project_path":{"type":"string"},"goal":{"type":"string"}},"required":["project_path"]}`,
			fn:     s.handleWorkflowOnboardExisting,
		},
		"ecosystem_diagnose": {
			desc:   "Run ecosystem health diagnostics",
			schema: `{"type":"object","properties":{"verbose":{"type":"boolean","description":"Show detailed diagnostics"}}}`,
			fn:     s.handleEcosystemDiagnose,
		},
		"analyze_stack_with_llm": {
			desc:   "Use LLM to analyze project stack and requirements, recommending specialized agents. First call returns a prompt to execute with your LLM. Pass the LLM response back to get agent recommendations.",
			schema: `{"type":"object","properties":{"project_path":{"type":"string","description":"Project root path"},"prd_file":{"type":"string","description":"Optional PRD file path"},"requirements":{"type":"array","items":{"type":"object"},"description":"Array of requirement objects with title and type"},"llm_response":{"type":"string","description":"The JSON response from your LLM after executing the prompt"}},"required":["project_path"]}`,
			fn:     s.handleAnalyzeStackWithLLM,
		},
	}

	for name, w := range workflows {
		s.handlers[name] = w.fn
		s.tools = append(s.tools, Tool{
			Name:        name,
			Description: w.desc,
			InputSchema: json.RawMessage(w.schema),
		})
	}
}

func (s *Server) CallTool(ctx context.Context, tool string, params map[string]interface{}) (interface{}, error) {
	paramsJSON, err := json.Marshal(params)
	if err != nil {
		return nil, err
	}

	req := &mcp.Request{
		Method: tool,
		Params: paramsJSON,
	}

	handler, ok := s.handlers[tool]
	if !ok {
		return nil, fmt.Errorf("tool not found: %s", tool)
	}

	result, err := handler(ctx, req)
	if err != nil {
		return nil, err
	}

	if result == nil {
		return nil, nil
	}

	return result.Result, nil
}

func (s *Server) Run(ctx context.Context) error {
	log.Printf("Ragnarok MCP Unified Server v%s starting...", s.serverVersion)
	log.Printf("Registered tools: %d", len(s.tools))

	decoder := json.NewDecoder(os.Stdin)
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			var raw json.RawMessage
			if err := decoder.Decode(&raw); err != nil {
				return err
			}
			go s.handleRequest(ctx, raw)
		}
	}
}

func (s *Server) handleRequest(ctx context.Context, raw json.RawMessage) {
	var base struct {
		Method string      `json:"method"`
		ID     interface{} `json:"id"`
	}
	if err := json.Unmarshal(raw, &base); err != nil {
		return
	}

	var resp interface{}
	switch base.Method {
	case "initialize":
		resp = map[string]interface{}{
			"jsonrpc": "2.0",
			"id":      base.ID,
			"result": map[string]interface{}{
				"protocolVersion": "2024-11-05",
				"capabilities":    map[string]interface{}{"tools": map[string]interface{}{}},
				"serverInfo":      map[string]interface{}{"name": "ragnarok-unified", "version": s.serverVersion},
			},
		}
	case "tools/list":
		resp = s.handleToolsList(base.ID)
	case "tools/call":
		resp = s.handleToolsCall(ctx, raw, base.ID)
	default:
		resp = map[string]interface{}{
			"jsonrpc": "2.0",
			"id":      base.ID,
			"error":   map[string]interface{}{"code": -32601, "message": "Method not found"},
		}
	}

	respJSON, _ := json.Marshal(resp)
	fmt.Println(string(respJSON))
}

func (s *Server) ListTools() []Tool {
	return s.tools
}

func (s *Server) ExecuteWorkflow(ctx context.Context, workflow string, params map[string]interface{}) (interface{}, error) {
	paramsJSON, err := json.Marshal(params)
	if err != nil {
		return nil, err
	}

	req := &mcp.Request{
		Method: workflow,
		Params: paramsJSON,
	}

	handler, ok := s.handlers[workflow]
	if !ok {
		return nil, fmt.Errorf("workflow not found: %s", workflow)
	}

	result, err := handler(ctx, req)
	if err != nil {
		return nil, err
	}

	if result == nil {
		return nil, nil
	}

	return result.Result, nil
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

	// Post-execution hook: Cross-module integration
	s.interceptTyrFindings(ctx, req.Params.Name, result)
	s.interceptTaskExecution(ctx, req.Params.Name, result)

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

func (s *Server) interceptTyrFindings(ctx context.Context, tool string, result interface{}) {
	if tool != "sast_run" && tool != "pkg_check" {
		return
	}

	resMap, ok := result.(*mcp.Response)
	if !ok || resMap == nil {
		return
	}

	dataMap, ok := resMap.Result.(map[string]interface{})
	if !ok {
		return
	}

	critical := false
	what := ""

	switch tool {
	case "sast_run":
		count := 0
		if c, ok := dataMap["total"].(int); ok {
			count = c
		} else if c, ok := dataMap["total"].(float64); ok {
			count = int(c)
		}
		if count > 0 {
			critical = true
			what = fmt.Sprintf("SAST scan found %d findings in %v", count, dataMap["target"])
		}
	case "pkg_check":
		if risky, ok := dataMap["typosquatting_risk"].(bool); ok && risky {
			critical = true
			what = fmt.Sprintf("Package %v has typosquatting risk!", dataMap["name"])
		}
		if cves, ok := dataMap["cve_count"].(int); ok && cves > 0 {
			critical = true
			what = fmt.Sprintf("Package %v has %d known CVEs!", dataMap["name"], cves)
		} else if cves, ok := dataMap["cve_count"].(float64); ok && cves > 0 {
			critical = true
			what = fmt.Sprintf("Package %v has %.0f known CVEs!", dataMap["name"], cves)
		}
	}

	if critical {
		log.Printf("  [Unified] Critical Tyr finding intercepted. Saving to Fenrir...")
		s.CallTool(ctx, "mem_save", map[string]interface{}{
			"title":        "Security Finding: " + tool,
			"type":         "bugfix",
			"what":         what,
			"why":          "Automatically captured by Tyr quality module.",
			"tyr_snapshot": "true",
			"learned":      "Dependencies and code must be audited before final deployment.",
		})
	}
}

func (s *Server) interceptTaskExecution(ctx context.Context, tool string, result interface{}) {
	if tool != "task_execute" && tool != "task_complete" && tool != "task_delegate" {
		return
	}

	resMap, ok := result.(*mcp.Response)
	if !ok || resMap == nil {
		return
	}

	dataMap, ok := resMap.Result.(map[string]interface{})
	if !ok {
		return
	}

	title := ""
	what := ""
	switch tool {
	case "task_execute":
		title = "Task Execution Started"
		what = fmt.Sprintf("Agent %v started task %v (Exec: %v)", dataMap["agent_id"], dataMap["task_id"], dataMap["execution_id"])
	case "task_complete":
		title = "Task Execution Completed"
		what = fmt.Sprintf("Task %v (Exec: %v) finished with status: %v", dataMap["task_id"], dataMap["execution_id"], dataMap["status"])
	case "task_delegate":
		title = "Task Delegation"
		what = fmt.Sprintf("Task %v delegated to %v agents", dataMap["task_id"], dataMap["total_agents"])
	}

	if title != "" {
		log.Printf("  [Unified] Task activity intercepted. Syncing with Fenrir...")
		s.CallTool(ctx, "mem_save", map[string]interface{}{
			"title":      title,
			"type":       "observation",
			"what":       what,
			"why":        "Automatically captured by Skoll orchestration module.",
			"skoll_sync": "true",
		})
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
