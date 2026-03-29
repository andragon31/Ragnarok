package unified

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

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
	dbPaths       map[string]string
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

	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create data directory %s: %w", dataDir, err)
	}

	log.Printf("Ragnarok unified server initializing...")
	log.Printf("Data directory: %s", dataDir)

	s := &Server{
		handlers:      make(map[string]mcp.ToolHandler),
		tools:         []Tool{},
		serverName:    "ragnarok",
		serverVersion: "1.4.0",
		dbPaths:       make(map[string]string),
	}

	if err := s.registerHandlers(dataDir); err != nil {
		log.Printf("Warning: some plugins failed to initialize: %v", err)
	}

	log.Printf("Server initialized with %d tools", len(s.tools))

	return s, nil
}

func (s *Server) registerHandlers(dataDir string) error {
	var errs []error

	log.Printf("Initializing Fenrir...")
	fCfg := &fenrirconfig.Config{DataDir: filepath.Join(dataDir, ".fenrir")}
	fDB, err := fenrirdb.NewDB(filepath.Join(fCfg.DataDir, "fenrir.db"))
	if err != nil {
		errs = append(errs, fmt.Errorf("fenrir: failed to open database: %w", err))
		log.Printf("  Fenrir: ❌ failed to open database at %s", filepath.Join(fCfg.DataDir, "fenrir.db"))
	} else {
		if err := fenrirdb.InitSchema(fDB); err != nil {
			errs = append(errs, fmt.Errorf("fenrir: failed to init schema: %w", err))
			log.Printf("  Fenrir: ❌ failed to init schema: %v", err)
		} else {
			s.dbPaths["fenrir"] = filepath.Join(fCfg.DataDir, "fenrir.db")
			fSrv := fenrirmcp.NewServer(fCfg, fDB)
			for k, v := range fSrv.Handlers() {
				s.handlers[k] = v
				s.tools = append(s.tools, Tool{
					Name:        k,
					Description: getToolDescription(k),
					InputSchema: json.RawMessage(getToolInputSchema(k)),
				})
			}
			log.Printf("  Fenrir: ✅ initialized (%d handlers)", len(fSrv.Handlers()))
		}
	}

	log.Printf("Initializing Hati...")
	hCfg, err := haticonfig.LoadConfig(filepath.Join(dataDir, ".hati"))
	if err != nil {
		errs = append(errs, fmt.Errorf("hati: failed to load config: %w", err))
		log.Printf("  Hati: ❌ failed to load config: %v", err)
	} else {
		hDB, err := hatidb.NewDB(hCfg.DBPath())
		if err != nil {
			errs = append(errs, fmt.Errorf("hati: failed to open database: %w", err))
			log.Printf("  Hati: ❌ failed to open database at %s", hCfg.DBPath())
		} else {
			if err := hatidb.InitSchema(hDB); err != nil {
				errs = append(errs, fmt.Errorf("hati: failed to init schema: %w", err))
				log.Printf("  Hati: ❌ failed to init schema: %v", err)
			} else {
				s.dbPaths["hati"] = hCfg.DBPath()
				hSrv := hatimcp.NewServer(hCfg, hDB)
				for k, v := range hSrv.Handlers() {
					s.handlers[k] = v
					s.tools = append(s.tools, Tool{
						Name:        k,
						Description: getToolDescription(k),
						InputSchema: json.RawMessage(getToolInputSchema(k)),
					})
				}
				log.Printf("  Hati: ✅ initialized (%d handlers)", len(hSrv.Handlers()))
			}
		}
	}

	log.Printf("Initializing Skoll...")
	skCfg, err := skollconfig.LoadConfig(filepath.Join(dataDir, ".skoll"))
	if err != nil {
		errs = append(errs, fmt.Errorf("skoll: failed to load config: %w", err))
		log.Printf("  Skoll: ❌ failed to load config: %v", err)
	} else {
		skDB, err := skolldb.NewDB(skCfg.DBPath())
		if err != nil {
			errs = append(errs, fmt.Errorf("skoll: failed to open database: %w", err))
			log.Printf("  Skoll: ❌ failed to open database at %s", skCfg.DBPath())
		} else {
			if err := skolldb.InitSchema(skDB); err != nil {
				errs = append(errs, fmt.Errorf("skoll: failed to init schema: %w", err))
				log.Printf("  Skoll: ❌ failed to init schema: %v", err)
			} else {
				s.dbPaths["skoll"] = skCfg.DBPath()
				skSrv := skollmcp.NewServer(skCfg, skDB)
				for k, v := range skSrv.Handlers() {
					s.handlers[k] = v
					s.tools = append(s.tools, Tool{
						Name:        k,
						Description: getToolDescription(k),
						InputSchema: json.RawMessage(getToolInputSchema(k)),
					})
				}
				log.Printf("  Skoll: ✅ initialized (%d handlers)", len(skSrv.Handlers()))
			}
		}
	}

	log.Printf("Initializing Tyr...")
	tCfg, err := tyrconfig.LoadConfig(filepath.Join(dataDir, ".tyr"))
	if err != nil {
		errs = append(errs, fmt.Errorf("tyr: failed to load config: %w", err))
		log.Printf("  Tyr: ❌ failed to load config: %v", err)
	} else {
		tDB, err := tyrdb.NewDB(tCfg.DBPath())
		if err != nil {
			errs = append(errs, fmt.Errorf("tyr: failed to open database: %w", err))
			log.Printf("  Tyr: ❌ failed to open database at %s", tCfg.DBPath())
		} else {
			if err := tyrdb.InitSchema(tDB); err != nil {
				errs = append(errs, fmt.Errorf("tyr: failed to init schema: %w", err))
				log.Printf("  Tyr: ❌ failed to init schema: %v", err)
			} else {
				s.dbPaths["tyr"] = tCfg.DBPath()
				tSrv := tyrmcp.NewServer(tCfg, tDB)
				for k, v := range tSrv.Handlers() {
					s.handlers[k] = v
					s.tools = append(s.tools, Tool{
						Name:        k,
						Description: getToolDescription(k),
						InputSchema: json.RawMessage(getToolInputSchema(k)),
					})
				}
				log.Printf("  Tyr: ✅ initialized (%d handlers)", len(tSrv.Handlers()))
			}
		}
	}

	s.registerWorkflowHandlers()

	if len(errs) > 0 {
		return fmt.Errorf("plugin initialization errors: %v", errs)
	}
	return nil
}

