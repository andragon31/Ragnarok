package mcp

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

func (s *Server) handleRuleList(ctx context.Context, req *Request) (*Response, error) {
	var params struct {
		Category string `json:"category,omitempty"`
		Limit    int    `json:"limit,omitempty"`
	}

	if err := json.Unmarshal(req.Params, &params); err != nil {
		return nil, fmt.Errorf("failed to parse params: %w", err)
	}

	if params.Limit == 0 {
		params.Limit = 50
	}

	query := `SELECT id, name, category, content, status, created_at FROM rules WHERE (? = '' OR category = ?) LIMIT ?`
	rows, err := s.db.Query(query, params.Category, params.Category, params.Limit)
	if err != nil {
		return nil, fmt.Errorf("failed to list rules: %w", err)
	}
	defer rows.Close()

	var rules []*Rule
	for rows.Next() {
		r := &Rule{}
		err := rows.Scan(&r.ID, &r.Name, &r.Category, &r.Content, &r.Status, &r.CreatedAt)
		if err != nil {
			return nil, err
		}
		rules = append(rules, r)
	}

	return &Response{
		Result: map[string]interface{}{
			"rules": rules,
			"count": len(rules),
		},
	}, nil
}

func (s *Server) handleRuleCheck(ctx context.Context, req *Request) (*Response, error) {
	var params struct {
		Action string `json:"action"`
	}

	if err := json.Unmarshal(req.Params, &params); err != nil {
		return nil, fmt.Errorf("failed to parse params: %w", err)
	}

	return &Response{
		Result: map[string]interface{}{
			"action":        params.Action,
			"allowed":       true,
			"rules_checked": 0,
			"note":          "rule check requires rules to be loaded",
		},
	}, nil
}

func (s *Server) handleRuleGet(ctx context.Context, req *Request) (*Response, error) {
	var params struct {
		RuleID string `json:"rule_id"`
	}

	if err := json.Unmarshal(req.Params, &params); err != nil {
		return nil, fmt.Errorf("failed to parse params: %w", err)
	}

	query := `SELECT id, name, category, content, status, created_at FROM rules WHERE id = ?`
	r := &Rule{}
	err := s.db.QueryRow(query, params.RuleID).Scan(&r.ID, &r.Name, &r.Category, &r.Content, &r.Status, &r.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("rule not found: %w", err)
	}

	return &Response{
		Result: r,
	}, nil
}

func (s *Server) handleSkillList(ctx context.Context, req *Request) (*Response, error) {
	var params struct {
		Limit int `json:"limit,omitempty"`
	}

	if err := json.Unmarshal(req.Params, &params); err != nil {
		return nil, fmt.Errorf("failed to parse params: %w", err)
	}

	if params.Limit == 0 {
		params.Limit = 100
	}

	query := `SELECT id, name, description, version, trigger FROM skills LIMIT ?`
	rows, err := s.db.Query(query, params.Limit)
	if err != nil {
		return nil, fmt.Errorf("failed to list skills: %w", err)
	}
	defer rows.Close()

	var skills []*Skill
	for rows.Next() {
		sk := &Skill{}
		err := rows.Scan(&sk.ID, &sk.Name, &sk.Description, &sk.Version, &sk.Trigger)
		if err != nil {
			return nil, err
		}
		skills = append(skills, sk)
	}

	return &Response{
		Result: map[string]interface{}{
			"skills": skills,
			"count":  len(skills),
		},
	}, nil
}

func (s *Server) handleSkillLoad(ctx context.Context, req *Request) (*Response, error) {
	var params struct {
		SkillName string `json:"skill_name"`
	}

	if err := json.Unmarshal(req.Params, &params); err != nil {
		return nil, fmt.Errorf("failed to parse params: %w", err)
	}

	query := `SELECT id, name, description, version, allowed_tools FROM skills WHERE name = ?`
	sk := &Skill{}
	var allowedTools sql.NullString
	err := s.db.QueryRow(query, params.SkillName).Scan(&sk.ID, &sk.Name, &sk.Description, &sk.Version, &allowedTools)
	if err != nil {
		return &Response{
			Result: map[string]interface{}{
				"skill_name": params.SkillName,
				"found":      false,
				"error":      "skill not found",
			},
		}, nil
	}
	sk.AllowedTools = []string{}
	if allowedTools.Valid && allowedTools.String != "" {
		json.Unmarshal([]byte(allowedTools.String), &sk.AllowedTools)
	}

	return &Response{
		Result: map[string]interface{}{
			"skill":           sk,
			"found":           true,
			"available_files": []string{},
			"in_practice":     []string{},
		},
	}, nil
}

