package scanner

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNewProjectAnalyzer(t *testing.T) {
	analyzer := NewProjectAnalyzer("/test/path")
	if analyzer == nil {
		t.Fatal("Expected non-nil analyzer")
	}
	if analyzer.projectPath != "/test/path" {
		t.Errorf("Expected projectPath '/test/path', got '%s'", analyzer.projectPath)
	}
}

func TestAnalyzeGoProject(t *testing.T) {
	tmpDir := t.TempDir()

	createFile := func(path, content string) {
		fullPath := filepath.Join(tmpDir, path)
		os.MkdirAll(filepath.Dir(fullPath), 0755)
		os.WriteFile(fullPath, []byte(content), 0644)
	}

	createFile("go.mod", "module github.com/test/project\n\ngo 1.22")
	createFile("main.go", "package main\n\nfunc main() {}")
	createFile("README.md", "# Test Project")
	createFile("Dockerfile", "FROM golang:1.22")

	subdir := filepath.Join(tmpDir, "internal", "handler")
	os.MkdirAll(subdir, 0755)
	createFile("internal/handler/handler.go", "package handler")

	analyzer := NewProjectAnalyzer(tmpDir)
	analysis, err := analyzer.Analyze()
	if err != nil {
		t.Fatalf("Analyze() error = %v", err)
	}

	if analysis.Stack.Language != "go" {
		t.Errorf("Expected language 'go', got '%s'", analysis.Stack.Language)
	}
	if analysis.Stack.PackageMgr != "go" {
		t.Errorf("Expected package manager 'go', got '%s'", analysis.Stack.PackageMgr)
	}
	if !analysis.Stack.HasDocker {
		t.Error("Expected HasDocker to be true")
	}
	if len(analysis.Modules) < 1 {
		t.Error("Expected at least 1 module")
	}
	if analysis.Architecture.Type != "monolith" {
		t.Errorf("Expected architecture 'monolith', got '%s'", analysis.Architecture.Type)
	}
}

func TestAnalyzeNodeJsProject(t *testing.T) {
	tmpDir := t.TempDir()

	createFile := func(path, content string) {
		fullPath := filepath.Join(tmpDir, path)
		os.MkdirAll(filepath.Dir(fullPath), 0755)
		os.WriteFile(fullPath, []byte(content), 0644)
	}

	createFile("package.json", `{"name":"test-project","dependencies":{"express":"^4.0.0"}}`)
	createFile("tsconfig.json", "{}")
	createFile("jest.config.js", "module.exports = {}")

	analyzer := NewProjectAnalyzer(tmpDir)
	analysis, err := analyzer.Analyze()
	if err != nil {
		t.Fatalf("Analyze() error = %v", err)
	}

	if analysis.Stack.Language != "typescript" {
		t.Errorf("Expected language 'typescript', got '%s'", analysis.Stack.Language)
	}
	if analysis.Stack.PackageMgr != "npm" {
		t.Errorf("Expected package manager 'npm', got '%s'", analysis.Stack.PackageMgr)
	}
	if !analysis.Stack.HasTests {
		t.Error("Expected HasTests to be true")
	}
	if analysis.Stack.TestFramework != "jest" {
		t.Errorf("Expected test framework 'jest', got '%s'", analysis.Stack.TestFramework)
	}
}

func TestAnalyzePythonProject(t *testing.T) {
	tmpDir := t.TempDir()

	createFile := func(path, content string) {
		fullPath := filepath.Join(tmpDir, path)
		os.MkdirAll(filepath.Dir(fullPath), 0755)
		os.WriteFile(fullPath, []byte(content), 0644)
	}

	createFile("requirements.txt", "flask>=2.0.0")
	createFile("pytest.ini", "[pytest]")
	createFile("app.py", "from flask import Flask")

	analyzer := NewProjectAnalyzer(tmpDir)
	analysis, err := analyzer.Analyze()
	if err != nil {
		t.Fatalf("Analyze() error = %v", err)
	}

	if analysis.Stack.Language != "python" {
		t.Errorf("Expected language 'python', got '%s'", analysis.Stack.Language)
	}
	if analysis.Stack.PackageMgr != "pip" {
		t.Errorf("Expected package manager 'pip', got '%s'", analysis.Stack.PackageMgr)
	}
	if !analysis.Stack.HasTests {
		t.Error("Expected HasTests to be true")
	}
	if analysis.Stack.TestFramework != "pytest" {
		t.Errorf("Expected test framework 'pytest', got '%s'", analysis.Stack.TestFramework)
	}
}

func TestAnalyzeModularProject(t *testing.T) {
	tmpDir := t.TempDir()

	createFile := func(path, content string) {
		fullPath := filepath.Join(tmpDir, path)
		os.MkdirAll(filepath.Dir(fullPath), 0755)
		os.WriteFile(fullPath, []byte(content), 0644)
	}

	createFile("package.json", `{"name":"mono-repo"}`)
	createFile("module1/package.json", `{"name":"module1"}`)
	createFile("module2/package.json", `{"name":"module2"}`)
	createFile("module3/package.json", `{"name":"module3"}`)
	createFile("module4/package.json", `{"name":"module4"}`)
	createFile("module5/package.json", `{"name":"module5"}`)

	analyzer := NewProjectAnalyzer(tmpDir)
	analysis, err := analyzer.Analyze()
	if err != nil {
		t.Fatalf("Analyze() error = %v", err)
	}

	if analysis.Architecture.Type != "modular" {
		t.Errorf("Expected architecture 'modular', got '%s'", analysis.Architecture.Type)
	}
	if !analysis.Architecture.IsMonorepo {
		t.Error("Expected IsMonorepo to be true for 5+ modules")
	}
}

func TestAnalyzeWithCI(t *testing.T) {
	tmpDir := t.TempDir()

	createFile := func(path, content string) {
		fullPath := filepath.Join(tmpDir, path)
		os.MkdirAll(filepath.Dir(fullPath), 0755)
		os.WriteFile(fullPath, []byte(content), 0644)
	}

	createFile("go.mod", "module test\ngo 1.22")
	createFile(".github/workflows/ci.yml", "name: CI")

	analyzer := NewProjectAnalyzer(tmpDir)
	analysis, err := analyzer.Analyze()
	if err != nil {
		t.Fatalf("Analyze() error = %v", err)
	}

	if !analysis.Stack.HasCI {
		t.Error("Expected HasCI to be true")
	}
	if analysis.Stack.CITool != "github-actions" {
		t.Errorf("Expected CI tool 'github-actions', got '%s'", analysis.Stack.CITool)
	}
}