func (s *Server) registerWorkflowHandlers() {
	workflows := map[string]struct {
		desc   string
		schema string
		fn     func(context.Context, *Request) (*Response, error)
	}{
		"workflow_project_bootstrap": {
			desc:   "Bootstrap complete project structure [DEPRECATED: Use workflow_stack_based_init]",
			schema: `{"type":"object","properties":{"project_path":{"type":"string"},"project_name":{"type":"string"},"prd_file":{"type":"string"}},"required":["project_path"]}`,
			fn:     s.handleWorkflowProjectBootstrap,
		},
		"workflow_prd_analyze": {
			desc:   "Analyze PRD and create full development plan with stack detection",
			schema: `{"type":"object","properties":{"prd_file":{"type":"string"},"project_path":{"type":"string"},"plan_title":{"type":"string"}},"required":["prd_file"]}`,
			fn:     s.handleWorkflowPRDAnalyze,
		},
		"workflow_agentic_init": {
			desc:   "Initialize agentic development structure [DEPRECATED: Use workflow_stack_based_init]",
			schema: `{"type":"object","properties":{"title":{"type":"string"},"description":{"type":"string"},"phases":{"type":"array","items":{"type":"string"}},"agent_name":{"type":"string"},"project_path":{"type":"string"}},"required":["title","phases"]}`,
			fn:     s.handleWorkflowAgenticInit,
		},
		"workflow_plan_develop": {
			desc:   "Execute development guided by tasks [DEPRECATED: Use workflow_plan_develop_v2]",
			schema: `{"type":"object","properties":{"plan_id":{"type":"string"},"agent_id":{"type":"string"},"auto_continue":{"type":"boolean"}},"required":["plan_id"]}`,
			fn:     s.handleWorkflowPlanDevelop,
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
		"workflow_checkpoint_create": {
			desc:   "Create quality checkpoint with human approval",
			schema: `{"type":"object","properties":{"plan_id":{"type":"string"},"phase_id":{"type":"string"},"description":{"type":"string"}},"required":["plan_id","description"]}`,
			fn:     s.handleWorkflowCheckpointCreate,
		},
		"ecosystem_diagnose": {
			desc:   "Run ecosystem health diagnostics",
			schema: `{"type":"object","properties":{"verbose":{"type":"boolean","description":"Show detailed diagnostics"}}}`,
			fn:     s.handleEcosystemDiagnose,
		},
		"workflow_project_lifecycle": {
			desc:   "Execute full project lifecycle: analyze, plan, assign agents, validate (Recommended for agents)",
			schema: `{"type":"object","properties":{"project_path":{"type":"string","description":"Project directory path"},"prd_file":{"type":"string","description":"PRD file path (optional)"},"title":{"type":"string","description":"Project title (optional)"},"requirements":{"type":"array","items":{"type":"string"},"description":"Requirements array (optional)"},"auto_start":{"type":"boolean","description":"Auto-start development after planning"}},"required":["project_path"]}`,
			fn:     s.handleWorkflowProjectLifecycle,
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

func (s *Server) ExecuteWorkflow(ctx context.Context, workflow string, params map[string]interface{}) (interface{}, error) {
	paramsJSON, err := json.Marshal(params)
	if err != nil {
		return nil, err
	}

	req := &Request{
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

func (s *Server) CallTool(ctx context.Context, tool string, params map[string]interface{}) (interface{}, error) {
	paramsJSON, err := json.Marshal(params)
	if err != nil {
		return nil, err
	}

	req := &Request{
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

	return result, nil
}

func getToolDescription(name string) string {
	descriptions := map[string]string{
		"mem_save":              "Save an observation to memory",
		"mem_find":              "Search memories",
		"mem_context":           "Get context for a module",
		"mem_timeline":          "Get recent memories",
		"mem_stats":             "Get memory statistics",
		"mem_session_start":     "Register session start",
		"mem_session_end":       "Mark session complete",
		"mem_get_observation":   "Get full content by ID",
		"mem_save_prompt":       "Save user prompt",
		"spec_save":             "Save a specification",
		"spec_list":             "List specifications",
		"spec_delta":            "Get spec changes",
		"spec_impact":           "Check spec impact",
		"spec_check":            "Verify spec compliance",
		"plan_create":           "Create a plan",
		"plan_list":             "List plans",
		"plan_get":              "Get plan details",
		"plan_complete":         "Mark plan complete",
		"plan_abandon":          "Abandon a plan",
		"plan_resume":           "Resume a plan",
		"plan_revise":           "Revise a plan",
		"plan_blockers":         "List plan blockers",
		"plan_dependencies":     "Get plan dependencies",
		"checkpoint_open":       "Open a checkpoint",
		"checkpoint_approve":    "Approve checkpoint",
		"skill_list":            "List skills",
		"skill_load":            "Load a skill",
		"skill_search":          "Search skills",
		"skill_verify":          "Verify a skill",
		"skill_generate":        "Generate a skill",
		"skill_version_check":   "Check skill version",
		"skill_read_file":       "Read skill file",
		"skills_import":         "Import skills",
		"skills_update":         "Update skills",
		"pkg_check":             "Check a package",
		"pkg_license":           "Check package license",
		"pkg_audit":             "Audit package",
		"pkg_audit_snapshot":    "Get audit snapshot",
		"pkg_audit_continuous":  "Continuous audit",
		"sast_run":              "Run SAST scan",
		"sast_findings":         "Get SAST findings",
		"sast_resolve":          "Resolve SAST finding",
		"standard_list":         "List standards",
		"standard_run":          "Run standard check",
		"standard_run_all":      "Run all standards",
		"precommit_validate":    "Validate pre-commit",
		"precommit_autofix":     "Auto-fix pre-commit",
		"rule_list":             "List rules",
		"rule_check":            "Check a rule",
		"rule_get":              "Get rule details",
		"prompt_analyze":        "Analyze prompt",
		"standards_generate":    "Generate standards",
		"rules_generate":        "Generate rules",
		"agents_md_get":         "Get AGENTS.md content",
		"module_hints":          "Get module hints",
		"project_scan":          "Scan project",
		"project_bootstrap":     "Bootstrap project",
		"hati_stats":            "Get Hati statistics",
		"hati_status":           "Get Hati status",
		"hati_commit_info":      "Get commit info",
		"hati_register_commit":  "Register commit",
		"skoll_status":          "Get Skoll status",
		"skoll_validate":        "Validate with Skoll",
		"tyr_stats":             "Get Tyr statistics",
		"quality_snapshot":      "Get quality snapshot",
		"notification_list":     "List notifications",
		"notification_send":     "Send notification",
		"agent_list":            "List agents",
		"agent_context":         "Get agent context",
		"agent_activate":        "Activate agent",
		"agent_handoff":         "Handoff to agent",
		"agent_register_work":   "Register work",
		"agent_unregister_work": "Unregister work",
		"agent_list_work":       "List agent work",
		"team_status":           "Get team status",
		"team_register":         "Register team",
		"api_docs_check":        "Check API docs",
		"dod_check":             "Check definition of done",
		"bootstrap_import":      "Import bootstrap data",
	}
	if desc, ok := descriptions[name]; ok {
		return desc
	}
	return fmt.Sprintf("Ragnarok tool: %s", name)
}

func getToolInputSchema(name string) string {
	schemas := map[string]string{
		"mem_save":                 `{"type":"object","properties":{"title":{"type":"string","description":"Brief title"},"type":{"type":"string","enum":["bugfix","decision","pattern","discovery","config","refactor"],"description":"Observation type"},"what":{"type":"string","description":"What was done"},"why":{"type":"string","description":"Why it was necessary"},"where":{"type":"string","description":"Files affected"},"learned":{"type":"string","description":"What to remember"}},"required":["title","type"]}`,
		"mem_find":                 `{"type":"object","properties":{"query":{"type":"string","description":"Search query"},"limit":{"type":"integer","description":"Max results"}},"required":["query"]}`,
		"mem_context":              `{"type":"object","properties":{"module":{"type":"string","description":"Module path"},"include_predictions":{"type":"boolean","description":"Include predictions"}}}`,
		"mem_timeline":             `{"type":"object","properties":{"limit":{"type":"integer","description":"Max results"}}}`,
		"mem_stats":                `{"type":"object","properties":{}}`,
		"mem_session_start":        `{"type":"object","properties":{"goal":{"type":"string","description":"Session goal"},"module":{"type":"string","description":"Module name"}},"required":["goal"]}`,
		"mem_session_end":          `{"type":"object","properties":{"summary":{"type":"string","description":"Session summary"}}}`,
		"mem_get_observation":      `{"type":"object","properties":{"id":{"type":"string","description":"Observation ID"}},"required":["id"]}`,
		"mem_save_prompt":          `{"type":"object","properties":{"prompt":{"type":"string","description":"Prompt text"},"response":{"type":"string","description":"Prompt response"}},"required":["prompt"]}`,
		"spec_save":                `{"type":"object","properties":{"name":{"type":"string","description":"Spec name"},"description":{"type":"string","description":"Spec description"},"content":{"type":"string","description":"Spec content"},"block":{"type":"boolean","description":"Block merge"}},"required":["name"]}`,
		"spec_list":                `{"type":"object","properties":{"module":{"type":"string","description":"Filter by module"}}}`,
		"spec_delta":               `{"type":"object","properties":{"base":{"type":"string","description":"Base commit"},"head":{"type":"string","description":"Head commit"}}}`,
		"spec_impact":              `{"type":"object","properties":{"module":{"type":"string","description":"Module path"},"specs":{"type":"array","items":{"type":"string"},"description":"Spec names"}}}`,
		"spec_check":               `{"type":"object","properties":{"module":{"type":"string","description":"Module path"}}}`,
		"plan_create":              `{"type":"object","properties":{"title":{"type":"string","description":"Plan title"},"description":{"type":"string","description":"Plan description"},"risk_level":{"type":"string","enum":["low","medium","high","critical"]},"session_id":{"type":"string","description":"Session ID"},"phases":{"type":"array","items":{"type":"string"},"description":"Phase names"}},"required":["title"]}`,
		"plan_list":                `{"type":"object","properties":{"status":{"type":"string","enum":["active","completed","abandoned","all"]}}}`,
		"plan_get":                 `{"type":"object","properties":{"id":{"type":"string","description":"Plan ID"}},"required":["id"]}`,
		"plan_complete":            `{"type":"object","properties":{"id":{"type":"string","description":"Plan ID"}},"required":["id"]}`,
		"plan_abandon":             `{"type":"object","properties":{"id":{"type":"string","description":"Plan ID"}},"required":["id"]}`,
		"plan_resume":              `{"type":"object","properties":{"id":{"type":"string","description":"Plan ID"}},"required":["id"]}`,
		"plan_revise":              `{"type":"object","properties":{"id":{"type":"string","description":"Plan ID"},"changes":{"type":"string","description":"Changes to make"}},"required":["id"]}`,
		"plan_blockers":            `{"type":"object","properties":{"id":{"type":"string","description":"Plan ID"}},"required":["id"]}`,
		"plan_dependencies":        `{"type":"object","properties":{"id":{"type":"string","description":"Plan ID"}},"required":["id"]}`,
		"checkpoint_open":          `{"type":"object","properties":{"plan_id":{"type":"string","description":"Plan ID"},"description":{"type":"string","description":"Checkpoint description"}},"required":["plan_id"]}`,
		"checkpoint_approve":       `{"type":"object","properties":{"checkpoint_id":{"type":"string","description":"Checkpoint ID"},"approver":{"type":"string","description":"Approver name"},"notes":{"type":"string","description":"Notes"}},"required":["checkpoint_id"]}`,
		"skill_list":               `{"type":"object","properties":{"category":{"type":"string","description":"Filter by category"}}}`,
		"skill_load":               `{"type":"object","properties":{"name":{"type":"string","description":"Skill name"},"version":{"type":"string","description":"Skill version"}},"required":["name"]}`,
		"skill_search":             `{"type":"object","properties":{"query":{"type":"string","description":"Search query"}},"required":["query"]}`,
		"skill_verify":             `{"type":"object","properties":{"name":{"type":"string","description":"Skill name"}},"required":["name"]}`,
		"skill_generate":           `{"type":"object","properties":{"name":{"type":"string","description":"Skill name"},"type":{"type":"string","description":"Skill type"},"description":{"type":"string","description":"Skill description"},"content":{"type":"string","description":"Skill content"}},"required":["name","type"]}`,
		"skill_version_check":      `{"type":"object","properties":{"name":{"type":"string","description":"Skill name"}},"required":["name"]}`,
		"skill_read_file":          `{"type":"object","properties":{"path":{"type":"string","description":"File path"}},"required":["path"]}`,
		"skills_import":            `{"type":"object","properties":{"path":{"type":"string","description":"Import path"}},"required":["path"]}`,
		"skills_update":            `{"type":"object","properties":{"name":{"type":"string","description":"Skill name"}},"required":["name"]}`,
		"pkg_check":                `{"type":"object","properties":{"name":{"type":"string","description":"Package name"},"ecosystem":{"type":"string","enum":["npm","pypi","go","cargo","nuget","maven","rubygems","packagist"],"description":"Package ecosystem"},"version":{"type":"string","description":"Package version"},"check_cves":{"type":"boolean","description":"Check for CVEs"},"check_typos":{"type":"boolean","description":"Check for typosquatting"},"no_cache":{"type":"boolean","description":"Bypass cache"}},"required":["name"]}`,
		"pkg_license":              `{"type":"object","properties":{"name":{"type":"string","description":"Package name"},"ecosystem":{"type":"string","description":"Package ecosystem"},"version":{"type":"string","description":"Package version"}},"required":["name"]}`,
		"pkg_audit":                `{"type":"object","properties":{"name":{"type":"string","description":"Package name"},"ecosystem":{"type":"string","description":"Package ecosystem"}},"required":["name"]}`,
		"pkg_audit_snapshot":       `{"type":"object","properties":{}}`,
		"pkg_audit_continuous":     `{"type":"object","properties":{"path":{"type":"string","description":"Project path"}}}`,
		"sast_run":                 `{"type":"object","properties":{"path":{"type":"string","description":"Path to scan"},"rules":{"type":"array","items":{"type":"string"},"description":"Rule IDs"}}}`,
		"sast_findings":            `{"type":"object","properties":{"severity":{"type":"string","enum":["critical","high","medium","low"]}}}`,
		"sast_resolve":             `{"type":"object","properties":{"id":{"type":"string","description":"Finding ID"},"resolution":{"type":"string","description":"Resolution notes"}},"required":["id"]}`,
		"standard_list":            `{"type":"object","properties":{"category":{"type":"string","description":"Filter by category"}}}`,
		"standard_run":             `{"type":"object","properties":{"name":{"type":"string","description":"Standard name"},"context":{"type":"object","description":"Check context"}},"required":["name"]}`,
		"standard_run_all":         `{"type":"object","properties":{"context":{"type":"object","description":"Check context"}}}`,
		"precommit_validate":       `{"type":"object","properties":{"path":{"type":"string","description":"Path to validate"}},"required":["path"]}`,
		"precommit_autofix":        `{"type":"object","properties":{"path":{"type":"string","description":"Path to fix"}},"required":["path"]}`,
		"rule_list":                `{"type":"object","properties":{"severity":{"type":"string","enum":["critical","high","medium","low"]}}}`,
		"rule_check":               `{"type":"object","properties":{"name":{"type":"string","description":"Rule name"}},"required":["name"]}`,
		"rule_get":                 `{"type":"object","properties":{"name":{"type":"string","description":"Rule name"}},"required":["name"]}`,
		"prompt_analyze":           `{"type":"object","properties":{"prompt":{"type":"string","description":"Prompt to analyze"}},"required":["prompt"]}`,
		"standards_generate":       `{"type":"object","properties":{"context":{"type":"string","description":"Context for generation"}}}`,
		"rules_generate":           `{"type":"object","properties":{"context":{"type":"string","description":"Context for generation"}}}`,
		"agents_md_get":            `{"type":"object","properties":{"path":{"type":"string","description":"AGENTS.md path"}}}`,
		"module_hints":             `{"type":"object","properties":{"module":{"type":"string","description":"Module path"}},"required":["module"]}`,
		"project_scan":             `{"type":"object","properties":{"path":{"type":"string","description":"Project path"}},"required":["path"]}`,
		"project_bootstrap":        `{"type":"object","properties":{"path":{"type":"string","description":"Project path"},"name":{"type":"string","description":"Project name"}},"required":["path"]}`,
		"hati_stats":               `{"type":"object","properties":{}}`,
		"hati_status":              `{"type":"object","properties":{}}`,
		"hati_commit_info":         `{"type":"object","properties":{"commit":{"type":"string","description":"Commit hash"}},"required":["commit"]}`,
		"hati_register_commit":     `{"type":"object","properties":{"commit":{"type":"string","description":"Commit hash"},"message":{"type":"string","description":"Commit message"},"author":{"type":"string","description":"Author"}},"required":["commit"]}`,
		"skoll_status":             `{"type":"object","properties":{}}`,
		"skoll_validate":           `{"type":"object","properties":{"type":{"type":"string","description":"Validation type"},"context":{"type":"object","description":"Validation context"}},"required":["type"]}`,
		"tyr_stats":                `{"type":"object","properties":{}}`,
		"quality_snapshot":         `{"type":"object","properties":{"module":{"type":"string","description":"Module path"}}}`,
		"notification_list":        `{"type":"object","properties":{"status":{"type":"string","enum":["unread","all"]}}}`,
		"notification_send":        `{"type":"object","properties":{"to":{"type":"string","description":"Recipient"},"message":{"type":"string","description":"Message"},"priority":{"type":"string","enum":["low","normal","high"]}},"required":["to","message"]}`,
		"agent_list":               `{"type":"object","properties":{}}`,
		"agent_context":            `{"type":"object","properties":{"agent_id":{"type":"string","description":"Agent ID"}}}`,
		"agent_activate":           `{"type":"object","properties":{"agent_id":{"type":"string","description":"Agent ID"}},"required":["agent_id"]}`,
		"agent_handoff":            `{"type":"object","properties":{"to":{"type":"string","description":"Target agent"},"context":{"type":"string","description":"Handoff context"}},"required":["to"]}`,
		"agent_register_work":      `{"type":"object","properties":{"agent_id":{"type":"string","description":"Agent ID"},"agent_name":{"type":"string","description":"Agent name"},"plan_id":{"type":"string","description":"Plan ID"},"phase_id":{"type":"string","description":"Phase ID"},"task":{"type":"string","description":"Task description"},"priority":{"type":"string","enum":["low","normal","high"]}},"required":["agent_id","plan_id"]}`,
		"agent_unregister_work":    `{"type":"object","properties":{"id":{"type":"string","description":"Work ID"}},"required":["id"]}`,
		"agent_list_work":          `{"type":"object","properties":{"status":{"type":"string","enum":["pending","active","completed"]}}}`,
		"team_status":              `{"type":"object","properties":{}}`,
		"team_register":            `{"type":"object","properties":{"name":{"type":"string","description":"Team name"},"members":{"type":"array","items":{"type":"string"},"description":"Member IDs"}},"required":["name"]}`,
		"api_docs_check":           `{"type":"object","properties":{"url":{"type":"string","description":"API docs URL"}},"required":["url"]}`,
		"dod_check":                `{"type":"object","properties":{"plan_id":{"type":"string","description":"Plan ID"}},"required":["plan_id"]}`,
		"bootstrap_import":         `{"type":"object","properties":{"project_path":{"type":"string","description":"Project path"},"plugins":{"type":"array","items":{"type":"string"},"description":"Plugins to import"}},"required":["project_path"]}`,
		"task_create":              `{"type":"object","properties":{"phase_id":{"type":"string","description":"Phase ID"},"prd_requirement_id":{"type":"string","description":"PRD Requirement ID"},"title":{"type":"string","description":"Task title"},"description":{"type":"string","description":"Task description"},"priority":{"type":"integer","description":"Priority"},"assigned_agent_id":{"type":"string","description":"Assigned Agent ID"},"assigned_agent_type":{"type":"string","description":"Agent type"},"estimated_hours":{"type":"number","description":"Estimated hours"},"milestone":{"type":"boolean","description":"Is milestone"}},"required":["phase_id","title"]}`,
		"task_get":                 `{"type":"object","properties":{"task_id":{"type":"string","description":"Task ID"}},"required":["task_id"]}`,
		"task_get_next":            `{"type":"object","properties":{"plan_id":{"type":"string","description":"Plan ID"},"agent_type":{"type":"string","description":"Agent type"},"agent_id":{"type":"string","description":"Agent ID"}},"required":["plan_id"]}`,
		"task_update":              `{"type":"object","properties":{"task_id":{"type":"string","description":"Task ID"},"status":{"type":"string","enum":["pending","in_progress","completed","blocked"]},"notes":{"type":"string","description":"Notes"},"actual_hours":{"type":"number","description":"Actual hours"},"assigned_agent_id":{"type":"string","description":"Assigned Agent ID"}},"required":["task_id"]}`,
		"task_list":                `{"type":"object","properties":{"phase_id":{"type":"string","description":"Phase ID"},"plan_id":{"type":"string","description":"Plan ID"},"status":{"type":"string","description":"Status"},"limit":{"type":"integer","description":"Limit"}}}`,
		"phase_create":             `{"type":"object","properties":{"plan_id":{"type":"string","description":"Plan ID"},"title":{"type":"string","description":"Phase title"},"description":{"type":"string","description":"Description"},"order_num":{"type":"integer","description":"Order"},"agent_hints":{"type":"string","description":"Agent hints"}},"required":["plan_id","title"]}`,
		"phase_update":             `{"type":"object","properties":{"phase_id":{"type":"string","description":"Phase ID"},"status":{"type":"string","enum":["pending","in_progress","completed","blocked"]}},"required":["phase_id"]}`,
		"plan_progress":            `{"type":"object","properties":{"plan_id":{"type":"string","description":"Plan ID"}},"required":["plan_id"]}`,
		"plan_create_from_prd":     `{"type":"object","properties":{"prd_id":{"type":"string","description":"PRD ID"},"title":{"type":"string","description":"Plan title"},"description":{"type":"string","description":"Description"},"phases":{"type":"array","items":{"type":"string"},"description":"Phase names"}},"required":["prd_id"]}`,
		"plan_activate":            `{"type":"object","properties":{"plan_id":{"type":"string","description":"Plan ID"}},"required":["plan_id"]}`,
		"prd_parse":                `{"type":"object","properties":{"file_path":{"type":"string","description":"PRD file path"},"content":{"type":"string","description":"PRD content"}},"required":["file_path"]}`,
		"prd_requirements_extract": `{"type":"object","properties":{"prd_id":{"type":"string","description":"PRD ID"}},"required":["prd_id"]}`,
		"human_review_create":      `{"type":"object","properties":{"review_type":{"type":"string","enum":["prd_approval","phase_approval","checkpoint_approval","deploy_approval","blocker_resolution","team_approval"]},"entity_type":{"type":"string","description":"Entity type"},"entity_id":{"type":"string","description":"Entity ID"},"question":{"type":"string","description":"Question"}},"required":["review_type","entity_id"]}`,
		"human_review_decide":      `{"type":"object","properties":{"review_id":{"type":"string","description":"Review ID"},"decision":{"type":"string","enum":["approved","rejected"]},"approver":{"type":"string","description":"Approver"},"notes":{"type":"string","description":"Notes"}},"required":["review_id","decision"]}`,
		"human_review_pending":     `{"type":"object","properties":{"review_type":{"type":"string","description":"Review type"},"entity_type":{"type":"string","description":"Entity type"}}}`,
		"task_execute":             `{"type":"object","properties":{"task_id":{"type":"string","description":"Task ID"},"hati_task_id":{"type":"string","description":"Hati Task ID"},"agent_id":{"type":"string","description":"Agent ID"},"phase_id":{"type":"string","description":"Phase ID"}},"required":["task_id","agent_id"]}`,
		"task_delegate":            `{"type":"object","properties":{"task_id":{"type":"string","description":"Task ID"},"hati_task_id":{"type":"string","description":"Hati Task ID"},"agent_ids":{"type":"array","items":{"type":"string"},"description":"Agent IDs"},"phase_id":{"type":"string","description":"Phase ID"}},"required":["task_id","agent_ids"]}`,
		"task_status":              `{"type":"object","properties":{"execution_id":{"type":"string","description":"Execution ID"},"task_id":{"type":"string","description":"Task ID"},"agent_id":{"type":"string","description":"Agent ID"}}}`,
		"task_heartbeat":           `{"type":"object","properties":{"execution_id":{"type":"string","description":"Execution ID"}},"required":["execution_id"]}`,
		"task_complete":            `{"type":"object","properties":{"execution_id":{"type":"string","description":"Execution ID"},"status":{"type":"string","enum":["completed","failed"]},"result":{"type":"string","description":"Result"},"error":{"type":"string","description":"Error"}},"required":["execution_id"]}`,
		"task_cancel":              `{"type":"object","properties":{"execution_id":{"type":"string","description":"Execution ID"},"reason":{"type":"string","description":"Reason for cancellation"}},"required":["execution_id"]}`,
	}
	if schema, ok := schemas[name]; ok {
		return schema
	}
	return `{"type":"object","properties":{}}`
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
				if errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF) {
					log.Printf("Client disconnected, shutting down gracefully")
					return nil
				}
				if strings.Contains(err.Error(), "closed") || strings.Contains(err.Error(), "EOF") {
					log.Printf("Connection closed by client")
					return nil
				}
				log.Printf("Decode error: %v", err)
				continue
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
					resp = map[string]interface{}{
						"jsonrpc": "2.0",
						"id":      baseReq.ID,
						"error":   map[string]string{"code": "-32601", "message": "Method not found: " + baseReq.Method},
					}
				} else {
					var req mcp.Request
					if err := json.Unmarshal(raw, &req); err != nil {
						resp = map[string]interface{}{
							"jsonrpc": "2.0",
							"id":      baseReq.ID,
							"error":   map[string]string{"code": "-32700", "message": "Parse error: " + err.Error()},
						}
					} else {
						handlerCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
						defer cancel()
						result, err := handler(handlerCtx, &req)
						if err != nil {
							log.Printf("Handler error for %s: %v", baseReq.Method, err)
							resp = map[string]interface{}{
								"jsonrpc": "2.0",
								"id":      baseReq.ID,
								"error":   map[string]string{"code": "-32603", "message": "Internal error: " + err.Error()},
							}
						} else {
							resp = map[string]interface{}{
								"jsonrpc": "2.0",
								"id":      baseReq.ID,
								"result":  result,
							}
						}
					}
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

func (s *Server) ListTools() []Tool {
	return s.tools
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