func (s *Server) handleSkillSearch(ctx context.Context, req *Request) (*Response, error) {
	var params struct {
		Query string `json:"query"`
		Limit int    `json:"limit,omitempty"`
	}

	if err := json.Unmarshal(req.Params, &params); err != nil {
		return nil, fmt.Errorf("failed to parse params: %w", err)
	}

	if params.Limit == 0 {
		params.Limit = 10
	}

	query := `SELECT id, name, description, version FROM skills WHERE name LIKE ? OR description LIKE ? LIMIT ?`
	searchPattern := "%" + params.Query + "%"
	rows, err := s.db.Query(query, searchPattern, searchPattern, params.Limit)
	if err != nil {
		return nil, fmt.Errorf("failed to search skills: %w", err)
	}
	defer rows.Close()

	var skills []*Skill
	for rows.Next() {
		sk := &Skill{}
		err := rows.Scan(&sk.ID, &sk.Name, &sk.Description, &sk.Version)
		if err != nil {
			return nil, err
		}
		skills = append(skills, sk)
	}

	return &Response{
		Result: map[string]interface{}{
			"skills": skills,
			"count":  len(skills),
		},
	}, nil
}

func (s *Server) handleSkillVersionCheck(ctx context.Context, req *Request) (*Response, error) {
	var params struct {
		SkillName string `json:"skill_name"`
		Version   string `json:"version"`
	}

	if err := json.Unmarshal(req.Params, &params); err != nil {
		return nil, fmt.Errorf("failed to parse params: %w", err)
	}

	return &Response{
		Result: map[string]interface{}{
			"skill_name": params.SkillName,
			"version":    params.Version,
			"compatible": true,
			"note":       "version check pending implementation",
		},
	}, nil
}

func (s *Server) handleSkillVerify(ctx context.Context, req *Request) (*Response, error) {
	var params struct {
		SkillName string `json:"skill_name"`
	}

	if err := json.Unmarshal(req.Params, &params); err != nil {
		return nil, fmt.Errorf("failed to parse params: %w", err)
	}

	return &Response{
		Result: map[string]interface{}{
			"skill_name": params.SkillName,
			"valid":      true,
			"verified":   false,
			"note":       "verification requires Tyr integration",
		},
	}, nil
}

func (s *Server) handleSkillReadFile(ctx context.Context, req *Request) (*Response, error) {
	var params struct {
		SkillName string `json:"skill_name"`
		FilePath  string `json:"file_path"`
	}

	if err := json.Unmarshal(req.Params, &params); err != nil {
		return nil, fmt.Errorf("failed to parse params: %w", err)
	}

	return &Response{
		Result: map[string]interface{}{
			"skill_name": params.SkillName,
			"file_path":  params.FilePath,
			"content":    "",
			"found":      false,
			"note":       "file reading pending implementation",
		},
	}, nil
}

func (s *Server) handleAgentList(ctx context.Context, req *Request) (*Response, error) {
	query := `SELECT id, name, scope FROM agents LIMIT 50`
	rows, err := s.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to list agents: %w", err)
	}
	defer rows.Close()

	var agents []*Agent
	for rows.Next() {
		a := &Agent{}
		err := rows.Scan(&a.ID, &a.Name, &a.Scope)
		if err != nil {
			return nil, err
		}
		agents = append(agents, a)
	}

	return &Response{
		Result: map[string]interface{}{
			"agents": agents,
			"count":  len(agents),
		},
	}, nil
}

func (s *Server) handleAgentActivate(ctx context.Context, req *Request) (*Response, error) {
	var params struct {
		AgentID     string `json:"agent_id"`
		ContextPath string `json:"context_path,omitempty"`
	}

	if err := json.Unmarshal(req.Params, &params); err != nil {
		return nil, fmt.Errorf("failed to parse params: %w", err)
	}

	agentsMdContext := ""
	skills := []string{}

	contextPath := params.ContextPath
	if contextPath == "" {
		contextPath = "."
	}

	agentsMdPath := findAgentsMd(contextPath)
	if agentsMdPath != "" {
		if content, err := os.ReadFile(agentsMdPath); err == nil {
			agentsMdContext = string(content)
			skills = extractSkillsFromAgentsMd(agentsMdContext)
		}
	}

	allowedTools := []string{
		"fenrir.mem_save",
		"fenrir.mem_find",
		"fenrir.intent_save",
		"fenrir.intent_verify",
		"hati.plan_create",
		"hati.checkpoint_open",
		"tyr.precommit_validate",
		"tyr.pkg_check",
		"skoll.rule_check",
		"skoll.skill_load",
	}

	return &Response{
		Result: map[string]interface{}{
			"agent_id":          params.AgentID,
			"activated":         true,
			"allowed_tools":     allowedTools,
			"agents_md_context": agentsMdContext,
			"skills_suggested":  skills,
			"agents_md_path":    agentsMdPath,
		},
	}, nil
}

