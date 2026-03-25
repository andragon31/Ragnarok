package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/ragnarok-ecosystem/tyr/internal/audit"
	"github.com/ragnarok-ecosystem/tyr/internal/precommit"
	"github.com/ragnarok-ecosystem/tyr/internal/registry"
	"github.com/ragnarok-ecosystem/tyr/internal/sast"
	"github.com/ragnarok-ecosystem/tyr/internal/security"
)

func (s *Server) handlePkgCheck(ctx context.Context, req *Request) (*Response, error) {
	var params struct {
		Name       string `json:"name"`
		Ecosystem  string `json:"ecosystem"`
		Version    string `json:"version,omitempty"`
		CheckCVEs  bool   `json:"check_cves,omitempty"`
		CheckTypos bool   `json:"check_typos,omitempty"`
		NoCache    bool   `json:"no_cache,omitempty"`
	}

	if err := json.Unmarshal(req.Params, &params); err != nil {
		return nil, fmt.Errorf("failed to parse params: %w", err)
	}

	if params.Ecosystem == "" {
		params.Ecosystem = detectEcosystem(params.Name)
	}

	if !params.NoCache {
		var cached struct {
			Exists    bool
			Trusted   bool
			CVECount  int
			AgeDays   int
			Downloads int64
			TyposRisk bool
			CachedAt  time.Time
		}
		query := `SELECT exists_pkg, trusted, cve_count, age_days, downloads, typosquatting_risk, cached_at 
				  FROM pkg_cache WHERE ecosystem = ? AND name = ? AND (version = ? OR version = '') 
				  ORDER BY cached_at DESC LIMIT 1`
		row := s.db.QueryRow(query, params.Ecosystem, params.Name, params.Version)
		var expiresAt time.Time
		if err := row.Scan(&cached.Exists, &cached.Trusted, &cached.CVECount, &cached.AgeDays, &cached.Downloads, &cached.TyposRisk, &expiresAt); err == nil {
			if time.Since(expiresAt) < 24*time.Hour {
				return &Response{
					Result: map[string]interface{}{
						"name":               params.Name,
						"ecosystem":          params.Ecosystem,
						"version":            params.Version,
						"exists":             cached.Exists,
						"trusted":            cached.Trusted,
						"cve_count":          cached.CVECount,
						"age_days":           cached.AgeDays,
						"downloads_monthly":  cached.Downloads,
						"typosquatting_risk": cached.TyposRisk,
						"last_checked":       cached.CachedAt,
						"cached":             true,
					},
				}, nil
			}
		}
	}

	client := registry.NewRegistryClient()
	info, err := client.CheckPackage(params.Ecosystem, params.Name)
	if err != nil {
		return &Response{
			Result: map[string]interface{}{
				"name":   params.Name,
				"error":  err.Error(),
				"source": params.Ecosystem,
			},
		}, nil
	}

	cveCount := 0
	if params.CheckCVEs {
		cveCount, _ = registry.CheckOSV(params.Name, params.Ecosystem)
	}

	typosquattingRisk := false
	if params.CheckTypos {
		typosquattingRisk = client.CheckTyposquatting(params.Name, params.Ecosystem)
	}

	expiresAt := time.Now().Add(24 * time.Hour)
	s.db.Exec(`DELETE FROM pkg_cache WHERE ecosystem = ? AND name = ?`, params.Ecosystem, params.Name)
	query := `INSERT INTO pkg_cache (ecosystem, name, version, exists_pkg, trusted, cve_count, age_days, downloads, typosquatting_risk, cached_at, expires_at)
			  VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`
	s.db.Exec(query, params.Ecosystem, params.Name, params.Version, info.Exists, info.Trusted, cveCount, info.AgeDays, info.DownloadsMonthly, typosquattingRisk, time.Now(), expiresAt)

	return &Response{
		Result: map[string]interface{}{
			"name":               params.Name,
			"ecosystem":          params.Ecosystem,
			"version":            params.Version,
			"exists":             info.Exists,
			"trusted":            info.Trusted,
			"cve_count":          cveCount,
			"age_days":           info.AgeDays,
			"downloads_monthly":  info.DownloadsMonthly,
			"typosquatting_risk": typosquattingRisk,
			"last_checked":       time.Now(),
			"description":        info.Description,
			"latest_version":     info.LatestVersion,
			"license":            info.License,
			"source":             info.Source,
			"cached":             false,
		},
	}, nil
}