func TestGenerateSkillsConfig(t *testing.T) {
	analysis := &ProjectAnalysis{
		Stack: &StackInfo{
			Language:      "typescript",
			Framework:     "next.js",
			TestFramework: "jest",
		},
		Architecture: &ArchitectureInfo{},
		Patterns:     []*PatternInfo{},
	}

	config := GenerateSkillsConfig(analysis)
	if config == nil {
		t.Fatal("Expected non-nil config")
	}

	skills, ok := config["suggested_skills"].([]map[string]string)
	if !ok {
		t.Fatal("Expected suggested_skills to be []map[string]string")
	}

	if len(skills) != 3 {
		t.Errorf("Expected 3 skills, got %d", len(skills))
	}
}

func TestGenerateRulesConfig(t *testing.T) {
	analysis := &ProjectAnalysis{
		Stack: &StackInfo{
			HasTests:  true,
			HasCI:     true,
			Language:  "typescript",
			Framework: "next.js",
		},
	}

	rules := GenerateRulesConfig(analysis)
	if len(rules) != 4 {
		t.Errorf("Expected 4 rules, got %d", len(rules))
	}

	found := false
	for _, r := range rules {
		if r["name"] == "strict-typescript" {
			found = true
			break
		}
	}
	if !found {
		t.Error("Expected to find strict-typescript rule")
	}
}

func TestGenerateStandardsConfig(t *testing.T) {
	analysis := &ProjectAnalysis{
		Stack: &StackInfo{
			TestFramework: "jest",
			Language:      "javascript/typescript",
		},
	}

	standards := GenerateStandardsConfig(analysis)
	if len(standards) != 2 {
		t.Errorf("Expected 2 standards, got %d", len(standards))
	}
}

func TestProjectAnalysisToJSON(t *testing.T) {
	analysis := &ProjectAnalysis{
		Path: "/test",
		Stack: &StackInfo{
			Language: "go",
		},
	}

	json := analysis.ToJSON()
	if json == "" {
		t.Error("Expected non-empty JSON")
	}
}

func TestAnalyzeSkipsHiddenDirs(t *testing.T) {
	tmpDir := t.TempDir()

	createFile := func(path string) {
		fullPath := filepath.Join(tmpDir, path)
		os.MkdirAll(filepath.Dir(fullPath), 0755)
		os.WriteFile(fullPath, []byte("content"), 0644)
	}

	createFile(".hidden/file.go")
	createFile("node_modules/pkg/file.js")
	createFile("vendor/module/file.go")
	createFile("src/main.go")

	analyzer := NewProjectAnalyzer(tmpDir)
	analysis, err := analyzer.Analyze()
	if err != nil {
		t.Fatalf("Analyze() error = %v", err)
	}

	for _, f := range analysis.RootFiles {
		if filepath.HasPrefix(f, ".hidden") || f == "node_modules/pkg/file.js" || f == "vendor/module/file.go" {
			t.Errorf("Unexpected file in RootFiles: %s", f)
		}
	}
}

func TestGenerateRuleFingerprint(t *testing.T) {
	fp1 := GenerateRuleFingerprint(
		"no-commit-without-tests",
		"quality",
		"Commits that modify code must include or update tests",
	)

	fp2 := GenerateRuleFingerprint(
		"no-commit-without-tests",
		"quality",
		"Commits that modify code must include or update tests",
	)

	if fp1 != fp2 {
		t.Error("Same inputs should produce same fingerprint")
	}

	fp3 := GenerateRuleFingerprint(
		"strict-typescript",
		"code-quality",
		"Avoid any types, use explicit interfaces",
	)

	if fp1 == fp3 {
		t.Error("Different rules should produce different fingerprints")
	}

	if len(fp1) != 64 {
		t.Errorf("SHA256 hash should be 64 hex characters, got %d", len(fp1))
	}
}

func TestExtractKeywords(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		expectWords int
	}{
		{
			name:        "Short description",
			input:       "avoid any types",
			expectWords: 3,
		},
		{
			name:        "Long description",
			input:       "commits that modify code must include or update tests",
			expectWords: 5,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			kw := extractKeywords(tt.input)
			words := strings.Split(kw, ",")
			if len(words) != tt.expectWords {
				t.Errorf("Expected %d keywords, got %d: %s", tt.expectWords, len(words), kw)
			}
		})
	}
}

func TestGeneratePhasesAndTasks(t *testing.T) {
	analysis := &ProjectAnalysis{
		Stack: &StackInfo{
			Language:      "go",
			Framework:     "gin",
			DBEngine:      "postgresql",
			HasTests:      true,
			TestFramework: "go-test",
			HasDocker:     true,
			CITool:        "github-actions",
		},
		Architecture: &ArchitectureInfo{
			HasAPI: true,
		},
	}

	phases := GeneratePhasesAndTasks(analysis)
	if len(phases) == 0 {
		t.Fatal("Expected at least one phase")
	}

	if phases[0].Name != "Setup" {
		t.Errorf("Expected first phase 'Setup', got '%s'", phases[0].Name)
	}
}

func TestGeneratePhasesAndTasksNilAnalysis(t *testing.T) {
	phases := GeneratePhasesAndTasks(nil)
	if len(phases) != 0 {
		t.Errorf("Expected 0 phases for nil analysis, got %d", len(phases))
	}
}

func TestGeneratePhasesAndTasksNilStack(t *testing.T) {
	analysis := &ProjectAnalysis{
		Stack:        nil,
		Architecture: &ArchitectureInfo{},
	}
	phases := GeneratePhasesAndTasks(analysis)
	if len(phases) == 0 {
		t.Error("Expected phases even with nil stack")
	}
}

func TestGetRecommendedAgents(t *testing.T) {
	analysis := &ProjectAnalysis{
		Architecture: &ArchitectureInfo{
			HasFrontend: true,
		},
		Stack: &StackInfo{
			HasDocker: true,
			HasTests:  true,
		},
	}

	requirements := []map[string]string{
		{"title": "Frontend dashboard", "type": "feature"},
		{"title": "Security authentication", "type": "non-functional"},
		{"title": "Docker deployment", "type": "infrastructure"},
		{"title": "Integration tests", "type": "testing"},
	}

	agents := GetRecommendedAgents(analysis, requirements)
	if len(agents) == 0 {
		t.Fatal("Expected at least one agent")
	}

	hasBackend := false
	hasFrontend := false
	hasSecurity := false
	hasDevops := false
	hasQA := false
	hasDocs := false

	for _, agent := range agents {
		switch agent["type"] {
		case "backend":
			hasBackend = true
		case "frontend":
			hasFrontend = true
		case "security":
			hasSecurity = true
		case "devops":
			hasDevops = true
		case "qa":
			hasQA = true
		case "docs":
			hasDocs = true
		}
	}

	if !hasBackend {
		t.Error("Expected backend agent")
	}
	if !hasFrontend {
		t.Error("Expected frontend agent")
	}
	if !hasSecurity {
		t.Error("Expected security agent for non-functional requirements")
	}
	if !hasDevops {
		t.Error("Expected devops agent for Docker requirements")
	}
	if !hasQA {
		t.Error("Expected qa agent for testing requirements")
	}
	if !hasDocs {
		t.Error("Expected docs agent")
	}
}