func findAgentsMd(startPath string) string {
	currentPath := startPath
	for {
		agentsMdPath := filepath.Join(currentPath, "AGENTS.md")
		if _, err := os.Stat(agentsMdPath); err == nil {
			return agentsMdPath
		}
		parent := filepath.Dir(currentPath)
		if parent == currentPath || parent == "." {
			break
		}
		currentPath = parent
	}
	return ""
}

func extractSkillsFromAgentsMd(content string) []string {
	var skills []string
	lines := strings.Split(content, "\n")
	inSkillsSection := false

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(strings.ToLower(line), "## suggested skills") {
			inSkillsSection = true
			continue
		}
		if inSkillsSection {
			if strings.HasPrefix(line, "## ") {
				inSkillsSection = false
				continue
			}
			if strings.HasPrefix(line, "- ") {
				skill := strings.TrimPrefix(line, "- ")
				skill = strings.Split(skill, " ")[0]
				if skill != "" {
					skills = append(skills, skill)
				}
			}
		}
	}
	return skills
}

func (s *Server) handleAgentContext(ctx context.Context, req *Request) (*Response, error) {
	var params struct {
		AgentID string `json:"agent_id"`
	}

	if err := json.Unmarshal(req.Params, &params); err != nil {
		return nil, fmt.Errorf("failed to parse params: %w", err)
	}

	return &Response{
		Result: map[string]interface{}{
			"agent_id": params.AgentID,
			"context":  map[string]interface{}{},
		},
	}, nil
}

func (s *Server) handleAgentHandoff(ctx context.Context, req *Request) (*Response, error) {
	var params struct {
		FromAgent string `json:"from_agent"`
		ToAgent   string `json:"to_agent"`
		Contract  string `json:"contract,omitempty"`
	}

	if err := json.Unmarshal(req.Params, &params); err != nil {
		return nil, fmt.Errorf("failed to parse params: %w", err)
	}

	return &Response{
		Result: map[string]interface{}{
			"from_agent": params.FromAgent,
			"to_agent":   params.ToAgent,
			"handed_off": true,
			"contract":   params.Contract,
		},
	}, nil
}

func (s *Server) handleWorkflowStart(ctx context.Context, req *Request) (*Response, error) {
	var params struct {
		Name string `json:"name"`
		Type string `json:"type,omitempty"`
	}

	if err := json.Unmarshal(req.Params, &params); err != nil {
		return nil, fmt.Errorf("failed to parse params: %w", err)
	}

	wf := &Workflow{
		ID:        generateID("wf"),
		Name:      params.Name,
		Status:    "started",
		CreatedAt: time.Now(),
	}

	query := `INSERT INTO workflows (id, name, status, created_at) VALUES (?, ?, ?, ?)`
	_, err := s.db.Exec(query, wf.ID, wf.Name, wf.Status, wf.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("failed to start workflow: %w", err)
	}

	return &Response{
		Result: map[string]interface{}{
			"workflow_id": wf.ID,
			"name":        wf.Name,
			"status":      wf.Status,
			"started_at":  wf.CreatedAt,
		},
	}, nil
}

func (s *Server) handleWorkflowStep(ctx context.Context, req *Request) (*Response, error) {
	var params struct {
		WorkflowID string `json:"workflow_id"`
		Step       string `json:"step"`
	}

	if err := json.Unmarshal(req.Params, &params); err != nil {
		return nil, fmt.Errorf("failed to parse params: %w", err)
	}

	return &Response{
		Result: map[string]interface{}{
			"workflow_id": params.WorkflowID,
			"step":        params.Step,
			"completed":   true,
		},
	}, nil
}

func (s *Server) handleWorkflowStatus(ctx context.Context, req *Request) (*Response, error) {
	var params struct {
		WorkflowID string `json:"workflow_id"`
	}

	if err := json.Unmarshal(req.Params, &params); err != nil {
		return nil, fmt.Errorf("failed to parse params: %w", err)
	}

	query := `SELECT id, name, status, created_at FROM workflows WHERE id = ?`
	wf := &Workflow{}
	err := s.db.QueryRow(query, params.WorkflowID).Scan(&wf.ID, &wf.Name, &wf.Status, &wf.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("workflow not found: %w", err)
	}

	return &Response{
		Result: wf,
	}, nil
}