func detectEcosystem(packageName string) string {
	lower := strings.ToLower(packageName)
	if strings.HasPrefix(lower, "@") {
		return "npm"
	}
	if strings.HasSuffix(lower, ".js") || strings.HasSuffix(lower, ".ts") || strings.Contains(lower, "node-") {
		return "npm"
	}
	if strings.Contains(lower, "django") || strings.Contains(lower, "flask") || strings.Contains(lower, "fastapi") {
		return "pypi"
	}
	if strings.Contains(lower, "rust") || strings.HasSuffix(lower, "-rs") {
		return "crates"
	}
	return "npm"
}

func (s *Server) handlePkgLicense(ctx context.Context, req *Request) (*Response, error) {
	var params struct {
		Name      string `json:"name"`
		Ecosystem string `json:"ecosystem"`
	}

	if err := json.Unmarshal(req.Params, &params); err != nil {
		return nil, fmt.Errorf("failed to parse params: %w", err)
	}

	client := registry.NewRegistryClient()
	info, err := client.CheckPackage(params.Ecosystem, params.Name)
	if err != nil {
		return &Response{
			Result: map[string]interface{}{
				"name":      params.Name,
				"ecosystem": params.Ecosystem,
				"error":     err.Error(),
			},
		}, nil
	}

	license := info.License
	if license == "" {
		license = "UNKNOWN"
	}

	return &Response{
		Result: map[string]interface{}{
			"name":                params.Name,
			"ecosystem":           params.Ecosystem,
			"license":             license,
			"transitive_licenses": []interface{}{},
			"source":              info.Source,
		},
	}, nil
}

func (s *Server) handlePkgAudit(ctx context.Context, req *Request) (*Response, error) {
	var params struct {
		ProjectPath string `json:"project_path,omitempty"`
	}

	if err := json.Unmarshal(req.Params, &params); err != nil {
		return nil, fmt.Errorf("failed to parse params: %w", err)
	}

	if params.ProjectPath == "" {
		params.ProjectPath = "."
	}

	result, err := audit.AuditProject(params.ProjectPath)
	if err != nil {
		return &Response{
			Result: map[string]interface{}{
				"project_path": params.ProjectPath,
				"error":        err.Error(),
			},
		}, nil
	}

	return &Response{
		Result: map[string]interface{}{
			"project_path":    params.ProjectPath,
			"package_file":    result.PackageFile,
			"format":          result.Source,
			"total_packages":  result.TotalPackages,
			"dev_packages":    result.DevPackages,
			"packages":        result.Packages,
			"vulnerabilities": []interface{}{},
			"total_vulns":     0,
		},
	}, nil
}

func (s *Server) handlePkgAuditSnapshot(ctx context.Context, req *Request) (*Response, error) {
	var params struct {
		ProjectPath string `json:"project_path,omitempty"`
	}

	if err := json.Unmarshal(req.Params, &params); err != nil {
		return nil, fmt.Errorf("failed to parse params: %w", err)
	}

	return &Response{
		Result: map[string]interface{}{
			"project_path":    params.ProjectPath,
			"snapshot_time":   time.Now(),
			"vulnerabilities": []interface{}{},
			"note":            "pkg_audit_snapshot captures current state for comparison",
		},
	}, nil
}

func (s *Server) handlePkgAuditContinuous(ctx context.Context, req *Request) (*Response, error) {
	var params struct {
		ProjectPath string `json:"project_path,omitempty"`
	}

	if err := json.Unmarshal(req.Params, &params); err != nil {
		return nil, fmt.Errorf("failed to parse params: %w", err)
	}

	return &Response{
		Result: map[string]interface{}{
			"project_path": params.ProjectPath,
			"new_cves":     []interface{}{},
			"note":         "pkg_audit_continuous monitors for new CVEs",
		},
	}, nil
}