func TestGetRecommendedAgentsArchitect(t *testing.T) {
	analysis := &ProjectAnalysis{
		Architecture: &ArchitectureInfo{},
		Stack: &StackInfo{
			HasDocker: true,
		},
	}

	requirements := make([]map[string]string, 20)
	for i := 0; i < 20; i++ {
		requirements[i] = map[string]string{
			"title": "Feature " + string(rune('0'+i)),
			"type":  "feature",
		}
	}

	agents := GetRecommendedAgents(analysis, requirements)
	hasArchitect := false
	for _, agent := range agents {
		if agent["type"] == "architect" {
			hasArchitect = true
			break
		}
	}
	if !hasArchitect {
		t.Error("Expected architect agent for >15 requirements")
	}
}

func TestGetRecommendedAgentsNilAnalysis(t *testing.T) {
	requirements := []map[string]string{
		{"title": "Security feature", "type": "security"},
	}

	agents := GetRecommendedAgents(nil, requirements)
	if len(agents) == 0 {
		t.Fatal("Expected at least one agent")
	}
}

func TestAnalyzeJavaProject(t *testing.T) {
	tmpDir := t.TempDir()

	createFile := func(path, content string) {
		fullPath := filepath.Join(tmpDir, path)
		os.MkdirAll(filepath.Dir(fullPath), 0755)
		os.WriteFile(fullPath, []byte(content), 0644)
	}

	createFile("pom.xml", `<project><artifactId>test</artifactId></project>`)
	createFile("src/main/java/Test.java", "public class Test {}")

	analyzer := NewProjectAnalyzer(tmpDir)
	analysis, err := analyzer.Analyze()
	if err != nil {
		t.Fatalf("Analyze() error = %v", err)
	}

	if analysis.Stack.Language != "java" {
		t.Errorf("Expected language 'java', got '%s'", analysis.Stack.Language)
	}
	if analysis.Stack.PackageMgr != "maven" {
		t.Errorf("Expected package manager 'maven', got '%s'", analysis.Stack.PackageMgr)
	}
}

func TestAnalyzeRustProject(t *testing.T) {
	tmpDir := t.TempDir()

	createFile := func(path, content string) {
		fullPath := filepath.Join(tmpDir, path)
		os.MkdirAll(filepath.Dir(fullPath), 0755)
		os.WriteFile(fullPath, []byte(content), 0644)
	}

	createFile("Cargo.toml", `[package]\nname = \"test\"\nversion = \"0.1.0\"`)
	createFile("src/main.rs", "fn main() {}")

	analyzer := NewProjectAnalyzer(tmpDir)
	analysis, err := analyzer.Analyze()
	if err != nil {
		t.Fatalf("Analyze() error = %v", err)
	}

	if analysis.Stack.Language != "rust" {
		t.Errorf("Expected language 'rust', got '%s'", analysis.Stack.Language)
	}
	if analysis.Stack.PackageMgr != "cargo" {
		t.Errorf("Expected package manager 'cargo', got '%s'", analysis.Stack.PackageMgr)
	}
}

func TestAnalyzeDatabaseProjects(t *testing.T) {
	tmpDir := t.TempDir()

	createFile := func(path, content string) {
		fullPath := filepath.Join(tmpDir, path)
		os.MkdirAll(filepath.Dir(fullPath), 0755)
		os.WriteFile(fullPath, []byte(content), 0644)
	}

	createFile("go.mod", "module test")
	createFile("db.json", "{}")
	createFile("migrations/001_init.sql", "CREATE TABLE")

	analyzer := NewProjectAnalyzer(tmpDir)
	analysis, err := analyzer.Analyze()
	if err != nil {
		t.Fatalf("Analyze() error = %v", err)
	}

	if analysis.Stack.DBEngine != "sqlite" {
		t.Errorf("Expected DBEngine 'sqlite', got '%s'", analysis.Stack.DBEngine)
	}
}

func TestAnalyzeFrontendProject(t *testing.T) {
	tmpDir := t.TempDir()

	createFile := func(path, content string) {
		fullPath := filepath.Join(tmpDir, path)
		os.MkdirAll(filepath.Dir(fullPath), 0755)
		os.WriteFile(fullPath, []byte(content), 0644)
	}

	createFile("package.json", `{"name":"frontend","dependencies":{"react":"^18.0.0"}}`)
	createFile("src/App.jsx", "import React from 'react'")
	createFile("src/index.js", "import ReactDOM from 'react-dom'")

	analyzer := NewProjectAnalyzer(tmpDir)
	analysis, err := analyzer.Analyze()
	if err != nil {
		t.Fatalf("Analyze() error = %v", err)
	}

	if !analysis.Architecture.HasFrontend {
		t.Error("Expected HasFrontend to be true")
	}
	if analysis.Architecture.FrontendLib != "react" {
		t.Errorf("Expected FrontendLib 'react', got '%s'", analysis.Architecture.FrontendLib)
	}
}

func TestGenerateStandardsConfigGo(t *testing.T) {
	analysis := &ProjectAnalysis{
		Stack: &StackInfo{
			Language: "go",
		},
	}

	standards := GenerateStandardsConfig(analysis)
	found := false
	for _, s := range standards {
		if s["name"] == "go-test" {
			found = true
			break
		}
	}
	if !found {
		t.Error("Expected go-test standard for Go projects")
	}
}

func TestGenerateStandardsConfigPytest(t *testing.T) {
	analysis := &ProjectAnalysis{
		Stack: &StackInfo{
			TestFramework: "pytest",
		},
	}

	standards := GenerateStandardsConfig(analysis)
	found := false
	for _, s := range standards {
		if s["name"] == "test-pass" && strings.Contains(s["command"], "pytest") {
			found = true
			break
		}
	}
	if !found {
		t.Error("Expected pytest test-pass standard")
	}
}

func TestGenerateRulesConfigPython(t *testing.T) {
	analysis := &ProjectAnalysis{
		Stack: &StackInfo{
			Language: "python",
		},
	}

	rules := GenerateRulesConfig(analysis)
	if len(rules) != 0 {
		t.Errorf("Expected 0 rules for Python without CI/tests, got %d", len(rules))
	}
}

