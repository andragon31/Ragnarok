package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/andragon31/Ragnarok/internal/fenrir/scanner"
)

func (s *Server) handleProjectScan(ctx context.Context, req *Request) (*Response, error) {
	var params struct {
		ProjectPath string `json:"project_path"`
		Layer       string `json:"layer,omitempty"`
	}

	if err := json.Unmarshal(req.Params, &params); err != nil {
		return nil, fmt.Errorf("failed to parse params: %w", err)
	}

	if params.ProjectPath == "" {
		params.ProjectPath = "."
	}

	absPath, err := filepath.Abs(params.ProjectPath)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve path: %w", err)
	}

	analyzer := scanner.NewProjectAnalyzer(absPath)
	analysis, err := analyzer.Analyze()
	if err != nil {
		return nil, fmt.Errorf("failed to analyze project: %w", err)
	}

	if params.Layer != "" && params.Layer != "all" {
		return filterAnalysisByLayer(analysis, params.Layer)
	}

	analysis.Name = filepath.Base(absPath)

	return &Response{
		Result: map[string]interface{}{
			"path":         analysis.Path,
			"name":         analysis.Name,
			"stack":        analysis.Stack,
			"architecture": analysis.Architecture,
			"modules":      analysis.Modules,
			"patterns":     analysis.Patterns,
			"config_files": analysis.ConfigFiles,
			"root_files":   analysis.RootFiles,
		},
	}, nil
}

func filterAnalysisByLayer(analysis *scanner.ProjectAnalysis, layer string) (*Response, error) {
	switch layer {
	case "stack":
		return &Response{
			Result: map[string]interface{}{
				"layer": "stack",
				"stack": analysis.Stack,
			},
		}, nil
	case "arch", "architecture":
		return &Response{
			Result: map[string]interface{}{
				"layer":        layer,
				"architecture": analysis.Architecture,
			},
		}, nil
	case "modules":
		return &Response{
			Result: map[string]interface{}{
				"layer":   layer,
				"modules": analysis.Modules,
			},
		}, nil
	case "patterns":
		return &Response{
			Result: map[string]interface{}{
				"layer":    layer,
				"patterns": analysis.Patterns,
			},
		}, nil
	case "config":
		return &Response{
			Result: map[string]interface{}{
				"layer":        layer,
				"config_files": analysis.ConfigFiles,
			},
		}, nil
	default:
		return &Response{
			Result: map[string]interface{}{
				"layer":        layer,
				"note":         "unknown layer, showing full analysis",
				"stack":        analysis.Stack,
				"architecture": analysis.Architecture,
				"modules":      analysis.Modules,
				"patterns":     analysis.Patterns,
			},
		}, nil
	}
}

func (s *Server) handleProjectBootstrap(ctx context.Context, req *Request) (*Response, error) {
	var params struct {
		ProjectPath string `json:"project_path"`
	}

	if err := json.Unmarshal(req.Params, &params); err != nil {
		return nil, fmt.Errorf("failed to parse params: %w", err)
	}

	if params.ProjectPath == "" {
		params.ProjectPath = "."
	}

	absPath, err := filepath.Abs(params.ProjectPath)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve path: %w", err)
	}

	analyzer := scanner.NewProjectAnalyzer(absPath)
	analysis, err := analyzer.Analyze()
	if err != nil {
		return nil, fmt.Errorf("failed to analyze project: %w", err)
	}

	analysis.Name = filepath.Base(absPath)

	skillsConfig := scanner.GenerateSkillsConfig(analysis)
	rulesConfig := scanner.GenerateRulesConfig(analysis)
	standardsConfig := scanner.GenerateStandardsConfig(analysis)

	bootstrapDir := filepath.Join(absPath, ".ragnarok")
	os.MkdirAll(bootstrapDir, 0755)

	skillsFile := filepath.Join(bootstrapDir, "skills.json")
	skillsJSON, _ := json.MarshalIndent(skillsConfig, "", "  ")
	os.WriteFile(skillsFile, skillsJSON, 0644)

	rulesFile := filepath.Join(bootstrapDir, "rules.json")
	rulesJSON, _ := json.MarshalIndent(rulesConfig, "", "  ")
	os.WriteFile(rulesFile, rulesJSON, 0644)

	standardsFile := filepath.Join(bootstrapDir, "standards.json")
	standardsJSON, _ := json.MarshalIndent(standardsConfig, "", "  ")
	os.WriteFile(standardsFile, standardsJSON, 0644)

	return &Response{
		Result: map[string]interface{}{
			"project_path":    absPath,
			"project_name":    analysis.Name,
			"stack":           analysis.Stack,
			"architecture":    analysis.Architecture,
			"skills_count":    len(skillsConfig["suggested_skills"].([]map[string]string)),
			"rules_count":     len(rulesConfig),
			"standards_count": len(standardsConfig),
			"bootstrap_dir":   bootstrapDir,
			"files_created": []string{
				skillsFile,
				rulesFile,
				standardsFile,
			},
		},
	}, nil
}