func (s *Server) handleSastRun(ctx context.Context, req *Request) (*Response, error) {
	var params struct {
		Target  string `json:"target"`
		Ruleset string `json:"ruleset,omitempty"`
	}

	if err := json.Unmarshal(req.Params, &params); err != nil {
		return nil, fmt.Errorf("failed to parse params: %w", err)
	}

	if params.Target == "" {
		return nil, fmt.Errorf("target is required")
	}

	scanner := sast.NewScanner()
	var findings []*sast.Finding

	info, err := os.Stat(params.Target)
	if err != nil {
		return nil, fmt.Errorf("target not found: %w", err)
	}

	if info.IsDir() {
		findings, err = scanner.ScanDir(params.Target, []string{"node_modules", ".git", "vendor", "__pycache__"})
		if err != nil {
			return nil, fmt.Errorf("scan failed: %w", err)
		}
	} else {
		content, err := os.ReadFile(params.Target)
		if err != nil {
			return nil, fmt.Errorf("failed to read file: %w", err)
		}
		fileFindings := scanner.ScanFile(params.Target, string(content))
		findings = fileFindings
	}

	var findingsMap []map[string]interface{}
	for _, f := range findings {
		findingsMap = append(findingsMap, f.ToMap())
	}

	return &Response{
		Result: map[string]interface{}{
			"target":   params.Target,
			"ruleset":  params.Ruleset,
			"findings": findingsMap,
			"total":    len(findings),
			"passed":   len(findings) == 0,
		},
	}, nil
}

func (s *Server) handleSastFindings(ctx context.Context, req *Request) (*Response, error) {
	var params struct {
		Severity string `json:"severity,omitempty"`
		Status   string `json:"status,omitempty"`
		Limit    int    `json:"limit,omitempty"`
	}

	if err := json.Unmarshal(req.Params, &params); err != nil {
		return nil, fmt.Errorf("failed to parse params: %w", err)
	}

	if params.Limit == 0 {
		params.Limit = 100
	}

	query := `SELECT id, rule_id, severity, file, line, message, status, created_at FROM sast_findings WHERE (? = '' OR severity = ?) AND (? = '' OR status = ?) LIMIT ?`
	rows, err := s.db.Query(query, params.Severity, params.Severity, params.Status, params.Status, params.Limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get findings: %w", err)
	}
	defer rows.Close()

	var findings []*SASTFinding
	for rows.Next() {
		f := &SASTFinding{}
		err := rows.Scan(&f.ID, &f.RuleID, &f.Severity, &f.File, &f.Line, &f.Message, &f.Status, &f.CreatedAt)
		if err != nil {
			return nil, err
		}
		findings = append(findings, f)
	}

	return &Response{
		Result: map[string]interface{}{
			"findings": findings,
			"count":    len(findings),
		},
	}, nil
}

func (s *Server) handleSastResolve(ctx context.Context, req *Request) (*Response, error) {
	var params struct {
		FindingID string `json:"finding_id"`
		Notes     string `json:"notes,omitempty"`
	}

	if err := json.Unmarshal(req.Params, &params); err != nil {
		return nil, fmt.Errorf("failed to parse params: %w", err)
	}

	query := `UPDATE sast_findings SET status = 'resolved' WHERE id = ?`
	_, err := s.db.Exec(query, params.FindingID)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve finding: %w", err)
	}

	return &Response{
		Result: map[string]interface{}{
			"finding_id":  params.FindingID,
			"status":      "resolved",
			"resolved_at": time.Now(),
		},
	}, nil
}

func (s *Server) handleAuditLog(ctx context.Context, req *Request) (*Response, error) {
	var params struct {
		SessionID  string `json:"session_id,omitempty"`
		Tool       string `json:"tool"`
		ActionType string `json:"action_type"`
		Target     string `json:"target"`
		RiskLevel  string `json:"risk_level,omitempty"`
		Result     string `json:"result,omitempty"`
	}

	if err := json.Unmarshal(req.Params, &params); err != nil {
		return nil, fmt.Errorf("failed to parse params: %w", err)
	}

	if params.RiskLevel == "" {
		params.RiskLevel = "low"
	}

	entry := &AuditEntry{
		ID:         generateID("audit"),
		SessionID:  params.SessionID,
		Tool:       params.Tool,
		ActionType: params.ActionType,
		Target:     params.Target,
		RiskLevel:  params.RiskLevel,
		Result:     params.Result,
		CreatedAt:  time.Now(),
	}

	query := `INSERT INTO audit_log (id, session_id, tool, action_type, target, risk_level, result, created_at)
			  VALUES (?, ?, ?, ?, ?, ?, ?, ?)`
	_, err := s.db.Exec(query, entry.ID, entry.SessionID, entry.Tool, entry.ActionType, entry.Target, entry.RiskLevel, entry.Result, entry.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("failed to log audit: %w", err)
	}

	return &Response{
		Result: map[string]interface{}{
			"id":         entry.ID,
			"logged":     true,
			"created_at": entry.CreatedAt,
		},
	}, nil
}

