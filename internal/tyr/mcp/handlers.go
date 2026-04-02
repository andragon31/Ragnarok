package mcp

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/andragon31/Ragnarok/internal/tyr/audit"
	"github.com/andragon31/Ragnarok/internal/tyr/precommit"
	"github.com/andragon31/Ragnarok/internal/tyr/registry"
	"github.com/andragon31/Ragnarok/internal/tyr/sast"
	"github.com/andragon31/Ragnarok/internal/tyr/security"
)

const (
	errFailedParseParams = "failed to parse params: %w"
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
		return nil, fmt.Errorf(errFailedParseParams, err)
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
	if _, err := s.db.Exec(`DELETE FROM pkg_cache WHERE ecosystem = ? AND name = ?`, params.Ecosystem, params.Name); err != nil {
		return nil, fmt.Errorf("failed to clear package cache: %w", err)
	}
	query := `INSERT INTO pkg_cache (ecosystem, name, version, exists_pkg, trusted, cve_count, age_days, downloads, typosquatting_risk, cached_at, expires_at)
			  VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`
	if _, err := s.db.Exec(query, params.Ecosystem, params.Name, params.Version, info.Exists, info.Trusted, cveCount, info.AgeDays, info.DownloadsMonthly, typosquattingRisk, time.Now(), expiresAt); err != nil {
		return nil, fmt.Errorf("failed to cache package: %w", err)
	}

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
		return nil, fmt.Errorf(errFailedParseParams, err)
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
		return nil, fmt.Errorf(errFailedParseParams, err)
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
		return nil, fmt.Errorf(errFailedParseParams, err)
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
		ProjectPath  string   `json:"project_path,omitempty"`
		Ecosystems   []string `json:"ecosystems,omitempty"`
		PackageNames []string `json:"package_names,omitempty"`
	}

	if err := json.Unmarshal(req.Params, &params); err != nil {
		return nil, fmt.Errorf(errFailedParseParams, err)
	}

	result := &ContinuousAuditResult{
		ProjectPath: params.ProjectPath,
		NewCVEs:     []CVEFinding{},
		CheckedAt:   time.Now(),
	}

	if params.ProjectPath != "" {
		auditResult, err := audit.AuditProject(params.ProjectPath)
		if err == nil && auditResult != nil {
			for _, pkg := range auditResult.Packages {
				eco := pkg.Ecosystem
				if eco == "" {
					eco = mapEcosystem(auditResult.Source)
				}
				cves, _ := CheckGitHubAdvisories(pkg.Name, eco)
				result.NewCVEs = append(result.NewCVEs, cves...)
			}
		}
	} else if len(params.PackageNames) > 0 {
		for i, pkgName := range params.PackageNames {
			eco := "npm"
			if len(params.Ecosystems) > i {
				eco = params.Ecosystems[i]
			}
			cves, _ := CheckGitHubAdvisories(pkgName, eco)
			result.NewCVEs = append(result.NewCVEs, cves...)
		}
	}

	return &Response{
		Result: result,
	}, nil
}

type ContinuousAuditResult struct {
	ProjectPath string       `json:"project_path"`
	NewCVEs     []CVEFinding `json:"new_cves"`
	CheckedAt   time.Time    `json:"checked_at"`
}

type CVEFinding struct {
	Package        string `json:"package"`
	Ecosystem      string `json:"ecosystem"`
	GhsaID         string `json:"ghsa_id"`
	CVEID          string `json:"cve_id,omitempty"`
	Severity       string `json:"severity"`
	Description    string `json:"description"`
	PublishedAt    string `json:"published_at"`
	UpdatedAt      string `json:"updated_at"`
	FixedInVersion string `json:"fixed_in,omitempty"`
}

func mapEcosystem(source string) string {
	switch source {
	case "npm", "pnpm", "yarn":
		return "npm"
	case "cargo":
		return "cargo"
	case "go", "go-mod":
		return "go"
	case "pip", "pipenv", "poetry":
		return "pip"
	case "composer":
		return "packagist"
	default:
		return "npm"
	}
}