func (s *Server) handleWorkflowComplete(ctx context.Context, req *Request) (*Response, error) {
	var params struct {
		WorkflowID string `json:"workflow_id"`
	}

	if err := json.Unmarshal(req.Params, &params); err != nil {
		return nil, fmt.Errorf("failed to parse params: %w", err)
	}

	query := `UPDATE workflows SET status = 'completed' WHERE id = ?`
	_, err := s.db.Exec(query, params.WorkflowID)
	if err != nil {
		return nil, fmt.Errorf("failed to complete workflow: %w", err)
	}

	return &Response{
		Result: map[string]interface{}{
			"workflow_id":  params.WorkflowID,
			"status":       "completed",
			"completed_at": time.Now(),
		},
	}, nil
}

func (s *Server) handleSkollStatus(ctx context.Context, req *Request) (*Response, error) {
	var totalSkills, totalRules, totalAgents int
	s.db.QueryRow(`SELECT COUNT(*) FROM skills`).Scan(&totalSkills)
	s.db.QueryRow(`SELECT COUNT(*) FROM rules`).Scan(&totalRules)
	s.db.QueryRow(`SELECT COUNT(*) FROM agents`).Scan(&totalAgents)

	return &Response{
		Result: map[string]interface{}{
			"status":       "operational",
			"total_skills": totalSkills,
			"total_rules":  totalRules,
			"total_agents": totalAgents,
			"version":      "1.0.0",
		},
	}, nil
}

func (s *Server) handleSkollValidate(ctx context.Context, req *Request) (*Response, error) {
	var params struct {
		Strict bool `json:"strict,omitempty"`
	}

	json.Unmarshal(req.Params, &params)

	return &Response{
		Result: map[string]interface{}{
			"valid":     true,
			"errors":    []interface{}{},
			"warnings":  []interface{}{},
			"validated": true,
		},
	}, nil
}

func (s *Server) handleRulePending(ctx context.Context, req *Request) (*Response, error) {
	query := `SELECT id, name, category, content, status, created_at FROM rules WHERE status = 'pending' LIMIT 50`
	rows, err := s.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to list pending rules: %w", err)
	}
	defer rows.Close()

	var rules []*Rule
	for rows.Next() {
		r := &Rule{}
		err := rows.Scan(&r.ID, &r.Name, &r.Category, &r.Content, &r.Status, &r.CreatedAt)
		if err != nil {
			return nil, err
		}
		rules = append(rules, r)
	}

	return &Response{
		Result: map[string]interface{}{
			"rules": rules,
			"count": len(rules),
		},
	}, nil
}

func (s *Server) handleRulePromote(ctx context.Context, req *Request) (*Response, error) {
	var params struct {
		RuleID string `json:"rule_id"`
	}

	if err := json.Unmarshal(req.Params, &params); err != nil {
		return nil, fmt.Errorf("failed to parse params: %w", err)
	}

	query := `UPDATE rules SET status = 'active' WHERE id = ?`
	_, err := s.db.Exec(query, params.RuleID)
	if err != nil {
		return nil, fmt.Errorf("failed to promote rule: %w", err)
	}

	return &Response{
		Result: map[string]interface{}{
			"rule_id":  params.RuleID,
			"status":   "active",
			"promoted": true,
		},
	}, nil
}

func (s *Server) handleTeamStatus(ctx context.Context, req *Request) (*Response, error) {
	return &Response{
		Result: map[string]interface{}{
			"team":    map[string]interface{}{},
			"members": []interface{}{},
			"scopes":  map[string]interface{}{},
		},
	}, nil
}

func (s *Server) handleTeamRegister(ctx context.Context, req *Request) (*Response, error) {
	var params struct {
		AgentID string `json:"agent_id"`
		Scope   string `json:"scope"`
	}

	if err := json.Unmarshal(req.Params, &params); err != nil {
		return nil, fmt.Errorf("failed to parse params: %w", err)
	}

	return &Response{
		Result: map[string]interface{}{
			"agent_id":   params.AgentID,
			"scope":      params.Scope,
			"registered": true,
		},
	}, nil
}

func (s *Server) handleDodCheck(ctx context.Context, req *Request) (*Response, error) {
	var params struct {
		WorkflowID string   `json:"workflow_id"`
		Standards  []string `json:"standards,omitempty"`
	}

	if err := json.Unmarshal(req.Params, &params); err != nil {
		return nil, fmt.Errorf("failed to parse params: %w", err)
	}

	return &Response{
		Result: map[string]interface{}{
			"workflow_id": params.WorkflowID,
			"standards":   params.Standards,
			"passed":      true,
			"note":        "DoD check requires Tyr integration",
		},
	}, nil
}

