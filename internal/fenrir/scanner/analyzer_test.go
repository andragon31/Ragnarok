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

func TestAnalyzeNodeJSProject(t *testing.T) {
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