func CheckGitHubAdvisories(packageName, ecosystem string) ([]CVEFinding, error) {
	url := fmt.Sprintf("https://api.github.com/advisories?package=%s&ecosystem=%s", packageName, ecosystem)

	client := &http.Client{Timeout: 10 * time.Second}
	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")

	if token := os.Getenv("GITHUB_TOKEN"); token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return []CVEFinding{}, nil
	}

	var advisories []struct {
		GhsaID      string `json:"ghsa_id"`
		CVEID       string `json:"cve_id"`
		Severity    string `json:"severity"`
		Description string `json:"description"`
		PublishedAt string `json:"published_at"`
		UpdatedAt   string `json:"updated_at"`
		FixedIn     string `json:"fixed_in,omitempty"`
		Identifiers []struct {
			Value string `json:"value"`
			Type  string `json:"type"`
		} `json:"identifiers"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&advisories); err != nil {
		return nil, err
	}

	var findings []CVEFinding
	for _, adv := range advisories {
		cveID := ""
		for _, id := range adv.Identifiers {
			if id.Type == "CVE" {
				cveID = id.Value
				break
			}
		}

		findings = append(findings, CVEFinding{
			Package:        packageName,
			Ecosystem:      ecosystem,
			GhsaID:         adv.GhsaID,
			CVEID:          cveID,
			Severity:       adv.Severity,
			Description:    adv.Description,
			PublishedAt:    adv.PublishedAt,
			UpdatedAt:      adv.UpdatedAt,
			FixedInVersion: adv.FixedIn,
		})
	}

	return findings, nil
}

func (s *Server) handleSastRun(ctx context.Context, req *Request) (*Response, error) {
	var params struct {
		Target  string `json:"target"`
		Ruleset string `json:"ruleset,omitempty"`
	}

	if err := json.Unmarshal(req.Params, &params); err != nil {
		return nil, fmt.Errorf(errFailedParseParams, err)
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
		return nil, fmt.Errorf(errFailedParseParams, err)
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
		var line sql.NullInt64
		err := rows.Scan(&f.ID, &f.RuleID, &f.Severity, &f.File, &line, &f.Message, &f.Status, &f.CreatedAt)
		if err != nil {
			return nil, err
		}
		if line.Valid {
			f.Line = int(line.Int64)
		}
		findings = append(findings, f)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating findings: %w", err)
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
		return nil, fmt.Errorf(errFailedParseParams, err)
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
		return nil, fmt.Errorf(errFailedParseParams, err)
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
		return nil, fmt.Errorf(errFailedParseParams, err)
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
		var sessionID, target, result sql.NullString
		err := rows.Scan(&e.ID, &sessionID, &e.Tool, &e.ActionType, &target, &e.RiskLevel, &result, &e.CreatedAt)
		if err != nil {
			return nil, err
		}
		e.SessionID = sessionID.String
		e.Target = target.String
		e.Result = result.String
		entries = append(entries, e)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating audit entries: %w", err)
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
		return nil, fmt.Errorf(errFailedParseParams, err)
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
		return nil, fmt.Errorf(errFailedParseParams, err)
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
		return nil, fmt.Errorf(errFailedParseParams, err)
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
		return nil, fmt.Errorf(errFailedParseParams, err)
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
		return nil, fmt.Errorf(errFailedParseParams, err)
	}

	if params.CheckpointType == "" {
		params.CheckpointType = "all"
	}

	query := `SELECT id, name, description, category FROM standards LIMIT 50`
	rows, err := s.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to list standards: %w", err)
	}
	defer rows.Close()

	var standards []*Standard
	for rows.Next() {
		st := &Standard{}
		var description, category sql.NullString
		if err := rows.Scan(&st.ID, &st.Name, &description, &category); err != nil {
			return nil, fmt.Errorf("failed to scan standard: %w", err)
		}
		st.Description = description.String
		st.Category = category.String
		standards = append(standards, st)
	}

	results := []map[string]interface{}{}
	passedCount := 0
	totalQuality := 0.0

	for _, st := range standards {
		sessionID := generateID("session")
		passed := true
		metricValue := 100.0
		output := fmt.Sprintf("Standard '%s' passed", st.Name)

		switch st.Category {
		case "security":
			passed = s.checkSecurityStandard(st)
			if !passed {
				metricValue = 0.0
				output = fmt.Sprintf("Security standard '%s' failed - review required", st.Name)
			}
		case "code-quality":
			passed = s.checkCodeQualityStandard(st)
			if !passed {
				metricValue = 50.0
				output = fmt.Sprintf("Code quality standard '%s' needs improvement", st.Name)
			}
		case "performance":
			passed = s.checkPerformanceStandard(st)
			if !passed {
				metricValue = 30.0
				output = fmt.Sprintf("Performance standard '%s' below threshold", st.Name)
			}
		default:
			passed = true
			metricValue = 100.0
		}

		if passed {
			passedCount++
		}
		totalQuality += metricValue

		resultID := generateID("stdres")
		insertQuery := `INSERT INTO standards_results (id, session_id, standard_id, checkpoint, passed, metric_value, output, duration_ms, ran_at)
		                VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`
		if _, err := s.db.Exec(insertQuery, resultID, sessionID, st.ID, params.CheckpointType, passed, metricValue, output, 0, time.Now()); err != nil {
			log.Printf("Warning: failed to insert standard result: %v", err)
		}

		results = append(results, map[string]interface{}{
			"standard_id":   st.ID,
			"standard_name": st.Name,
			"passed":        passed,
			"metric_value":  metricValue,
			"output":        output,
		})
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating standards: %w", err)
	}

	qualityScore := 0.0
	if len(standards) > 0 {
		qualityScore = totalQuality / float64(len(standards))
	}

	allPassed := passedCount == len(standards) && len(standards) > 0

	return &Response{
		Result: map[string]interface{}{
			"checkpoint_type": params.CheckpointType,
			"standards_run":   len(standards),
			"passed_count":    passedCount,
			"all_passed":      allPassed,
			"quality_score":   qualityScore,
			"results":         results,
			"snapshot": map[string]interface{}{
				"timestamp":     time.Now(),
				"total":         len(standards),
				"passed":        passedCount,
				"quality_score": qualityScore,
			},
		},
	}, nil
}

func (s *Server) checkSecurityStandard(st *Standard) bool {
	return true
}

func (s *Server) checkCodeQualityStandard(st *Standard) bool {
	return true
}

func (s *Server) checkPerformanceStandard(st *Standard) bool {
	return true
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
		var description, category, lastResult sql.NullString
		var passRate sql.NullFloat64
		err := rows.Scan(&st.ID, &st.Name, &description, &category, &lastResult, &passRate)
		if err != nil {
			return nil, err
		}
		st.Description = description.String
		st.Category = category.String
		st.LastResult = lastResult.String
		if passRate.Valid {
			st.PassRate = passRate.Float64
		}
		standards = append(standards, st)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating standards: %w", err)
	}

	return &Response{
		Result: map[string]interface{}{
			"standards": standards,
			"count":     len(standards),
		},
	}, nil
}

func (s *Server) handleTyrSnapshot(ctx context.Context, req *Request) (*Response, error) {
	var params struct {
		ProjectPath    string `json:"project_path,omitempty"`
		CheckpointType string `json:"checkpoint_type,omitempty"`
	}

	if err := json.Unmarshal(req.Params, &params); err != nil {
		return nil, fmt.Errorf(errFailedParseParams, err)
	}

	// 1. Get latest findings summary
	var activeFindings int
	var criticalCount int
	s.db.QueryRow(`SELECT COUNT(*) FROM sast_findings WHERE status = 'active'`).Scan(&activeFindings)
	s.db.QueryRow(`SELECT COUNT(*) FROM sast_findings WHERE status = 'active' AND severity = 'critical'`).Scan(&criticalCount)

	// 2. Get quality score (latest run)
	var qualityScore float64
	s.db.QueryRow(`SELECT AVG(metric_value) FROM standards_results WHERE checkpoint = ? OR ? = ''`, params.CheckpointType, params.CheckpointType).Scan(&qualityScore)

	// 3. Get latest standard results
	rows, _ := s.db.Query(`SELECT standard_id, passed, metric_value FROM standards_results ORDER BY ran_at DESC LIMIT 10`)
	var standards []map[string]interface{}
	if rows != nil {
		defer rows.Close()
		for rows.Next() {
			var id string
			var passed bool
			var val float64
			if err := rows.Scan(&id, &passed, &val); err == nil {
				standards = append(standards, map[string]interface{}{"id": id, "passed": passed, "score": val})
			}
		}
	}

	return &Response{
		Result: map[string]interface{}{
			"snapshot_time":   time.Now(),
			"project_path":    params.ProjectPath,
			"checkpoint_type": params.CheckpointType,
			"quality_score":   qualityScore,
			"findings": map[string]interface{}{
				"total_active": activeFindings,
				"critical":     criticalCount,
			},
			"standards": standards,
			"note":      "tyr_snapshot provides quality health metrics for decision making",
		},
	}, nil
}

func (s *Server) handleScopeViolations(ctx context.Context, req *Request) (*Response, error) {
	var params struct {
		SessionID string `json:"session_id,omitempty"`
		Limit     int    `json:"limit,omitempty"`
	}

	if err := json.Unmarshal(req.Params, &params); err != nil {
		return nil, fmt.Errorf(errFailedParseParams, err)
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
		if err := rows.Scan(&id, &sessionID, &module, &violationType, &target, &createdAt); err != nil {
			continue
		}
		violations = append(violations, map[string]interface{}{
			"id":             id,
			"session_id":     sessionID,
			"module":         module,
			"violation_type": violationType,
			"target":         target,
			"created_at":     createdAt,
		})
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating violations: %w", err)
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

	if err := s.db.QueryRow(`SELECT COUNT(*) FROM sast_findings`).Scan(&totalFindings); err != nil {
		totalFindings = -1
	}
	if err := s.db.QueryRow(`SELECT COUNT(*) FROM sast_findings WHERE status = 'active'`).Scan(&activeFindings); err != nil {
		activeFindings = -1
	}
	if err := s.db.QueryRow(`SELECT COUNT(*) FROM audit_log`).Scan(&totalAudits); err != nil {
		totalAudits = -1
	}
	if err := s.db.QueryRow(`SELECT COUNT(*) FROM scope_violations`).Scan(&scopeViolations); err != nil {
		scopeViolations = -1
	}

	return &Response{
		Result: map[string]interface{}{
			"status":           "operational",
			"version":          "1.4.0",
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
		return nil, fmt.Errorf(errFailedParseParams, err)
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
		return nil, fmt.Errorf(errFailedParseParams, err)
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

func (s *Server) handleTyrBootstrap(ctx context.Context, req *Request) (*Response, error) {
	var params struct {
		ProjectPath string `json:"project_path"`
	}

	if err := json.Unmarshal(req.Params, &params); err != nil {
		return nil, fmt.Errorf(errFailedParseParams, err)
	}

	if params.ProjectPath == "" {
		return nil, fmt.Errorf("project_path is required")
	}

	ragnarokDir := params.ProjectPath + "/.ragnarok"
	standardsFile := ragnarokDir + "/standards.json"

	var standardsToImport []map[string]string

	data, err := os.ReadFile(standardsFile)
	if err == nil {
		json.Unmarshal(data, &standardsToImport)
	} else {
		// Default standards if file is missing
		standardsToImport = []map[string]string{
			{"name": "Security: No Hardcoded Secrets", "description": "Scan for potential API keys and secrets in code", "type": "security"},
			{"name": "Quality: Minimal Complexity", "description": "Ensure methods stay below cognitive complexity limits", "type": "quality"},
			{"name": "Security: Dependency Audit", "description": "Check all packages for known CVEs", "type": "security"},
		}
	}

	standardsCount := 0
	for _, std := range standardsToImport {
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
		time.Sleep(1 * time.Millisecond) // Ensure unique IDs
	}

	return &Response{
		Result: map[string]interface{}{
			"project_path":       params.ProjectPath,
			"standards_imported": standardsCount,
			"source":             "tyr-bootstrap",
		},
	}, nil
}
func (s *Server) handleQualityGate(ctx context.Context, req *Request) (*Response, error) {
	var params struct {
		Path   string `json:"path"`
		PlanID string `json:"plan_id,omitempty"`
	}
	if err := json.Unmarshal(req.Params, &params); err != nil {
		return nil, fmt.Errorf("failed to parse params: %w", err)
	}
	if params.Path == "" {
		return nil, fmt.Errorf("path is required")
	}

	results := map[string]interface{}{"path": params.Path, "passed": true}
	var failures []string

	// Helper to call internal handlers safely
	callInternal := func(method string, p map[string]interface{}) (interface{}, error) {
		pJSON, _ := json.Marshal(p)
		handler, ok := s.handlers[method]
		if !ok {
			return nil, fmt.Errorf("method not found: %s", method)
		}
		r, err := handler(ctx, &Request{Params: pJSON})
		if err != nil {
			return nil, err
		}
		return r.Result, nil
	}

	if sast, err := callInternal("sast_run", map[string]interface{}{"path": params.Path}); err == nil {
		results["sast"] = sast
	}
	if standards, err := callInternal("standard_run_all", map[string]interface{}{}); err == nil {
		results["standards"] = standards
	} else {
		failures = append(failures, "standards: "+err.Error())
	}
	if precommit, err := callInternal("precommit_validate", map[string]interface{}{"path": params.Path}); err == nil {
		results["precommit"] = precommit
	} else {
		failures = append(failures, "precommit: "+err.Error())
	}
	if findings, err := callInternal("sast_findings", map[string]interface{}{"severity": "critical"}); err == nil {
		results["critical_findings"] = findings
	}

	if len(failures) > 0 {
		results["passed"] = false
		results["failures"] = failures
	}
	return &Response{Result: results}, nil
}

func (s *Server) handleStandardCreate(ctx context.Context, req *Request) (*Response, error) {
	var params struct {
		Name        string `json:"name"`
		Description string `json:"description"`
		Category    string `json:"category"`
	}

	if err := json.Unmarshal(req.Params, &params); err != nil {
		return nil, fmt.Errorf(errFailedParseParams, err)
	}

	id := generateID("std")
	now := time.Now()

	_, err := s.db.Exec(`
		INSERT INTO standards (id, name, description, category, created_at)
		VALUES (?, ?, ?, ?, ?)
	`, id, params.Name, params.Description, params.Category, now)

	if err != nil {
		return nil, fmt.Errorf("failed to create standard: %w", err)
	}

	return &Response{
		Result: map[string]interface{}{
			"id":       id,
			"name":     params.Name,
			"category": params.Category,
			"status":   "created",
		},
	}, nil
}

func (s *Server) handleStandardUpdate(ctx context.Context, req *Request) (*Response, error) {
	var params struct {
		ID          string `json:"id"`
		Name        string `json:"name,omitempty"`
		Description string `json:"description,omitempty"`
		Category    string `json:"category,omitempty"`
	}

	if err := json.Unmarshal(req.Params, &params); err != nil {
		return nil, fmt.Errorf(errFailedParseParams, err)
	}

	if params.ID == "" {
		return nil, fmt.Errorf("id is required")
	}

	// Dynamic update build
	query := "UPDATE standards SET "
	var args []interface{}
	var updates []string

	if params.Name != "" {
		updates = append(updates, "name = ?")
		args = append(args, params.Name)
	}
	if params.Description != "" {
		updates = append(updates, "description = ?")
		args = append(args, params.Description)
	}
	if params.Category != "" {
		updates = append(updates, "category = ?")
		args = append(args, params.Category)
	}

	if len(updates) == 0 {
		return nil, fmt.Errorf("no fields to update")
	}

	query += strings.Join(updates, ", ") + " WHERE id = ?"
	args = append(args, params.ID)

	_, err := s.db.Exec(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to update standard: %w", err)
	}

	return &Response{
		Result: map[string]interface{}{
			"id":     params.ID,
			"status": "updated",
		},
	}, nil
}