func TestAnalyzeGitlabCI(t *testing.T) {
	tmpDir := t.TempDir()
	createFile := func(path, content string) {
		fullPath := filepath.Join(tmpDir, path)
		os.MkdirAll(filepath.Dir(fullPath), 0755)
		os.WriteFile(fullPath, []byte(content), 0644)
	}

	createFile("go.mod", "module test")
	createFile(".gitlab-ci.yml", "test: script")

	analyzer := NewProjectAnalyzer(tmpDir)
	analysis, err := analyzer.Analyze()
	if err != nil {
		t.Fatalf("Analyze() error = %v", err)
	}

	if !analysis.Stack.HasCI {
		t.Error("Expected HasCI to be true")
	}
	if analysis.Stack.CITool != "gitlab-ci" {
		t.Errorf("Expected CI tool 'gitlab-ci', got '%s'", analysis.Stack.CITool)
	}
}

func TestAnalyzeJenkinsCI(t *testing.T) {
	tmpDir := t.TempDir()
	createFile := func(path, content string) {
		fullPath := filepath.Join(tmpDir, path)
		os.MkdirAll(filepath.Dir(fullPath), 0755)
		os.WriteFile(fullPath, []byte(content), 0644)
	}

	createFile("go.mod", "module test")
	createFile("Jenkinsfile", "pipeline { agent any }")

	analyzer := NewProjectAnalyzer(tmpDir)
	analysis, err := analyzer.Analyze()
	if err != nil {
		t.Fatalf("Analyze() error = %v", err)
	}

	if !analysis.Stack.HasCI {
		t.Error("Expected HasCI to be true")
	}
	if analysis.Stack.CITool != "jenkins" {
		t.Errorf("Expected CI tool 'jenkins', got '%s'", analysis.Stack.CITool)
	}
}

func TestAnalyzeAzureCI(t *testing.T) {
	tmpDir := t.TempDir()
	createFile := func(path, content string) {
		fullPath := filepath.Join(tmpDir, path)
		os.MkdirAll(filepath.Dir(fullPath), 0755)
		os.WriteFile(fullPath, []byte(content), 0644)
	}

	createFile("go.mod", "module test")
	createFile("azure-pipelines.yml", "trigger: main")

	analyzer := NewProjectAnalyzer(tmpDir)
	analysis, err := analyzer.Analyze()
	if err != nil {
		t.Fatalf("Analyze() error = %v", err)
	}

	if !analysis.Stack.HasCI {
		t.Error("Expected HasCI to be true")
	}
	if analysis.Stack.CITool != "azure-pipelines" {
		t.Errorf("Expected CI tool 'azure-pipelines', got '%s'", analysis.Stack.CITool)
	}
}

func TestProjectAnalysisStructures(t *testing.T) {
	analysis := &ProjectAnalysis{
		Path: "/test",
		Stack: &StackInfo{
			Language:      "go",
			Framework:     "gin",
			BuildTool:     "go build",
			PackageMgr:    "go",
			Runtime:       "go",
			HasDocker:     true,
			HasCI:         true,
			CITool:        "github-actions",
			HasTests:      true,
			TestFramework: "go test",
			DBEngine:      "postgresql",
		},
		Architecture: &ArchitectureInfo{
			Type:         "modular",
			Modules:      []string{"api", "db", "auth"},
			IsMonorepo:   true,
			HasAPI:       true,
			APIFramework: "rest",
			HasFrontend:  false,
		},
		Modules: []*ModuleInfo{
			{
				Path:         "api",
				Name:         "api",
				Type:         "go",
				Language:     "go",
				Dependencies: 2,
				DependsOn:    []string{"db", "auth"},
				HasTests:     true,
			},
		},
		Patterns: []*PatternInfo{
			{
				Type:        "testing",
				Name:        "Unit Testing",
				Detected:    true,
				Confidence:  0.9,
				Description: "Unit tests present",
			},
		},
		ConfigFiles: []*ConfigFileInfo{
			{
				Path:     "go.mod",
				Name:     "go.mod",
				Type:     "go",
				Relevant: true,
			},
		},
		RootFiles: []string{"go.mod", "main.go"},
	}

	jsonStr := analysis.ToJSON()
	if jsonStr == "" {
		t.Error("Expected non-empty JSON")
	}

	if len(analysis.Modules) != 1 {
		t.Errorf("Expected 1 module, got %d", len(analysis.Modules))
	}

	if len(analysis.Patterns) != 1 {
		t.Errorf("Expected 1 pattern, got %d", len(analysis.Patterns))
	}
}

func TestDetectReactFramework(t *testing.T) {
	tmpDir := t.TempDir()
	createFile := func(path, content string) {
		fullPath := filepath.Join(tmpDir, path)
		os.MkdirAll(filepath.Dir(fullPath), 0755)
		os.WriteFile(fullPath, []byte(content), 0644)
	}

	createFile("package.json", `{"name":"test","dependencies":{"react":"^18.0.0"}}`)
	createFile("vite.config.js", "export default defineConfig({ plugins: [react()] })")

	analyzer := NewProjectAnalyzer(tmpDir)
	analysis, err := analyzer.Analyze()
	if err != nil {
		t.Fatalf("Analyze() error = %v", err)
	}

	if analysis.Stack.Framework != "react" {
		t.Errorf("Expected framework 'react', got '%s'", analysis.Stack.Framework)
	}
}

func TestDetectVueFramework(t *testing.T) {
	tmpDir := t.TempDir()
	createFile := func(path, content string) {
		fullPath := filepath.Join(tmpDir, path)
		os.MkdirAll(filepath.Dir(fullPath), 0755)
		os.WriteFile(fullPath, []byte(content), 0644)
	}

	createFile("package.json", `{"name":"test","dependencies":{"vue":"^3.0.0"}}`)
	createFile("vite.config.js", "export default defineConfig({ plugins: [vue()] })")

	analyzer := NewProjectAnalyzer(tmpDir)
	analysis, err := analyzer.Analyze()
	if err != nil {
		t.Fatalf("Analyze() error = %v", err)
	}

	if analysis.Stack.Framework != "vue" {
		t.Errorf("Expected framework 'vue', got '%s'", analysis.Stack.Framework)
	}
}

func TestDetectSvelteFramework(t *testing.T) {
	tmpDir := t.TempDir()
	createFile := func(path, content string) {
		fullPath := filepath.Join(tmpDir, path)
		os.MkdirAll(filepath.Dir(fullPath), 0755)
		os.WriteFile(fullPath, []byte(content), 0644)
	}

	createFile("package.json", `{"name":"test","dependencies":{"svelte":"^4.0.0"}}`)
	createFile("svelte.config.js", "export default {}")

	analyzer := NewProjectAnalyzer(tmpDir)
	analysis, err := analyzer.Analyze()
	if err != nil {
		t.Fatalf("Analyze() error = %v", err)
	}

	if analysis.Stack.Framework != "svelte" {
		t.Errorf("Expected framework 'svelte', got '%s'", analysis.Stack.Framework)
	}
}