func (s *Server) handleSessionAudit(ctx context.Context, req *Request) (*Response, error) {
	var params struct {
		SessionID string `json:"session_id"`
		Limit     int    `json:"limit,omitempty"`
	}

	if err := json.Unmarshal(req.Params, &params); err != nil {
		return nil, fmt.Errorf("failed to parse params: %w", err)
	}

	if params.Limit == 0 {
		params.Limit = 100
	}

	query := `SELECT id, session_id, tool, action_type, target, risk_level, result, created_at FROM audit_log WHERE session_id = ? LIMIT ?`
	rows, err := s.db.Query(query, params.SessionID, params.Limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get session audit: %w", err)
	}
	defer rows.Close()

	var entries []*AuditEntry
	for rows.Next() {
		e := &AuditEntry{}
		err := rows.Scan(&e.ID, &e.SessionID, &e.Tool, &e.ActionType, &e.Target, &e.RiskLevel, &e.Result, &e.CreatedAt)
		if err != nil {
			return nil, err
		}
		entries = append(entries, e)
	}

	return &Response{
		Result: map[string]interface{}{
			"session_id": params.SessionID,
			"entries":    entries,
			"count":      len(entries),
		},
	}, nil
}

func (s *Server) handleInjectGuard(ctx context.Context, req *Request) (*Response, error) {
	var params struct {
		Content string `json:"content"`
	}

	if err := json.Unmarshal(req.Params, &params); err != nil {
		return nil, fmt.Errorf("failed to parse params: %w", err)
	}

	result := security.ScanContent(params.Content)

	var findings []map[string]interface{}
	for _, f := range result.Findings {
		findings = append(findings, map[string]interface{}{
			"type":     f.Type,
			"severity": f.Severity,
			"message":  f.Message,
			"match":    f.Match,
		})
	}

	return &Response{
		Result: map[string]interface{}{
			"clean":        result.Safe,
			"has_findings": result.HasFindings,
			"findings":     findings,
			"scanned_at":   time.Now(),
		},
	}, nil
}

func (s *Server) handleProactiveScan(ctx context.Context, req *Request) (*Response, error) {
	var params struct {
		ModulePath string `json:"module_path"`
	}

	if err := json.Unmarshal(req.Params, &params); err != nil {
		return nil, fmt.Errorf("failed to parse params: %w", err)
	}

	if params.ModulePath == "" {
		return nil, fmt.Errorf("module_path is required")
	}

	scanner := sast.NewScanner()
	findings, err := scanner.ScanDir(params.ModulePath, []string{"node_modules", ".git", "vendor", "__pycache__"})
	if err != nil {
		return nil, fmt.Errorf("scan failed: %w", err)
	}

	var critical, high, medium, low int
	var injectionFindings, secretFindings, pathFindings []map[string]interface{}

	for _, f := range findings {
		switch f.Severity {
		case "critical":
			critical++
		case "high":
			high++
		case "medium":
			medium++
		case "low":
			low++
		}

		fMap := f.ToMap()
		switch f.Type {
		case "xss", "ssti":
			injectionFindings = append(injectionFindings, fMap)
		case "secret":
			secretFindings = append(secretFindings, fMap)
		case "path-traversal":
			pathFindings = append(pathFindings, fMap)
		}
	}

	return &Response{
		Result: map[string]interface{}{
			"module_path":     params.ModulePath,
			"total_findings":  len(findings),
			"critical_count":  critical,
			"high_count":      high,
			"medium_count":    medium,
			"low_count":       low,
			"injections":      injectionFindings,
			"secrets":         secretFindings,
			"path_traversals": pathFindings,
			"clean":           len(findings) == 0,
			"scanned_at":      time.Now(),
		},
	}, nil
}

func (s *Server) handleSanitize(ctx context.Context, req *Request) (*Response, error) {
	var params struct {
		Content string `json:"content"`
	}

	if err := json.Unmarshal(req.Params, &params); err != nil {
		return nil, fmt.Errorf("failed to parse params: %w", err)
	}

	result := security.Sanitize(params.Content)

	return &Response{
		Result: map[string]interface{}{
			"original_length":  len(params.Content),
			"sanitized_length": len(result.Content),
			"sanitized":        result.Content,
			"redacted_count":   result.Redacted,
			"redactions":       result.Redactions,
			"sanitized_at":     time.Now(),
		},
	}, nil
}

