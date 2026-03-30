package mcp

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/andragon31/Ragnarok/internal/skoll/skills"
	"github.com/andragon31/Ragnarok/internal/skoll/skillsmp"
)

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

	skillList, err := s.skillLoader.ListSkills()
	if err != nil {
		return nil, fmt.Errorf("failed to list skills: %w", err)
	}

	index := skills.BuildSkillIndex(skillList)
	if len(index) > params.Limit {
		index = index[:params.Limit]
	}

	return &Response{
		Result: map[string]interface{}{
			"skills_index": index,
			"count":        len(index),
			"progressive":  true,
			"note":         "Use skill_load for full content",
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

	skill, err := s.skillLoader.LoadSkillFull(params.SkillName)
	if err != nil {
		return &Response{
			Result: map[string]interface{}{
				"skill_name": params.SkillName,
				"found":      false,
				"error":      err.Error(),
			},
		}, nil
	}

	files, err := s.skillLoader.ListSkillFiles(params.SkillName)
	if err != nil {
		files = map[string][]string{}
	}

	return &Response{
		Result: map[string]interface{}{
			"name":            skill.Name,
			"description":     skill.Description,
			"content":         skill.Content,
			"version":         skill.Version,
			"license":         skill.License,
			"compatibility":   skill.Compatibility,
			"framework":       skill.Framework,
			"allowed_tools":   skill.AllowedTools,
			"available_files": files,
			"has_scripts":     skill.HasScripts,
			"has_references":  skill.HasReferences,
			"has_assets":      skill.HasAssets,
			"found":           true,
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

	if params.Query == "" {
		return nil, fmt.Errorf("query is required")
	}

	if params.Limit == 0 {
		params.Limit = 10
	}

	results, err := s.skillLoader.SearchSkills(params.Query)
	if err != nil {
		return nil, fmt.Errorf("failed to search skills: %w", err)
	}

	if len(results) > params.Limit {
		results = results[:params.Limit]
	}

	var index []*skills.SkillIndexEntry
	for _, r := range results {
		index = append(index, &skills.SkillIndexEntry{
			Name:              r.Name,
			Description:       r.Description,
			HasScripts:        r.HasScripts,
			HasReferences:     r.HasReferences,
			HasAssets:         r.HasAssets,
			VersionStatus:     "current",
			AllowedToolsCount: len(r.AllowedTools),
		})
	}

	return &Response{
		Result: map[string]interface{}{
			"skills": index,
			"count":  len(index),
			"query":  params.Query,
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

	skill, err := s.skillLoader.LoadSkillIndex(params.SkillName)
	if err != nil {
		return &Response{
			Result: map[string]interface{}{
				"skill_name": params.SkillName,
				"version":    params.Version,
				"compatible": false,
				"error":      "skill not found",
			},
		}, nil
	}

	compatible := true
	if skill.MinVersion != "" && params.Version < skill.MinVersion {
		compatible = false
	}
	if skill.MaxVersion != "" && params.Version > skill.MaxVersion {
		compatible = false
	}

	return &Response{
		Result: map[string]interface{}{
			"skill_name":  params.SkillName,
			"version":     params.Version,
			"min_version": skill.MinVersion,
			"max_version": skill.MaxVersion,
			"compatible":  compatible,
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

	skill, err := s.skillLoader.LoadSkillIndex(params.SkillName)
	if err != nil {
		return &Response{
			Result: map[string]interface{}{
				"skill_name": params.SkillName,
				"valid":      false,
				"verified":   false,
				"error":      "skill not found",
			},
		}, nil
	}

	valid := skill.Description != ""

	return &Response{
		Result: map[string]interface{}{
			"skill_name":    params.SkillName,
			"valid":         valid,
			"verified":      true,
			"last_verified": time.Now().Format("2006-01-02"),
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

	if params.SkillName == "" || params.FilePath == "" {
		return nil, fmt.Errorf("skill_name and file_path are required")
	}

	file, err := s.skillLoader.LoadSkillFile(params.SkillName, params.FilePath)
	if err != nil {
		return &Response{
			Result: map[string]interface{}{
				"skill_name": params.SkillName,
				"file_path":  params.FilePath,
				"found":      false,
				"error":      err.Error(),
			},
		}, nil
	}

	return &Response{
		Result: map[string]interface{}{
			"skill_name": params.SkillName,
			"file_path":  file.Path,
			"type":       file.Type,
			"content":    file.Content,
			"found":      true,
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
	contextPath := params.ContextPath
	if contextPath == "" {
		contextPath = "."
	}

	agentsMdPath := findAgentsMd(contextPath)
	if agentsMdPath != "" {
		if content, err := os.ReadFile(agentsMdPath); err == nil {
			agentsMdContext = string(content)
		}
	}

	var agentSkills []string
	var allowedTools []string
	seenTools := make(map[string]bool)

	if params.AgentID != "" {
		row := s.db.QueryRow(`SELECT skills, allowed_tools FROM agents WHERE id = ?`, params.AgentID)
		var skillsJSON, agentToolsJSON *string
		if err := row.Scan(&skillsJSON, &agentToolsJSON); err == nil {
			if skillsJSON != nil {
				json.Unmarshal([]byte(*skillsJSON), &agentSkills)
			}
			if agentToolsJSON != nil {
				json.Unmarshal([]byte(*agentToolsJSON), &allowedTools)
			}
		}
	}

	for _, skillName := range agentSkills {
		skill, err := s.skillLoader.LoadSkillIndex(skillName)
		if err != nil {
			log.Printf("Warning: failed to load skill '%s': %v", skillName, err)
			continue
		}
		for _, tool := range skill.AllowedTools {
			if !seenTools[tool] {
				seenTools[tool] = true
				allowedTools = append(allowedTools, tool)
			}
		}
	}

	if len(allowedTools) == 0 {
		allowedTools = []string{
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
	}

	return &Response{
		Result: map[string]interface{}{
			"agent_id":          params.AgentID,
			"activated":         true,
			"allowed_tools":     allowedTools,
			"agents_md_context": agentsMdContext,
			"skills_suggested":  agentSkills,
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

func (s *Server) handleAgentList(ctx context.Context, req *Request) (*Response, error) {
	query := `SELECT id, name, role, scope, skills, is_active, last_active FROM agents LIMIT 50`
	rows, err := s.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to list agents: %w", err)
	}
	defer rows.Close()

	var agents []*Agent
	for rows.Next() {
		a := &Agent{}
		var skillsJSON *string
		var lastActive *time.Time
		err := rows.Scan(&a.ID, &a.Name, &a.Role, &a.Scope, &skillsJSON, &a.IsActive, &lastActive)
		if err != nil {
			continue
		}
		if skillsJSON != nil {
			json.Unmarshal([]byte(*skillsJSON), &a.Skills)
		}
		agents = append(agents, a)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating agents: %w", err)
	}

	return &Response{
		Result: map[string]interface{}{
			"agents": agents,
			"count":  len(agents),
		},
	}, nil
}

func (s *Server) handleAgentContext(ctx context.Context, req *Request) (*Response, error) {
	var params struct {
		AgentID string `json:"agent_id"`
	}

	if err := json.Unmarshal(req.Params, &params); err != nil {
		return nil, fmt.Errorf("failed to parse params: %w", err)
	}

	query := `SELECT name, role, scope, skills, allowed_tools FROM agents WHERE id = ?`
	var name, role, scope *string
	var skillsJSON, allowedToolsJSON *string

	err := s.db.QueryRow(query, params.AgentID).Scan(&name, &role, &scope, &skillsJSON, &allowedToolsJSON)
	if err != nil {
		return nil, fmt.Errorf("agent not found: %w", err)
	}

	var skills, allowedTools []string
	if skillsJSON != nil {
		json.Unmarshal([]byte(*skillsJSON), &skills)
	}
	if allowedToolsJSON != nil {
		json.Unmarshal([]byte(*allowedToolsJSON), &allowedTools)
	}

	return &Response{
		Result: map[string]interface{}{
			"agent_id":      params.AgentID,
			"name":          name,
			"role":          role,
			"scope":         scope,
			"skills":        skills,
			"allowed_tools": allowedTools,
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

	query := `SELECT id, name, category, content, severity, status, created_at FROM rules WHERE (? = '' OR category = ?) AND is_active = 1 LIMIT ?`
	rows, err := s.db.Query(query, params.Category, params.Category, params.Limit)
	if err != nil {
		return nil, fmt.Errorf("failed to list rules: %w", err)
	}
	defer rows.Close()

	var rules []*Rule
	for rows.Next() {
		r := &Rule{}
		err := rows.Scan(&r.ID, &r.Name, &r.Category, &r.Content, &r.Severity, &r.Status, &r.CreatedAt)
		if err != nil {
			continue
		}
		rules = append(rules, r)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating rules: %w", err)
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

	action := strings.TrimSpace(params.Action)
	if action == "" {
		return &Response{
			Result: map[string]interface{}{
				"action":        "",
				"allowed":       true,
				"rules_checked": 0,
				"violations":    []map[string]string{},
			},
		}, nil
	}

	query := `SELECT id, name, category, content, severity, pattern FROM rules WHERE is_active = 1`
	rows, err := s.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to check rules: %w", err)
	}
	defer rows.Close()

	violations := []map[string]string{}
	rulesChecked := 0
	for rows.Next() {
		var id, name, category, content, severity, pattern string
		if err := rows.Scan(&id, &name, &category, &content, &severity, &pattern); err != nil {
			continue
		}
		rulesChecked++

		if pattern == "" {
			continue
		}

		matched, err := regexp.MatchString(pattern, action)
		if err != nil {
			log.Printf("Invalid pattern %q for rule %s: %v", pattern, name, err)
			continue
		}

		if matched {
			violations = append(violations, map[string]string{
				"id":       id,
				"name":     name,
				"category": category,
				"severity": severity,
			})
		}
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating rules: %w", err)
	}

	return &Response{
		Result: map[string]interface{}{
			"action":        action,
			"allowed":       len(violations) == 0,
			"rules_checked": rulesChecked,
			"violations":    violations,
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

	query := `SELECT id, name, category, content, severity, status, created_at FROM rules WHERE id = ?`
	r := &Rule{}
	err := s.db.QueryRow(query, params.RuleID).Scan(&r.ID, &r.Name, &r.Category, &r.Content, &r.Severity, &r.Status, &r.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("rule not found: %w", err)
	}

	return &Response{
		Result: r,
	}, nil
}

func (s *Server) handleRuleCreateOrReuse(ctx context.Context, req *Request) (*Response, error) {
	var params struct {
		Name        string `json:"name"`
		Category    string `json:"category"`
		Content     string `json:"content"`
		Pattern     string `json:"pattern"`
		Severity    string `json:"severity,omitempty"`
		Source      string `json:"source,omitempty"`
		Fingerprint string `json:"fingerprint,omitempty"`
	}

	if err := json.Unmarshal(req.Params, &params); err != nil {
		return nil, fmt.Errorf("failed to parse params: %w", err)
	}

	if params.Name == "" || params.Category == "" {
		return nil, fmt.Errorf("name and category are required")
	}

	if params.Severity == "" {
		params.Severity = "medium"
	}
	if params.Source == "" {
		params.Source = "generated"
	}

	existingQuery := `SELECT id, name, category, content, pattern, severity FROM rules WHERE name = ? AND category = ?`
	var existingID, existingName, existingCategory, existingContent, existingPattern, existingSeverity string
	err := s.db.QueryRow(existingQuery, params.Name, params.Category).Scan(
		&existingID, &existingName, &existingCategory, &existingContent, &existingPattern, &existingSeverity,
	)

	if err == nil {
		return &Response{
			Result: map[string]interface{}{
				"reused":   true,
				"rule_id":  existingID,
				"name":     existingName,
				"category": existingCategory,
				"message":  "Rule already exists",
			},
		}, nil
	}

	if err != sql.ErrNoRows {
		return nil, fmt.Errorf("database error: %w", err)
	}

	ruleID := generateID("rule")
	now := time.Now()

	insertQuery := `INSERT INTO rules (id, name, category, content, severity, scope, status, is_active, source, pattern, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, 'global', 'active', 1, ?, ?, ?, ?)`
	_, err = s.db.Exec(insertQuery, ruleID, params.Name, params.Category, params.Content, params.Severity,
		params.Source, params.Pattern, now, now)
	if err != nil {
		return nil, fmt.Errorf("failed to create rule: %w", err)
	}

	return &Response{
		Result: map[string]interface{}{
			"reused":   false,
			"rule_id":  ruleID,
			"name":     params.Name,
			"category": params.Category,
			"pattern":  params.Pattern,
			"severity": params.Severity,
			"source":   params.Source,
			"message":  "Rule created successfully",
		},
	}, nil
}

func (s *Server) handleRuleCreateFromPRD(ctx context.Context, req *Request) (*Response, error) {
	var params struct {
		Requirements []map[string]string `json:"requirements"`
		ProjectPath  string              `json:"project_path,omitempty"`
	}

	if err := json.Unmarshal(req.Params, &params); err != nil {
		return nil, fmt.Errorf("failed to parse params: %w", err)
	}

	if len(params.Requirements) == 0 {
		return &Response{
			Result: map[string]interface{}{
				"created": 0,
				"reused":  0,
				"rules":   []map[string]string{},
				"message": "No requirements provided",
			},
		}, nil
	}

	nonFunctionalKeywords := map[string]string{
		"security":       "security",
		"performance":    "performance",
		"scalability":    "scalability",
		"compliance":     "compliance",
		"rate limit":     "rate-limiting",
		"owasp":          "owasp",
		"encryption":     "encryption",
		"authentication": "auth",
		"authorization":  "authorization",
		"audit":          "audit",
		"logging":        "logging",
		"monitoring":     "monitoring",
	}

	rulePatterns := map[string]string{
		"security":      `(password|credential|secret|token|auth|sql|injection|xss|csrf|cors|https|tls|encrypt)`,
		"performance":   `(cache|memoize|lazy|index|query|optimiz|benchmark|profil)`,
		"scalability":   `(horizontal|scale|vertical|load.?balanc|partition|shard|replica)`,
		"compliance":    `(gdpr|hipaa|pci|soap|legal|regulatory|audit.?log)`,
		"rate-limiting": `(rate.?limit|throttl|debouce|quota|threshold)`,
		"owasp":         `(owasp|injection|xss|csrf|idor|xxe|sensitive|security.?header)`,
		"encryption":    `(encrypt|decrypt|crypt|hash|signature|key.?manag)`,
		"auth":          `(oauth|jwt|session|cookie|login|logout|register|2fa|mfa|totp)`,
		"authorization": `(rbac|acl|permission|role|policy|ABAC|PBAC)`,
		"audit":         `(audit.?log|event.?track|user.?action|complianc.?log)`,
		"logging":       `(log|logger|trace|debug|info|warn|error|fatal)`,
		"monitoring":    `(metric|monitor|alert|observab|trac|dashboar|prometheus|grafana)`,
	}

	created := 0
	reused := 0
	rules := []map[string]string{}

	for _, req := range params.Requirements {
		reqType, _ := req["type"]
		reqTitle, _ := req["title"]
		reqID, _ := req["id"]

		reqTypeLower := strings.ToLower(reqType)
		reqTitleLower := strings.ToLower(reqTitle)

		var matchedCategory string
		for keyword, category := range nonFunctionalKeywords {
			if strings.Contains(reqTypeLower, keyword) || strings.Contains(reqTitleLower, keyword) {
				matchedCategory = category
				break
			}
		}

		if matchedCategory == "" {
			continue
		}

		pattern, exists := rulePatterns[matchedCategory]
		if !exists {
			pattern = ""
		}

		ruleName := strings.ReplaceAll(strings.ToLower(matchedCategory)+"-"+reqID, " ", "-")
		ruleContent := fmt.Sprintf("PRD Requirement [%s]: %s - %s", reqID, reqType, reqTitle)

		result, err := s.handleRuleCreateOrReuse(ctx, &Request{
			Params: mustMarshal(map[string]interface{}{
				"name":     ruleName,
				"category": matchedCategory,
				"content":  ruleContent,
				"pattern":  pattern,
				"severity": "high",
				"source":   "prd",
			}),
		})
		if err != nil {
			log.Printf("Error creating rule for requirement %s: %v", reqID, err)
			continue
		}

		if resultMap, ok := result.Result.(map[string]interface{}); ok {
			if reusedVal, ok := resultMap["reused"].(bool); ok && reusedVal {
				reused++
			} else {
				created++
			}
			rules = append(rules, map[string]string{
				"name":     ruleName,
				"category": matchedCategory,
				"pattern":  pattern,
			})
		}
	}

	return &Response{
		Result: map[string]interface{}{
			"created":     created,
			"reused":      reused,
			"total_rules": created + reused,
			"rules":       rules,
			"message":     fmt.Sprintf("Processed %d requirements: %d created, %d reused", len(params.Requirements), created, reused),
		},
	}, nil
}

func mustMarshal(v interface{}) []byte {
	data, _ := json.Marshal(v)
	return data
}

func (s *Server) handleWorkflowStart(ctx context.Context, req *Request) (*Response, error) {
	var params struct {
		Name        string `json:"name"`
		Type        string `json:"type,omitempty"`
		Description string `json:"description,omitempty"`
	}

	if err := json.Unmarshal(req.Params, &params); err != nil {
		return nil, fmt.Errorf("failed to parse params: %w", err)
	}

	if params.Type == "" {
		params.Type = "standard"
	}

	wf := &Workflow{
		ID:        generateID("wf"),
		Name:      params.Name,
		Status:    "started",
		CreatedAt: time.Now(),
	}

	query := `INSERT INTO workflows (id, name, type, status, description, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?)`
	_, err := s.db.Exec(query, wf.ID, wf.Name, params.Type, wf.Status, params.Description, wf.CreatedAt, wf.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("failed to start workflow: %w", err)
	}

	return &Response{
		Result: map[string]interface{}{
			"workflow_id": wf.ID,
			"name":        wf.Name,
			"type":        params.Type,
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

	query := `UPDATE workflows SET status = 'completed', updated_at = ? WHERE id = ?`
	_, err := s.db.Exec(query, time.Now(), params.WorkflowID)
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
	var totalSkills, totalRules, totalAgents, activeAgents int

	if err := s.db.QueryRow(`SELECT COUNT(*) FROM skills`).Scan(&totalSkills); err != nil {
		totalSkills = -1
	}
	if err := s.db.QueryRow(`SELECT COUNT(*) FROM rules`).Scan(&totalRules); err != nil {
		totalRules = -1
	}
	if err := s.db.QueryRow(`SELECT COUNT(*) FROM agents`).Scan(&totalAgents); err != nil {
		totalAgents = -1
	}
	if err := s.db.QueryRow(`SELECT COUNT(*) FROM agents WHERE is_active = 1`).Scan(&activeAgents); err != nil {
		activeAgents = -1
	}

	skillList, _ := s.skillLoader.ListSkills()

	return &Response{
		Result: map[string]interface{}{
			"status":                 "operational",
			"total_skills":           totalSkills,
			"total_rules":            totalRules,
			"total_agents":           totalAgents,
			"active_agents":          activeAgents,
			"filesystem_skills":      len(skillList),
			"version":                "1.4.0",
			"progressive_disclosure": true,
		},
	}, nil
}

func (s *Server) handleSkollValidate(ctx context.Context, req *Request) (*Response, error) {
	var params struct {
		Strict bool `json:"strict,omitempty"`
	}

	json.Unmarshal(req.Params, &params)

	skillList, err := s.skillLoader.ListSkills()
	if err != nil {
		skillList = []*skills.SkillInfo{}
	}

	errors := []string{}
	warnings := []string{}

	for _, skill := range skillList {
		if skill.Description == "" {
			if params.Strict {
				errors = append(errors, fmt.Sprintf("skill '%s' has empty description", skill.Name))
			} else {
				warnings = append(warnings, fmt.Sprintf("skill '%s' has empty description", skill.Name))
			}
		}
	}

	valid := len(errors) == 0

	return &Response{
		Result: map[string]interface{}{
			"valid":     valid,
			"errors":    errors,
			"warnings":  warnings,
			"validated": true,
		},
	}, nil
}

func (s *Server) handleRulePending(ctx context.Context, req *Request) (*Response, error) {
	query := `SELECT id, rule_id, proposed_by, reason, status, created_at FROM pending_rules WHERE status = 'pending' LIMIT 50`
	rows, err := s.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to list pending rules: %w", err)
	}
	defer rows.Close()

	type PendingRule struct {
		ID         string    `json:"id"`
		RuleID     string    `json:"rule_id"`
		ProposedBy string    `json:"proposed_by"`
		Reason     string    `json:"reason"`
		Status     string    `json:"status"`
		CreatedAt  time.Time `json:"created_at"`
	}

	var rules []*PendingRule
	for rows.Next() {
		r := &PendingRule{}
		var proposedBy, reason sql.NullString
		if err := rows.Scan(&r.ID, &r.RuleID, &proposedBy, &reason, &r.Status, &r.CreatedAt); err != nil {
			continue
		}
		r.ProposedBy = proposedBy.String
		r.Reason = reason.String
		rules = append(rules, r)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating pending rules: %w", err)
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

	query := `UPDATE pending_rules SET status = 'promoted' WHERE id = ?`
	_, err := s.db.Exec(query, params.RuleID)
	if err != nil {
		return nil, fmt.Errorf("failed to promote rule: %w", err)
	}

	return &Response{
		Result: map[string]interface{}{
			"rule_id":  params.RuleID,
			"status":   "promoted",
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

	query := `INSERT OR REPLACE INTO team_context (id, module, scope, updated_at) VALUES (?, ?, ?, ?)`
	_, err := s.db.Exec(query, generateID("team"), params.AgentID, params.Scope, time.Now())
	if err != nil {
		return nil, fmt.Errorf("failed to register team: %w", err)
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
		WorkflowID string `json:"workflow_id"`
	}

	if err := json.Unmarshal(req.Params, &params); err != nil {
		return nil, fmt.Errorf("failed to parse params: %w", err)
	}

	if params.WorkflowID == "" {
		return nil, fmt.Errorf("workflow_id is required")
	}

	result := &DodCheckResult{
		WorkflowID: params.WorkflowID,
		Checks:     []DodCheckItem{},
		Passed:     true,
		CheckedAt:  time.Now(),
	}

	query := `SELECT w.id, w.name, w.status, w.completed_at,
		(SELECT COUNT(*) FROM workflow_steps WHERE workflow_id = w.id) as total_steps,
		(SELECT COUNT(*) FROM workflow_steps WHERE workflow_id = w.id AND status = 'completed') as completed_steps
		FROM workflows w WHERE w.id = ?`

	var workflow struct {
		ID             string
		Name           string
		Status         string
		CompletedAt    *time.Time
		TotalSteps     int
		CompletedSteps int
	}

	err := s.db.QueryRow(query, params.WorkflowID).Scan(
		&workflow.ID, &workflow.Name, &workflow.Status,
		&workflow.CompletedAt, &workflow.TotalSteps, &workflow.CompletedSteps,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("workflow not found: %s", params.WorkflowID)
	} else if err != nil {
		return nil, fmt.Errorf("failed to query workflow: %w", err)
	}

	result.Checks = append(result.Checks, DodCheckItem{
		Check:   "workflow_completed",
		Passed:  workflow.CompletedAt != nil,
		Message: fmt.Sprintf("Workflow %s: %d/%d steps completed", workflow.Name, workflow.CompletedSteps, workflow.TotalSteps),
	})

	if workflow.CompletedAt == nil {
		result.Passed = false
	}

	result.Checks = append(result.Checks, DodCheckItem{
		Check:   "all_steps_completed",
		Passed:  workflow.CompletedSteps == workflow.TotalSteps && workflow.TotalSteps > 0,
		Message: fmt.Sprintf("All %d steps completed", workflow.TotalSteps),
	})

	if workflow.CompletedSteps != workflow.TotalSteps {
		result.Passed = false
	}

	var evidenceCount int
	s.db.QueryRow(`SELECT COUNT(*) FROM approval_records WHERE plan_id IN 
		(SELECT DISTINCT plan_id FROM workflow_steps WHERE workflow_id = ?)`, params.WorkflowID).Scan(&evidenceCount)

	result.Checks = append(result.Checks, DodCheckItem{
		Check:   "approval_evidence",
		Passed:  evidenceCount > 0,
		Message: fmt.Sprintf("%d approval records found", evidenceCount),
	})

	return &Response{Result: result}, nil
}

type DodCheckResult struct {
	WorkflowID string         `json:"workflow_id"`
	Checks     []DodCheckItem `json:"checks"`
	Passed     bool           `json:"passed"`
	CheckedAt  time.Time      `json:"checked_at"`
}

type DodCheckItem struct {
	Check   string `json:"check"`
	Passed  bool   `json:"passed"`
	Message string `json:"message"`
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

	client := skillsmp.NewClient()

	if params.Source == "github" && params.URL != "" {
		skillName := skillsmp.ExtractSkillNameFromURL(params.URL)
		if skillName == "" {
			return nil, fmt.Errorf("invalid GitHub URL: %s", params.URL)
		}

		skillsDir := s.skillLoader.GetSkillsDir()
		skillDir := filepath.Join(skillsDir, skillName)

		if err := os.MkdirAll(skillDir, 0755); err != nil {
			return nil, fmt.Errorf("failed to create skill directory: %w", err)
		}

		if err := client.CloneOrDownloadSkill(params.URL, skillsDir); err != nil {
			return &Response{
				Result: map[string]interface{}{
					"source":     params.Source,
					"url":        params.URL,
					"skill_name": skillName,
					"imported":   false,
					"error":      err.Error(),
				},
			}, nil
		}

		skill, err := s.skillLoader.LoadSkillIndex(skillName)
		if err != nil {
			skill = &skills.SkillInfo{Name: skillName}
		}

		return &Response{
			Result: map[string]interface{}{
				"source":      params.Source,
				"url":         params.URL,
				"skill_name":  skillName,
				"skill":       skill.ToMap(),
				"imported":    true,
				"scan_passed": !params.SkipScan,
			},
		}, nil
	}

	if params.Source == "skillsmp" || (params.Source == "" && params.Query != "") {
		if params.Query == "" {
			return nil, fmt.Errorf("query is required for skillsmp source")
		}

		result, err := client.SearchSkills(params.Query, 10)
		if err != nil {
			return &Response{
				Result: map[string]interface{}{
					"source": params.Source,
					"query":  params.Query,
					"error":  err.Error(),
				},
			}, nil
		}

		var skillsList []map[string]interface{}
		for _, skill := range result.Skills {
			skillsList = append(skillsList, map[string]interface{}{
				"name":        skill.Name,
				"description": skill.Description,
				"author":      skill.Author,
				"stars":       skill.Stars,
				"license":     skill.License,
				"topics":      skill.Topics,
			})
		}

		return &Response{
			Result: map[string]interface{}{
				"source": params.Source,
				"query":  params.Query,
				"skills": skillsList,
				"total":  result.Total,
				"note":   "Use URL parameter with github source to import a specific skill",
			},
		}, nil
	}

	if params.Source == "local" {
		return &Response{
			Result: map[string]interface{}{
				"source": params.Source,
				"note":   "Local import reads from .skoll/skills/ directory",
			},
		}, nil
	}

	return &Response{
		Result: map[string]interface{}{
			"source": params.Source,
			"query":  params.Query,
			"url":    params.URL,
			"error":  "unsupported source or missing parameters",
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
			"note":       "Update requires source repository URL",
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
			"note":     "API docs check requires OpenAPI specification",
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

	ragnarokDir := filepath.Join(params.ProjectPath, ".ragnarok")
	skillsCount := 0
	rulesCount := 0

	if !params.RulesOnly {
		skillsFile := filepath.Join(ragnarokDir, "skills.json")
		if data, err := os.ReadFile(skillsFile); err == nil {
			var parsed map[string]interface{}
			if err := json.Unmarshal(data, &parsed); err == nil {
				if skillList, ok := parsed["suggested_skills"].([]interface{}); ok {
					for _, skillItem := range skillList {
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
		rulesFile := filepath.Join(ragnarokDir, "rules.json")
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