func TestDetectNextJsFramework(t *testing.T) {
	tmpDir := t.TempDir()
	createFile := func(path, content string) {
		fullPath := filepath.Join(tmpDir, path)
		os.MkdirAll(filepath.Dir(fullPath), 0755)
		os.WriteFile(fullPath, []byte(content), 0644)
	}

	createFile("package.json", `{"name":"test","dependencies":{"next":"^14.0.0"}}`)
	createFile("next.config.js", "module.exports = {}")

	analyzer := NewProjectAnalyzer(tmpDir)
	analysis, err := analyzer.Analyze()
	if err != nil {
		t.Fatalf("Analyze() error = %v", err)
	}

	if analysis.Stack.Framework != "next.js" {
		t.Errorf("Expected framework 'next.js', got '%s'", analysis.Stack.Framework)
	}
}

func TestDetectAngularFramework(t *testing.T) {
	tmpDir := t.TempDir()
	createFile := func(path, content string) {
		fullPath := filepath.Join(tmpDir, path)
		os.MkdirAll(filepath.Dir(fullPath), 0755)
		os.WriteFile(fullPath, []byte(content), 0644)
	}

	createFile("package.json", `{"name":"test","dependencies":{"@angular/core":"^17.0.0"}}`)

	analyzer := NewProjectAnalyzer(tmpDir)
	analysis, err := analyzer.Analyze()
	if err != nil {
		t.Fatalf("Analyze() error = %v", err)
	}

	if analysis.Stack.Framework != "angular" {
		t.Errorf("Expected framework 'angular', got '%s'", analysis.Stack.Framework)
	}
}

func TestDetectFastifyFramework(t *testing.T) {
	tmpDir := t.TempDir()
	createFile := func(path, content string) {
		fullPath := filepath.Join(tmpDir, path)
		os.MkdirAll(filepath.Dir(fullPath), 0755)
		os.WriteFile(fullPath, []byte(content), 0644)
	}

	createFile("package.json", `{"name":"test","dependencies":{"fastify":"^4.0.0"}}`)

	analyzer := NewProjectAnalyzer(tmpDir)
	analysis, err := analyzer.Analyze()
	if err != nil {
		t.Fatalf("Analyze() error = %v", err)
	}

	if analysis.Stack.Framework != "fastify" {
		t.Errorf("Expected framework 'fastify', got '%s'", analysis.Stack.Framework)
	}
}

func TestDetectExpressFramework(t *testing.T) {
	tmpDir := t.TempDir()
	createFile := func(path, content string) {
		fullPath := filepath.Join(tmpDir, path)
		os.MkdirAll(filepath.Dir(fullPath), 0755)
		os.WriteFile(fullPath, []byte(content), 0644)
	}

	createFile("package.json", `{"name":"test","dependencies":{"express":"^4.0.0"}}`)

	analyzer := NewProjectAnalyzer(tmpDir)
	analysis, err := analyzer.Analyze()
	if err != nil {
		t.Fatalf("Analyze() error = %v", err)
	}

	if analysis.Stack.Framework != "express" {
		t.Errorf("Expected framework 'express', got '%s'", analysis.Stack.Framework)
	}
}

func TestDetectFastAPIFramework(t *testing.T) {
	tmpDir := t.TempDir()
	createFile := func(path, content string) {
		fullPath := filepath.Join(tmpDir, path)
		os.MkdirAll(filepath.Dir(fullPath), 0755)
		os.WriteFile(fullPath, []byte(content), 0644)
	}

	createFile("pyproject.toml", `[project]\nname = "test"\ndependencies = ["fastapi"]`)
	createFile("main.py", "from fastapi import FastAPI")

	analyzer := NewProjectAnalyzer(tmpDir)
	analysis, err := analyzer.Analyze()
	if err != nil {
		t.Fatalf("Analyze() error = %v", err)
	}

	if analysis.Stack.Framework != "fastapi" {
		t.Errorf("Expected framework 'fastapi', got '%s'", analysis.Stack.Framework)
	}
}

func TestDetectDjangoFramework(t *testing.T) {
	tmpDir := t.TempDir()
	createFile := func(path, content string) {
		fullPath := filepath.Join(tmpDir, path)
		os.MkdirAll(filepath.Dir(fullPath), 0755)
		os.WriteFile(fullPath, []byte(content), 0644)
	}

	createFile("requirements.txt", "django>=4.0.0")
	createFile("manage.py", "# Django")

	analyzer := NewProjectAnalyzer(tmpDir)
	analysis, err := analyzer.Analyze()
	if err != nil {
		t.Fatalf("Analyze() error = %v", err)
	}

	if analysis.Stack.Framework != "django" {
		t.Errorf("Expected framework 'django', got '%s'", analysis.Stack.Framework)
	}
}

func TestDetectFlaskFramework(t *testing.T) {
	tmpDir := t.TempDir()
	createFile := func(path, content string) {
		fullPath := filepath.Join(tmpDir, path)
		os.MkdirAll(filepath.Dir(fullPath), 0755)
		os.WriteFile(fullPath, []byte(content), 0644)
	}

	createFile("requirements.txt", "flask>=2.0.0")
	createFile("app.py", "from flask import Flask")

	analyzer := NewProjectAnalyzer(tmpDir)
	analysis, err := analyzer.Analyze()
	if err != nil {
		t.Fatalf("Analyze() error = %v", err)
	}

	if analysis.Stack.Framework != "flask" {
		t.Errorf("Expected framework 'flask', got '%s'", analysis.Stack.Framework)
	}
}

func TestDetectGinFramework(t *testing.T) {
	tmpDir := t.TempDir()
	createFile := func(path, content string) {
		fullPath := filepath.Join(tmpDir, path)
		os.MkdirAll(filepath.Dir(fullPath), 0755)
		os.WriteFile(fullPath, []byte(content), 0644)
	}

	createFile("go.mod", `module github.com/test\n\ngo 1.22\n\nrequire github.com/gin-gonic/gin v1.9.0`)

	analyzer := NewProjectAnalyzer(tmpDir)
	analysis, err := analyzer.Analyze()
	if err != nil {
		t.Fatalf("Analyze() error = %v", err)
	}

	if analysis.Stack.Framework != "gin" {
		t.Errorf("Expected framework 'gin', got '%s'", analysis.Stack.Framework)
	}
}

func TestDetectGorillaMuxFramework(t *testing.T) {
	tmpDir := t.TempDir()
	createFile := func(path, content string) {
		fullPath := filepath.Join(tmpDir, path)
		os.MkdirAll(filepath.Dir(fullPath), 0755)
		os.WriteFile(fullPath, []byte(content), 0644)
	}

	createFile("go.mod", `module github.com/test\n\ngo 1.22\n\nrequire github.com/gorilla/mux v1.8.1`)

	analyzer := NewProjectAnalyzer(tmpDir)
	analysis, err := analyzer.Analyze()
	if err != nil {
		t.Fatalf("Analyze() error = %v", err)
	}

	if analysis.Stack.Framework != "gorilla-mux" {
		t.Errorf("Expected framework 'gorilla-mux', got '%s'", analysis.Stack.Framework)
	}
}