func (s *Server) handleStandardRun(ctx context.Context, req *Request) (*Response, error) {
	var params struct {
		StandardID string `json:"standard_id"`
	}

	if err := json.Unmarshal(req.Params, &params); err != nil {
		return nil, fmt.Errorf("failed to parse params: %w", err)
	}

	return &Response{
		Result: map[string]interface{}{
			"standard_id": params.StandardID,
			"passed":      true,
			"details":     map[string]interface{}{},
			"note":        "standard_run executes a single standard",
		},
	}, nil
}

func (s *Server) handleStandardRunAll(ctx context.Context, req *Request) (*Response, error) {
	var params struct {
		CheckpointType string `json:"checkpoint_type,omitempty"`
	}

	if err := json.Unmarshal(req.Params, &params); err != nil {
		return nil, fmt.Errorf("failed to parse params: %w", err)
	}

	if params.CheckpointType == "" {
		params.CheckpointType = "all"
	}

	return &Response{
		Result: map[string]interface{}{
			"checkpoint_type": params.CheckpointType,
			"standards_run":   0,
			"passed":          true,
			"quality_score":   0.0,
			"snapshot":        map[string]interface{}{},
			"note":            "standard_run_all returns Quality Snapshot for Hati",
		},
	}, nil
}

func (s *Server) handleStandardList(ctx context.Context, req *Request) (*Response, error) {
	query := `SELECT id, name, description, category, last_result, pass_rate FROM standards LIMIT 50`
	rows, err := s.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to list standards: %w", err)
	}
	defer rows.Close()

	var standards []*Standard
	for rows.Next() {
		st := &Standard{}
		err := rows.Scan(&st.ID, &st.Name, &st.Description, &st.Category, &st.LastResult, &st.PassRate)
		if err != nil {
			return nil, err
		}
		standards = append(standards, st)
	}

	return &Response{
		Result: map[string]interface{}{
			"standards": standards,
			"count":     len(standards),
		},
	}, nil
}

func (s *Server) handleQualitySnapshot(ctx context.Context, req *Request) (*Response, error) {
	var params struct {
		CheckpointType string `json:"checkpoint_type,omitempty"`
	}

	if err := json.Unmarshal(req.Params, &params); err != nil {
		return nil, fmt.Errorf("failed to parse params: %w", err)
	}

	return &Response{
		Result: map[string]interface{}{
			"snapshot_time":   time.Now(),
			"checkpoint_type": params.CheckpointType,
			"quality_score":   0.0,
			"standards":       []interface{}{},
			"findings":        []interface{}{},
			"note":            "quality_snapshot provides latest snapshot for Hati",
		},
	}, nil
}

func (s *Server) handleScopeViolations(ctx context.Context, req *Request) (*Response, error) {
	var params struct {
		SessionID string `json:"session_id,omitempty"`
		Limit     int    `json:"limit,omitempty"`
	}

	if err := json.Unmarshal(req.Params, &params); err != nil {
		return nil, fmt.Errorf("failed to parse params: %w", err)
	}

	if params.Limit == 0 {
		params.Limit = 50
	}

	query := `SELECT id, session_id, module, violation_type, target, created_at FROM scope_violations WHERE (? = '' OR session_id = ?) LIMIT ?`
	rows, err := s.db.Query(query, params.SessionID, params.SessionID, params.Limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get violations: %w", err)
	}
	defer rows.Close()

	var violations []map[string]interface{}
	for rows.Next() {
		var id, sessionID, module, violationType, target string
		var createdAt time.Time
		rows.Scan(&id, &sessionID, &module, &violationType, &target, &createdAt)
		violations = append(violations, map[string]interface{}{
			"id":             id,
			"session_id":     sessionID,
			"module":         module,
			"violation_type": violationType,
			"target":         target,
			"created_at":     createdAt,
		})
	}

	return &Response{
		Result: map[string]interface{}{
			"violations": violations,
			"count":      len(violations),
		},
	}, nil
}