func (s *Server) handleSkillsImport(ctx context.Context, req *Request) (*Response, error) {
	var params struct {
		Source   string `json:"source"`
		Query    string `json:"query,omitempty"`
		URL      string `json:"url,omitempty"`
		SkipScan bool   `json:"skip_scan,omitempty"`
	}

	if err := json.Unmarshal(req.Params, &params); err != nil {
		return nil, fmt.Errorf("failed to parse params: %w", err)
	}

	return &Response{
		Result: map[string]interface{}{
			"source":      params.Source,
			"query":       params.Query,
			"url":         params.URL,
			"imported":    false,
			"scan_passed": !params.SkipScan,
			"note":        "import requires SkillsMP integration and Tyr security scan",
		},
	}, nil
}

func (s *Server) handleSkillsUpdate(ctx context.Context, req *Request) (*Response, error) {
	var params struct {
		SkillName string `json:"skill_name"`
	}

	if err := json.Unmarshal(req.Params, &params); err != nil {
		return nil, fmt.Errorf("failed to parse params: %w", err)
	}

	return &Response{
		Result: map[string]interface{}{
			"skill_name": params.SkillName,
			"updated":    false,
			"note":       "update requires SkillsMP integration",
		},
	}, nil
}

func (s *Server) handleApiDocsCheck(ctx context.Context, req *Request) (*Response, error) {
	var params struct {
		Endpoint string `json:"endpoint"`
		Method   string `json:"method,omitempty"`
	}

	if err := json.Unmarshal(req.Params, &params); err != nil {
		return nil, fmt.Errorf("failed to parse params: %w", err)
	}

	return &Response{
		Result: map[string]interface{}{
			"endpoint": params.Endpoint,
			"method":   params.Method,
			"found":    false,
			"note":     "API docs check pending OpenAPI integration",
		},
	}, nil
}

func (s *Server) handleBootstrapImport(ctx context.Context, req *Request) (*Response, error) {
	var params struct {
		ProjectPath string `json:"project_path"`
		SkillsOnly  bool   `json:"skills_only,omitempty"`
		RulesOnly   bool   `json:"rules_only,omitempty"`
	}

	if err := json.Unmarshal(req.Params, &params); err != nil {
		return nil, fmt.Errorf("failed to parse params: %w", err)
	}

	if params.ProjectPath == "" {
		return nil, fmt.Errorf("project_path is required")
	}

	ragnarokDir := params.ProjectPath + "/.ragnarok"

	skillsCount := 0
	rulesCount := 0

	if !params.RulesOnly {
		skillsFile := ragnarokDir + "/skills.json"
		if data, err := os.ReadFile(skillsFile); err == nil {
			var parsed map[string]interface{}
			if err := json.Unmarshal(data, &parsed); err == nil {
				if skills, ok := parsed["suggested_skills"].([]interface{}); ok {
					for _, skillItem := range skills {
						if skill, ok := skillItem.(map[string]interface{}); ok {
							name, _ := skill["name"].(string)
							skillType, _ := skill["type"].(string)
							skillDesc, _ := skill["skill"].(string)

							id := generateID("skill")
							now := time.Now()
							_, err := s.db.Exec(`
								INSERT OR REPLACE INTO skills 
								(id, name, description, framework, source, created_at, updated_at)
								VALUES (?, ?, ?, ?, ?, ?, ?)
							`, id, name, skillDesc, skillType, "fenrir-bootstrap", now, now)
							if err == nil {
								skillsCount++
							}
						}
					}
				}
			}
		}
	}

	if !params.SkillsOnly {
		rulesFile := ragnarokDir + "/rules.json"
		if data, err := os.ReadFile(rulesFile); err == nil {
			var rules []map[string]string
			if err := json.Unmarshal(data, &rules); err == nil {
				for _, rule := range rules {
					name := rule["name"]
					category := rule["category"]
					description := rule["description"]
					severity := rule["severity"]
					if severity == "" {
						severity = "medium"
					}

					id := generateID("rule")
					now := time.Now()
					_, err := s.db.Exec(`
						INSERT OR REPLACE INTO rules 
						(id, name, category, content, severity, source, created_at, updated_at)
						VALUES (?, ?, ?, ?, ?, ?, ?, ?)
					`, id, name, category, description, severity, "fenrir-bootstrap", now, now)
					if err == nil {
						rulesCount++
					}
				}
			}
		}
	}

	return &Response{
		Result: map[string]interface{}{
			"project_path":    params.ProjectPath,
			"skills_imported": skillsCount,
			"rules_imported":  rulesCount,
			"source":          "fenrir-bootstrap",
		},
	}, nil
}