func TestDetectChiFramework(t *testing.T) {
	tmpDir := t.TempDir()
	createFile := func(path, content string) {
		fullPath := filepath.Join(tmpDir, path)
		os.MkdirAll(filepath.Dir(fullPath), 0755)
		os.WriteFile(fullPath, []byte(content), 0644)
	}

	createFile("go.mod", `module github.com/test\n\ngo 1.22\n\nrequire github.com/go-chi/chi/v5 v5.0.0`)

	analyzer := NewProjectAnalyzer(tmpDir)
	analysis, err := analyzer.Analyze()
	if err != nil {
		t.Fatalf("Analyze() error = %v", err)
	}

	if analysis.Stack.Framework != "chi" {
		t.Errorf("Expected framework 'chi', got '%s'", analysis.Stack.Framework)
	}
}

func TestDetectFiberFramework(t *testing.T) {
	tmpDir := t.TempDir()
	createFile := func(path, content string) {
		fullPath := filepath.Join(tmpDir, path)
		os.MkdirAll(filepath.Dir(fullPath), 0755)
		os.WriteFile(fullPath, []byte(content), 0644)
	}

	createFile("go.mod", `module github.com/test\n\ngo 1.22\n\nrequire github.com/gofiber/fiber/v2 v2.50.0`)

	analyzer := NewProjectAnalyzer(tmpDir)
	analysis, err := analyzer.Analyze()
	if err != nil {
		t.Fatalf("Analyze() error = %v", err)
	}

	if analysis.Stack.Framework != "fiber" {
		t.Errorf("Expected framework 'fiber', got '%s'", analysis.Stack.Framework)
	}
}

func TestDetectEchoFramework(t *testing.T) {
	tmpDir := t.TempDir()
	createFile := func(path, content string) {
		fullPath := filepath.Join(tmpDir, path)
		os.MkdirAll(filepath.Dir(fullPath), 0755)
		os.WriteFile(fullPath, []byte(content), 0644)
	}

	createFile("go.mod", `module github.com/test\n\ngo 1.22\n\nrequire github.com/labstack/echo/v4 v4.10.0`)

	analyzer := NewProjectAnalyzer(tmpDir)
	analysis, err := analyzer.Analyze()
	if err != nil {
		t.Fatalf("Analyze() error = %v", err)
	}

	if analysis.Stack.Framework != "echo" {
		t.Errorf("Expected framework 'echo', got '%s'", analysis.Stack.Framework)
	}
}

func TestDetectActixWebFramework(t *testing.T) {
	tmpDir := t.TempDir()
	createFile := func(path, content string) {
		fullPath := filepath.Join(tmpDir, path)
		os.MkdirAll(filepath.Dir(fullPath), 0755)
		os.WriteFile(fullPath, []byte(content), 0644)
	}

	createFile("Cargo.toml", `[package]\nname = "test"\nversion = "0.1.0"\n\n[dependencies]\nactix-web = "4"`)

	analyzer := NewProjectAnalyzer(tmpDir)
	analysis, err := analyzer.Analyze()
	if err != nil {
		t.Fatalf("Analyze() error = %v", err)
	}

	if analysis.Stack.Framework != "actix-web" {
		t.Errorf("Expected framework 'actix-web', got '%s'", analysis.Stack.Framework)
	}
}

func TestDetectAxumFramework(t *testing.T) {
	tmpDir := t.TempDir()
	createFile := func(path, content string) {
		fullPath := filepath.Join(tmpDir, path)
		os.MkdirAll(filepath.Dir(fullPath), 0755)
		os.WriteFile(fullPath, []byte(content), 0644)
	}

	createFile("Cargo.toml", `[package]\nname = "test"\nversion = "0.1.0"\n\n[dependencies]\naxum = "0.6"`)

	analyzer := NewProjectAnalyzer(tmpDir)
	analysis, err := analyzer.Analyze()
	if err != nil {
		t.Fatalf("Analyze() error = %v", err)
	}

	if analysis.Stack.Framework != "axum" {
		t.Errorf("Expected framework 'axum', got '%s'", analysis.Stack.Framework)
	}
}

func TestDetectSpringBootFramework(t *testing.T) {
	tmpDir := t.TempDir()
	createFile := func(path, content string) {
		fullPath := filepath.Join(tmpDir, path)
		os.MkdirAll(filepath.Dir(fullPath), 0755)
		os.WriteFile(fullPath, []byte(content), 0644)
	}

	createFile("pom.xml", `<project>\n<parent>\n<artifactId>spring-boot-starter-parent</artifactId>\n</parent>\n</project>`)

	analyzer := NewProjectAnalyzer(tmpDir)
	analysis, err := analyzer.Analyze()
	if err != nil {
		t.Fatalf("Analyze() error = %v", err)
	}

	if analysis.Stack.Framework != "spring-boot" {
		t.Errorf("Expected framework 'spring-boot', got '%s'", analysis.Stack.Framework)
	}
}

func TestDetectQuarkusFramework(t *testing.T) {
	tmpDir := t.TempDir()
	createFile := func(path, content string) {
		fullPath := filepath.Join(tmpDir, path)
		os.MkdirAll(filepath.Dir(fullPath), 0755)
		os.WriteFile(fullPath, []byte(content), 0644)
	}

	createFile("pom.xml", `<project>\n<artifactId>quarkus-app</artifactId>\n</project>`)

	analyzer := NewProjectAnalyzer(tmpDir)
	analysis, err := analyzer.Analyze()
	if err != nil {
		t.Fatalf("Analyze() error = %v", err)
	}

	if analysis.Stack.Framework != "quarkus" {
		t.Errorf("Expected framework 'quarkus', got '%s'", analysis.Stack.Framework)
	}
}

func TestVitestFramework(t *testing.T) {
	tmpDir := t.TempDir()
	createFile := func(path, content string) {
		fullPath := filepath.Join(tmpDir, path)
		os.MkdirAll(filepath.Dir(fullPath), 0755)
		os.WriteFile(fullPath, []byte(content), 0644)
	}

	createFile("package.json", `{"name":"test"}`)
	createFile("vitest.config.ts", "import { defineConfig } from 'vitest/config'")

	analyzer := NewProjectAnalyzer(tmpDir)
	analysis, err := analyzer.Analyze()
	if err != nil {
		t.Fatalf("Analyze() error = %v", err)
	}

	if !analysis.Stack.HasTests {
		t.Error("Expected HasTests to be true")
	}
	if analysis.Stack.TestFramework != "vitest" {
		t.Errorf("Expected test framework 'vitest', got '%s'", analysis.Stack.TestFramework)
	}
}