func (s *Server) handleSkillGenerate(ctx context.Context, req *Request) (*Response, error) {
	var params struct {
		ProjectPath string `json:"project_path"`
	}

	if err := json.Unmarshal(req.Params, &params); err != nil {
		return nil, fmt.Errorf("failed to parse params: %w", err)
	}

	if params.ProjectPath == "" {
		params.ProjectPath = "."
	}

	absPath, err := filepath.Abs(params.ProjectPath)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve path: %w", err)
	}

	analyzer := scanner.NewProjectAnalyzer(absPath)
	analysis, err := analyzer.Analyze()
	if err != nil {
		return nil, fmt.Errorf("failed to analyze project: %w", err)
	}

	skillsConfig := scanner.GenerateSkillsConfig(analysis)

	return &Response{
		Result: map[string]interface{}{
			"project_path": absPath,
			"stack":        analysis.Stack,
			"skills":       skillsConfig["suggested_skills"],
		},
	}, nil
}

func (s *Server) handleRulesGenerate(ctx context.Context, req *Request) (*Response, error) {
	var params struct {
		ProjectPath string `json:"project_path"`
	}

	if err := json.Unmarshal(req.Params, &params); err != nil {
		return nil, fmt.Errorf("failed to parse params: %w", err)
	}

	if params.ProjectPath == "" {
		params.ProjectPath = "."
	}

	absPath, err := filepath.Abs(params.ProjectPath)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve path: %w", err)
	}

	analyzer := scanner.NewProjectAnalyzer(absPath)
	analysis, err := analyzer.Analyze()
	if err != nil {
		return nil, fmt.Errorf("failed to analyze project: %w", err)
	}

	rulesConfig := scanner.GenerateRulesConfig(analysis)

	return &Response{
		Result: map[string]interface{}{
			"project_path": absPath,
			"stack":        analysis.Stack,
			"rules":        rulesConfig,
			"count":        len(rulesConfig),
		},
	}, nil
}

func (s *Server) handleStandardsGenerate(ctx context.Context, req *Request) (*Response, error) {
	var params struct {
		ProjectPath string `json:"project_path"`
	}

	if err := json.Unmarshal(req.Params, &params); err != nil {
		return nil, fmt.Errorf("failed to parse params: %w", err)
	}

	if params.ProjectPath == "" {
		params.ProjectPath = "."
	}

	absPath, err := filepath.Abs(params.ProjectPath)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve path: %w", err)
	}

	analyzer := scanner.NewProjectAnalyzer(absPath)
	analysis, err := analyzer.Analyze()
	if err != nil {
		return nil, fmt.Errorf("failed to analyze project: %w", err)
	}

	standardsConfig := scanner.GenerateStandardsConfig(analysis)

	return &Response{
		Result: map[string]interface{}{
			"project_path": absPath,
			"stack":        analysis.Stack,
			"standards":    standardsConfig,
			"count":        len(standardsConfig),
		},
	}, nil
}