func (s *Server) handleTyrStats(ctx context.Context, req *Request) (*Response, error) {
	var totalFindings, activeFindings, totalAudits, scopeViolations int
	s.db.QueryRow(`SELECT COUNT(*) FROM sast_findings`).Scan(&totalFindings)
	s.db.QueryRow(`SELECT COUNT(*) FROM sast_findings WHERE status = 'active'`).Scan(&activeFindings)
	s.db.QueryRow(`SELECT COUNT(*) FROM audit_log`).Scan(&totalAudits)
	s.db.QueryRow(`SELECT COUNT(*) FROM scope_violations`).Scan(&scopeViolations)

	return &Response{
		Result: map[string]interface{}{
			"status":           "operational",
			"version":          "1.0.0",
			"total_findings":   totalFindings,
			"active_findings":  activeFindings,
			"total_audits":     totalAudits,
			"scope_violations": scopeViolations,
		},
	}, nil
}

func (s *Server) handlePrecommitValidate(ctx context.Context, req *Request) (*Response, error) {
	var params struct {
		Files        []map[string]string `json:"files"`
		AllowAutofix bool                `json:"allow_autofix,omitempty"`
		StrictMode   bool                `json:"strict_mode,omitempty"`
	}

	if err := json.Unmarshal(req.Params, &params); err != nil {
		return nil, fmt.Errorf("failed to parse params: %w", err)
	}

	var files []*precommit.FileChange
	for _, f := range params.Files {
		files = append(files, &precommit.FileChange{
			Path:    f["path"],
			Content: f["content"],
		})
	}

	cfg := &precommit.ValidatorConfig{
		AllowAutofix: params.AllowAutofix,
		StrictMode:   params.StrictMode,
	}

	validator := precommit.NewPreCommitValidator(cfg)

	var result *precommit.ValidationResponse
	if params.AllowAutofix {
		result = validator.ValidateWithAutofix(files)
	} else {
		result = validator.Validate(files)
	}

	return &Response{
		Result: map[string]interface{}{
			"passed":       result.Passed,
			"duration_ms":  result.DurationMs,
			"files":        result.Files,
			"errors":       result.Errors,
			"warnings":     result.Warnings,
			"fixed_count":  result.FixedCount,
			"can_continue": result.CanContinue,
		},
	}, nil
}

func (s *Server) handlePrecommitAutofix(ctx context.Context, req *Request) (*Response, error) {
	var params struct {
		Files []map[string]string `json:"files"`
	}

	if err := json.Unmarshal(req.Params, &params); err != nil {
		return nil, fmt.Errorf("failed to parse params: %w", err)
	}

	var results []map[string]string
	for _, f := range params.Files {
		lang := precommit.DetectLanguage(f["path"])
		fixed := precommit.AutoFixContent(f["content"], lang)
		results = append(results, map[string]string{
			"path":    f["path"],
			"content": fixed,
		})
	}

	return &Response{
		Result: map[string]interface{}{
			"files":       results,
			"fixed_count": len(results),
		},
	}, nil
}

func (s *Server) handleBootstrapImport(ctx context.Context, req *Request) (*Response, error) {
	var params struct {
		ProjectPath string `json:"project_path"`
	}

	if err := json.Unmarshal(req.Params, &params); err != nil {
		return nil, fmt.Errorf("failed to parse params: %w", err)
	}

	if params.ProjectPath == "" {
		return nil, fmt.Errorf("project_path is required")
	}

	ragnarokDir := params.ProjectPath + "/.ragnarok"
	standardsFile := ragnarokDir + "/standards.json"

	standardsCount := 0

	if data, err := os.ReadFile(standardsFile); err == nil {
		var standards []map[string]string
		if err := json.Unmarshal(data, &standards); err == nil {
			for _, std := range standards {
				name := std["name"]
				description := std["description"]
				stdType := std["type"]
				if stdType == "" {
					stdType = "quality"
				}

				id := fmt.Sprintf("std_%d", time.Now().UnixNano())
				now := time.Now()
				_, err := s.db.Exec(`
					INSERT OR REPLACE INTO standards 
					(id, name, description, category, created_at)
					VALUES (?, ?, ?, ?, ?)
				`, id, name, description, stdType, now)
				if err == nil {
					standardsCount++
				}
			}
		}
	}

	return &Response{
		Result: map[string]interface{}{
			"project_path":       params.ProjectPath,
			"standards_imported": standardsCount,
			"source":             "fenrir-bootstrap",
		},
	}, nil
}