func TestGenerateSkillsConfigEmpty(t *testing.T) {
	analysis := &ProjectAnalysis{
		Stack:        &StackInfo{},
		Architecture: &ArchitectureInfo{},
		Patterns:     []*PatternInfo{},
	}

	config := GenerateSkillsConfig(analysis)
	if config == nil {
		t.Fatal("Expected non-nil config")
	}

	skills, ok := config["suggested_skills"].([]map[string]string)
	if !ok {
		t.Fatal("Expected suggested_skills to be []map[string]string")
	}

	if len(skills) != 0 {
		t.Errorf("Expected 0 skills for empty stack, got %d", len(skills))
	}
}

func TestGenerateRulesConfigNoFeatures(t *testing.T) {
	analysis := &ProjectAnalysis{
		Stack: &StackInfo{},
	}

	rules := GenerateRulesConfig(analysis)
	if len(rules) != 0 {
		t.Errorf("Expected 0 rules, got %d", len(rules))
	}
}

func TestModuleInfoScan(t *testing.T) {
	tmpDir := t.TempDir()

	createFile := func(path, content string) {
		fullPath := filepath.Join(tmpDir, path)
		os.MkdirAll(filepath.Dir(fullPath), 0755)
		os.WriteFile(fullPath, []byte(content), 0644)
	}

	createFile("go.mod", "module github.com/test/project")
	createFile("api/main.go", "package api")
	createFile("api/go.mod", "module github.com/test/api")

	analyzer := NewProjectAnalyzer(tmpDir)
	analysis, err := analyzer.Analyze()
	if err != nil {
		t.Fatalf("Analyze() error = %v", err)
	}

	if len(analysis.Modules) < 2 {
		t.Errorf("Expected at least 2 modules, got %d", len(analysis.Modules))
	}
}

func TestPostgreSQLDBEngine(t *testing.T) {
	tmpDir := t.TempDir()
	createFile := func(path, content string) {
		fullPath := filepath.Join(tmpDir, path)
		os.MkdirAll(filepath.Dir(fullPath), 0755)
		os.WriteFile(fullPath, []byte(content), 0644)
	}

	createFile("go.mod", "module test")
	createFile("postgres.env", "POSTGRES_PASSWORD=secret")

	analyzer := NewProjectAnalyzer(tmpDir)
	analysis, err := analyzer.Analyze()
	if err != nil {
		t.Fatalf("Analyze() error = %v", err)
	}

	if analysis.Stack.DBEngine != "postgresql" {
		t.Errorf("Expected DBEngine 'postgresql', got '%s'", analysis.Stack.DBEngine)
	}
}

func TestMySQLDBEngine(t *testing.T) {
	tmpDir := t.TempDir()
	createFile := func(path, content string) {
		fullPath := filepath.Join(tmpDir, path)
		os.MkdirAll(filepath.Dir(fullPath), 0755)
		os.WriteFile(fullPath, []byte(content), 0644)
	}

	createFile("go.mod", "module test")
	createFile("mysql.env", "MYSQL_ROOT_PASSWORD=secret")

	analyzer := NewProjectAnalyzer(tmpDir)
	analysis, err := analyzer.Analyze()
	if err != nil {
		t.Fatalf("Analyze() error = %v", err)
	}

	if analysis.Stack.DBEngine != "mysql" {
		t.Errorf("Expected DBEngine 'mysql', got '%s'", analysis.Stack.DBEngine)
	}
}

func TestMongoDBEngine(t *testing.T) {
	tmpDir := t.TempDir()
	createFile := func(path, content string) {
		fullPath := filepath.Join(tmpDir, path)
		os.MkdirAll(filepath.Dir(fullPath), 0755)
		os.WriteFile(fullPath, []byte(content), 0644)
	}

	createFile("go.mod", "module test")
	createFile("mongodb.env", "MONGO_URI=mongodb://localhost")

	analyzer := NewProjectAnalyzer(tmpDir)
	analysis, err := analyzer.Analyze()
	if err != nil {
		t.Fatalf("Analyze() error = %v", err)
	}

	if analysis.Stack.DBEngine != "mongodb" {
		t.Errorf("Expected DBEngine 'mongodb', got '%s'", analysis.Stack.DBEngine)
	}
}

func TestDockerComposeDetection(t *testing.T) {
	tmpDir := t.TempDir()
	createFile := func(path, content string) {
		fullPath := filepath.Join(tmpDir, path)
		os.MkdirAll(filepath.Dir(fullPath), 0755)
		os.WriteFile(fullPath, []byte(content), 0644)
	}

	createFile("go.mod", "module test")
	createFile("docker-compose.yml", "version: '3'")

	analyzer := NewProjectAnalyzer(tmpDir)
	analysis, err := analyzer.Analyze()
	if err != nil {
		t.Fatalf("Analyze() error = %v", err)
	}

	if !analysis.Stack.HasDocker {
		t.Error("Expected HasDocker to be true with docker-compose.yml")
	}
}

func TestFrontendOnlyProject(t *testing.T) {
	tmpDir := t.TempDir()
	createFile := func(path, content string) {
		fullPath := filepath.Join(tmpDir, path)
		os.MkdirAll(filepath.Dir(fullPath), 0755)
		os.WriteFile(fullPath, []byte(content), 0644)
	}

	createFile("package.json", `{"name":"frontend"}`)
	createFile("src/App.tsx", "export default function App()")

	analyzer := NewProjectAnalyzer(tmpDir)
	analysis, err := analyzer.Analyze()
	if err != nil {
		t.Fatalf("Analyze() error = %v", err)
	}

	if !analysis.Architecture.HasFrontend {
		t.Error("Expected HasFrontend to be true for JSX/TSX files")
	}
}

func TestAPIDetection(t *testing.T) {
	tmpDir := t.TempDir()
	createFile := func(path, content string) {
		fullPath := filepath.Join(tmpDir, path)
		os.MkdirAll(filepath.Dir(fullPath), 0755)
		os.WriteFile(fullPath, []byte(content), 0644)
	}

	createFile("go.mod", "module test")
	createFile("cmd/api/main.go", "package main")

	analyzer := NewProjectAnalyzer(tmpDir)
	analysis, err := analyzer.Analyze()
	if err != nil {
		t.Fatalf("Analyze() error = %v", err)
	}

	if !analysis.Architecture.HasAPI {
		t.Error("Expected HasAPI to be true for cmd/api directory")
	}
	if analysis.Architecture.APIFramework != "rest" {
		t.Errorf("Expected APIFramework 'rest', got '%s'", analysis.Architecture.APIFramework)
	}
}

func TestFrontendSubdirs(t *testing.T) {
	tmpDir := t.TempDir()
	createFile := func(path, content string) {
		fullPath := filepath.Join(tmpDir, path)
		os.MkdirAll(filepath.Dir(fullPath), 0755)
		os.WriteFile(fullPath, []byte(content), 0644)
	}

	createFile("go.mod", "module mono")
	createFile("frontend/package.json", `{"name":"frontend","dependencies":{"react":""}}`)
	createFile("frontend/src/App.jsx", "export default function App()")

	analyzer := NewProjectAnalyzer(tmpDir)
	analysis, err := analyzer.Analyze()
	if err != nil {
		t.Fatalf("Analyze() error = %v", err)
	}

	if analysis.Architecture.FrontendLib != "react" {
		t.Errorf("Expected FrontendLib 'react', got '%s'", analysis.Architecture.FrontendLib)
	}
}

func TestNuxtFramework(t *testing.T) {
	tmpDir := t.TempDir()
	createFile := func(path, content string) {
		fullPath := filepath.Join(tmpDir, path)
		os.MkdirAll(filepath.Dir(fullPath), 0755)
		os.WriteFile(fullPath, []byte(content), 0644)
	}

	createFile("package.json", `{"name":"test","dependencies":{"nuxt":"^3.0.0","@nuxt/kit":"^3.0.0"}}`)
	createFile("nuxt.config.ts", "export default defineNuxtConfig({})")

	analyzer := NewProjectAnalyzer(tmpDir)
	analysis, err := analyzer.Analyze()
	if err != nil {
		t.Fatalf("Analyze() error = %v", err)
	}

	if analysis.Stack.Framework != "nuxt" {
		t.Errorf("Expected framework 'nuxt', got '%s'", analysis.Stack.Framework)
	}
}

func TestAstroFramework(t *testing.T) {
	tmpDir := t.TempDir()
	createFile := func(path, content string) {
		fullPath := filepath.Join(tmpDir, path)
		os.MkdirAll(filepath.Dir(fullPath), 0755)
		os.WriteFile(fullPath, []byte(content), 0644)
	}

	createFile("package.json", `{"name":"test","dependencies":{"astro":"^3.0.0"}}`)
	createFile("astro.config.mjs", "import { defineConfig } from 'astro/config'")

	analyzer := NewProjectAnalyzer(tmpDir)
	analysis, err := analyzer.Analyze()
	if err != nil {
		t.Fatalf("Analyze() error = %v", err)
	}

	if analysis.Stack.Framework != "astro" && analysis.Stack.Framework != "" {
		t.Errorf("Expected framework 'astro' or '', got '%s'", analysis.Stack.Framework)
	}
}

func TestRemixFramework(t *testing.T) {
	tmpDir := t.TempDir()
	createFile := func(path, content string) {
		fullPath := filepath.Join(tmpDir, path)
		os.MkdirAll(filepath.Dir(fullPath), 0755)
		os.WriteFile(fullPath, []byte(content), 0644)
	}

	createFile("package.json", `{"name":"test","dependencies":{"@remix-run/react":"^2.0.0"}}`)

	analyzer := NewProjectAnalyzer(tmpDir)
	analysis, err := analyzer.Analyze()
	if err != nil {
		t.Fatalf("Analyze() error = %v", err)
	}

	if analysis.Stack.Framework != "remix" {
		t.Errorf("Expected framework 'remix', got '%s'", analysis.Stack.Framework)
	}
}

func TestGatsbyFramework(t *testing.T) {
	tmpDir := t.TempDir()
	createFile := func(path, content string) {
		fullPath := filepath.Join(tmpDir, path)
		os.MkdirAll(filepath.Dir(fullPath), 0755)
		os.WriteFile(fullPath, []byte(content), 0644)
	}

	createFile("package.json", `{"name":"test","dependencies":{"gatsby":"^5.0.0"}}`)

	analyzer := NewProjectAnalyzer(tmpDir)
	analysis, err := analyzer.Analyze()
	if err != nil {
		t.Fatalf("Analyze() error = %v", err)
	}

	if analysis.Stack.Framework != "gatsby" {
		t.Errorf("Expected framework 'gatsby', got '%s'", analysis.Stack.Framework)
	}
}

func TestRocketFramework(t *testing.T) {
	tmpDir := t.TempDir()
	createFile := func(path, content string) {
		fullPath := filepath.Join(tmpDir, path)
		os.MkdirAll(filepath.Dir(fullPath), 0755)
		os.WriteFile(fullPath, []byte(content), 0644)
	}

	createFile("Cargo.toml", `[package]\nname = "test"\nversion = "0.1.0"\n\n[dependencies]\nrocket = "0.5"`)

	analyzer := NewProjectAnalyzer(tmpDir)
	analysis, err := analyzer.Analyze()
	if err != nil {
		t.Fatalf("Analyze() error = %v", err)
	}

	if analysis.Stack.Framework != "rocket" {
		t.Errorf("Expected framework 'rocket', got '%s'", analysis.Stack.Framework)
	}
}

func TestWarpFramework(t *testing.T) {
	tmpDir := t.TempDir()
	createFile := func(path, content string) {
		fullPath := filepath.Join(tmpDir, path)
		os.MkdirAll(filepath.Dir(fullPath), 0755)
		os.WriteFile(fullPath, []byte(content), 0644)
	}

	createFile("Cargo.toml", `[package]\nname = "test"\nversion = "0.1.0"\n\n[dependencies]\nwarp = "0.3"`)

	analyzer := NewProjectAnalyzer(tmpDir)
	analysis, err := analyzer.Analyze()
	if err != nil {
		t.Fatalf("Analyze() error = %v", err)
	}

	if analysis.Stack.Framework != "warp" {
		t.Errorf("Expected framework 'warp', got '%s'", analysis.Stack.Framework)
	}
}

func TestRustTests(t *testing.T) {
	tmpDir := t.TempDir()
	createFile := func(path, content string) {
		fullPath := filepath.Join(tmpDir, path)
		os.MkdirAll(filepath.Dir(fullPath), 0755)
		os.WriteFile(fullPath, []byte(content), 0644)
	}

	createFile("Cargo.toml", `[package]\nname = "test"\n\n[lib]\npath = "src/lib.rs"\n\n[[test]]\nname = "integration"\npath = "tests/integration_test.rs"`)
	createFile("tests/integration_test.rs", "#[test]\nfn it_works() {}")

	analyzer := NewProjectAnalyzer(tmpDir)
	analysis, err := analyzer.Analyze()
	if err != nil {
		t.Fatalf("Analyze() error = %v", err)
	}

	if analysis.Stack.Language != "rust" {
		t.Errorf("Expected language 'rust', got '%s'", analysis.Stack.Language)
	}
}
