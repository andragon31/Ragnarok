package scanner

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

type ProjectAnalyzer struct {
	projectPath string
}

type ProjectAnalysis struct {
	Path         string            `json:"path"`
	Name         string            `json:"name"`
	Stack        *StackInfo        `json:"stack"`
	Architecture *ArchitectureInfo `json:"architecture"`
	Modules      []*ModuleInfo     `json:"modules"`
	Patterns     []*PatternInfo    `json:"patterns"`
	ConfigFiles  []*ConfigFileInfo `json:"config_files"`
	RootFiles    []string          `json:"root_files"`
}

type StackInfo struct {
	Language      string `json:"language"`
	Framework     string `json:"framework"`
	BuildTool     string `json:"build_tool"`
	PackageMgr    string `json:"package_manager"`
	Runtime       string `json:"runtime"`
	HasDocker     bool   `json:"has_docker"`
	HasCI         bool   `json:"has_ci"`
	CITool        string `json:"ci_tool"`
	HasTests      bool   `json:"has_tests"`
	TestFramework string `json:"test_framework"`
	DBEngine      string `json:"db_engine"`
}

type ArchitectureInfo struct {
	Type         string   `json:"type"`
	Modules      []string `json:"modules"`
	IsMonorepo   bool     `json:"is_monorepo"`
	HasAPI       bool     `json:"has_api"`
	APIFramework string   `json:"api_framework"`
	HasFrontend  bool     `json:"has_frontend"`
	FrontendLib  string   `json:"frontend_lib"`
}

type ModuleInfo struct {
	Path         string   `json:"path"`
	Name         string   `json:"name"`
	Type         string   `json:"type"`
	Language     string   `json:"language"`
	Dependencies int      `json:"dependencies"`
	DependsOn    []string `json:"depends_on,omitempty"`
	HasTests     bool     `json:"has_tests"`
}

type PatternInfo struct {
	Type        string   `json:"type"`
	Name        string   `json:"name"`
	Detected    bool     `json:"detected"`
	Confidence  float64  `json:"confidence"`
	Description string   `json:"description"`
	Files       []string `json:"files,omitempty"`
}

type ConfigFileInfo struct {
	Path     string `json:"path"`
	Name     string `json:"name"`
	Type     string `json:"type"`
	Relevant bool   `json:"relevant"`
}

func NewProjectAnalyzer(projectPath string) *ProjectAnalyzer {
	return &ProjectAnalyzer{projectPath: projectPath}
}

func (a *ProjectAnalyzer) Analyze() (*ProjectAnalysis, error) {
	analysis := &ProjectAnalysis{
		Path:         a.projectPath,
		Stack:        &StackInfo{},
		Architecture: &ArchitectureInfo{},
		Modules:      []*ModuleInfo{},
		Patterns:     []*PatternInfo{},
		ConfigFiles:  []*ConfigFileInfo{},
		RootFiles:    []string{},
	}

	if err := a.walkProject(analysis); err != nil {
		return nil, err
	}

	a.detectStack(analysis)
	a.detectArchitecture(analysis)
	a.detectPatterns(analysis)
	a.resolveDependencies(analysis)

	return analysis, nil
}

func (a *ProjectAnalyzer) walkProject(analysis *ProjectAnalysis) error {
	return filepath.Walk(a.projectPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}

		if info.IsDir() {
			relPath, _ := filepath.Rel(a.projectPath, path)

			if relPath == "." {
				a.analyzeDirectory(path, relPath, analysis)
				return nil
			}

			skipDirs := []string{".git", ".hidden", "node_modules", "__pycache__", "vendor", ".venv"}
			shouldSkip := false
			for _, skip := range skipDirs {
				if relPath == skip || strings.HasPrefix(relPath, skip+string(filepath.Separator)) {
					shouldSkip = true
					break
				}
			}
			if shouldSkip {
				return filepath.SkipDir
			}

			a.analyzeDirectory(path, relPath, analysis)
		} else {
			relPath, _ := filepath.Rel(a.projectPath, path)
			analysis.RootFiles = append(analysis.RootFiles, relPath)
			a.analyzeFile(path, relPath, analysis)
		}

		return nil
	})
}

func (a *ProjectAnalyzer) analyzeDirectory(path, relPath string, analysis *ProjectAnalysis) {
	entries, err := os.ReadDir(path)
	if err != nil {
		return
	}

	module := &ModuleInfo{
		Path: relPath,
		Name: filepath.Base(path),
	}

	for _, entry := range entries {
		name := entry.Name()
		switch name {
		case "package.json":
			module.Type = "npm"
			module.Language = "javascript"
			analysis.Stack.PackageMgr = "npm"
		case "go.mod":
			module.Type = "go"
			module.Language = "go"
			analysis.Stack.PackageMgr = "go"
		case "requirements.txt", "pyproject.toml", "setup.py":
			module.Type = "python"
			module.Language = "python"
			analysis.Stack.PackageMgr = "pip"
		case "Cargo.toml":
			module.Type = "rust"
			module.Language = "rust"
			analysis.Stack.PackageMgr = "cargo"
		case "pom.xml":
			module.Type = "java"
			module.Language = "java"
			analysis.Stack.PackageMgr = "maven"
		case "build.gradle":
			module.Type = "java"
			module.Language = "java"
			analysis.Stack.PackageMgr = "gradle"
		case "tests", "__tests__", "test":
			module.HasTests = true
			analysis.Stack.HasTests = true
		}
	}

	if module.Type != "" {
		analysis.Modules = append(analysis.Modules, module)
	}
}

func (a *ProjectAnalyzer) analyzeFile(path, relPath string, analysis *ProjectAnalysis) {
	baseName := filepath.Base(path)

	configFiles := map[string]string{
		"package.json":        "npm",
		"go.mod":              "go",
		"go.sum":              "go",
		"requirements.txt":    "python",
		"pyproject.toml":      "python",
		"setup.py":            "python",
		"Cargo.toml":          "rust",
		"pom.xml":             "java",
		"build.gradle":        "java",
		"docker-compose.yml":  "docker",
		"docker-compose.yaml": "docker",
		"Dockerfile":          "docker",
		"jest.config.js":      "test",
		"vitest.config.ts":    "test",
		"pytest.ini":          "test",
		"tox.ini":             "test",
		"tsconfig.json":       "typescript",
		"next.config.js":      "framework",
		"nuxt.config.ts":      "framework",
		".env.example":        "config",
		".env.sample":         "config",
	}

	if fileType, ok := configFiles[baseName]; ok {
		analysis.ConfigFiles = append(analysis.ConfigFiles, &ConfigFileInfo{
			Path:     relPath,
			Name:     baseName,
			Type:     fileType,
			Relevant: true,
		})
	}
}

func (a *ProjectAnalyzer) detectStack(analysis *ProjectAnalysis) {
	has := func(file string) bool {
		for _, f := range analysis.RootFiles {
			if f == file || filepath.Base(f) == file {
				return true
			}
		}
		return false
	}

	hasPrefix := func(prefix string) bool {
		for _, f := range analysis.RootFiles {
			if strings.HasPrefix(filepath.ToSlash(f), prefix) {
				return true
			}
		}
		return false
	}

	readFileContent := func(filename string) string {
		content, _ := os.ReadFile(filepath.Join(a.projectPath, filename))
		return string(content)
	}

	detectFrontend := func() {
		frontendDirs := []string{
			"", "frontend", "client", "web", "apps/web", "packages/web", "packages/ui",
			"ui", "src", "app", "apps/client",
		}

		scanDir := func(dir string) bool {
			pkgPath := filepath.Join(a.projectPath, dir, "package.json")
			if dir == "" {
				pkgPath = filepath.Join(a.projectPath, "package.json")
			}
			content, err := os.ReadFile(pkgPath)
			if err != nil {
				return false
			}
			contentStr := string(content)

			if hasPrefix(filepath.Join(dir, "app")) || has(filepath.Join(dir, "next.config")) {
				analysis.Stack.Framework = "next.js"
				return true
			}
			if has(filepath.Join(dir, "nuxt.config")) {
				analysis.Stack.Framework = "nuxt"
				return true
			}
			if has(filepath.Join(dir, "astro.config")) {
				analysis.Stack.Framework = "astro"
				return true
			}
			if has(filepath.Join(dir, "svelte.config")) {
				analysis.Stack.Framework = "sveltekit"
				return true
			}
			if has(filepath.Join(dir, "vite.config")) || has(filepath.Join(dir, "vitest.config")) {
				if strings.Contains(contentStr, "\"vue\"") || strings.Contains(contentStr, "'vue'") {
					analysis.Stack.Framework = "vue"
				} else if strings.Contains(contentStr, "\"react\"") || strings.Contains(contentStr, "'react'") {
					analysis.Stack.Framework = "react"
				} else {
					analysis.Stack.Framework = "vite"
				}
				return true
			}
			if strings.Contains(contentStr, "next") {
				analysis.Stack.Framework = "next.js"
				return true
			}
			if strings.Contains(contentStr, "@nuxt/") {
				analysis.Stack.Framework = "nuxt"
				return true
			}
			if strings.Contains(contentStr, "\"react\"") || strings.Contains(contentStr, "'react'") {
				analysis.Stack.Framework = "react"
				return true
			}
			if strings.Contains(contentStr, "\"vue\"") || strings.Contains(contentStr, "'vue'") {
				analysis.Stack.Framework = "vue"
				return true
			}
			if strings.Contains(contentStr, "\"@angular/core\"") {
				analysis.Stack.Framework = "angular"
				return true
			}
			if strings.Contains(contentStr, "remix") {
				analysis.Stack.Framework = "remix"
				return true
			}
			if strings.Contains(contentStr, "gatsby") {
				analysis.Stack.Framework = "gatsby"
				return true
			}
			if strings.Contains(contentStr, "svelte") {
				analysis.Stack.Framework = "svelte"
				return true
			}
			if strings.Contains(contentStr, "express") {
				analysis.Stack.Framework = "express"
				return true
			}
			if strings.Contains(contentStr, "fastify") {
				analysis.Stack.Framework = "fastify"
				return true
			}
			return false
		}

		for _, dir := range frontendDirs {
			if dir == "" {
				if scanDir("") {
					break
				}
			} else {
				if scanDir(dir) {
					break
				}
			}
		}

		extCounts := map[string]int{"tsx": 0, "jsx": 0, "vue": 0, "svelte": 0}
		for _, f := range analysis.RootFiles {
			lower := strings.ToLower(f)
			for ext := range extCounts {
				if strings.HasSuffix(lower, "."+ext) {
					extCounts[ext]++
				}
			}
		}

		if analysis.Stack.Framework == "" {
			if extCounts["tsx"] > 0 || extCounts["jsx"] > 0 {
				analysis.Stack.Framework = "react"
			} else if extCounts["vue"] > 0 {
				analysis.Stack.Framework = "vue"
			} else if extCounts["svelte"] > 0 {
				analysis.Stack.Framework = "svelte"
			}
		}
	}

	if has("package.json") || hasPrefix("frontend/package.json") || hasPrefix("client/package.json") || hasPrefix("apps/web/package.json") {
		analysis.Stack.Language = "javascript/typescript"
		analysis.Stack.PackageMgr = "npm"

		if has("tsconfig.json") || hasPrefix("frontend/tsconfig.json") || hasPrefix("apps/web/tsconfig.json") {
			analysis.Stack.Language = "typescript"
		}

		detectFrontend()
	}

	if has("go.mod") {
		analysis.Stack.Language = "go"
		analysis.Stack.PackageMgr = "go"

		content := readFileContent("go.mod")
		if strings.Contains(content, "gin-gonic") || strings.Contains(content, "github.com/gin-gonic") {
			analysis.Stack.Framework = "gin"
		} else if strings.Contains(content, "gorilla/mux") {
			analysis.Stack.Framework = "gorilla-mux"
		} else if strings.Contains(content, "chi-middleware") || strings.Contains(content, "go-chi") {
			analysis.Stack.Framework = "chi"
		} else if strings.Contains(content, "fiber") || strings.Contains(content, "gofiber") {
			analysis.Stack.Framework = "fiber"
		} else if strings.Contains(content, "echo") || strings.Contains(content, "labstack/echo") {
			analysis.Stack.Framework = "echo"
		}
	}

	if has("requirements.txt") || has("pyproject.toml") || has("setup.py") {
		analysis.Stack.Language = "python"
		analysis.Stack.PackageMgr = "pip"

		if has("manage.py") || has("Django") {
			analysis.Stack.Framework = "django"
		} else if has("pyproject.toml") {
			content := readFileContent("pyproject.toml")
			if strings.Contains(content, "flask") || strings.Contains(content, "Flask") {
				analysis.Stack.Framework = "flask"
			} else if strings.Contains(content, "fastapi") || strings.Contains(content, "FastAPI") {
				analysis.Stack.Framework = "fastapi"
			} else if strings.Contains(content, "django") || strings.Contains(content, "Django") {
				analysis.Stack.Framework = "django"
			} else if strings.Contains(content, "pytest") || strings.Contains(content, "pytest") {
				analysis.Stack.Framework = "pytest"
			}
		} else if has("requirements.txt") {
			content := readFileContent("requirements.txt")
			if strings.Contains(content, "django") {
				analysis.Stack.Framework = "django"
			} else if strings.Contains(content, "flask") {
				analysis.Stack.Framework = "flask"
			} else if strings.Contains(content, "fastapi") {
				analysis.Stack.Framework = "fastapi"
			}
		}
	}

	if has("Cargo.toml") {
		analysis.Stack.Language = "rust"
		analysis.Stack.PackageMgr = "cargo"

		content := readFileContent("Cargo.toml")
		if strings.Contains(content, "actix-web") || strings.Contains(content, "actix") {
			analysis.Stack.Framework = "actix-web"
		} else if strings.Contains(content, "axum") {
			analysis.Stack.Framework = "axum"
		} else if strings.Contains(content, "warp") {
			analysis.Stack.Framework = "warp"
		} else if strings.Contains(content, "rocket") {
			analysis.Stack.Framework = "rocket"
		}
	}

	if has("pom.xml") || has("build.gradle") {
		analysis.Stack.Language = "java"
		if has("pom.xml") {
			analysis.Stack.PackageMgr = "maven"
		} else {
			analysis.Stack.PackageMgr = "gradle"
		}

		if has("pom.xml") {
			content := readFileContent("pom.xml")
			if strings.Contains(content, "spring-boot") || strings.Contains(content, "springframework") {
				analysis.Stack.Framework = "spring-boot"
			} else if strings.Contains(content, "quarkus") {
				analysis.Stack.Framework = "quarkus"
			} else if strings.Contains(content, "micronaut") {
				analysis.Stack.Framework = "micronaut"
			}
		} else {
			content := readFileContent("build.gradle")
			if strings.Contains(content, "spring-boot") || strings.Contains(content, "springframework") {
				analysis.Stack.Framework = "spring-boot"
			} else if strings.Contains(content, "quarkus") {
				analysis.Stack.Framework = "quarkus"
			}
		}
	}

	if has("Dockerfile") || has("docker-compose.yml") {
		analysis.Stack.HasDocker = true
	}

	if hasPrefix(".github"+string(filepath.Separator)+"workflows") || hasPrefix(".github/workflows") {
		analysis.Stack.HasCI = true
		analysis.Stack.CITool = "github-actions"
	} else if has("azure-pipelines.yml") || has("azure-pipelines.yaml") {
		analysis.Stack.HasCI = true
		analysis.Stack.CITool = "azure-pipelines"
	} else if has(".gitlab-ci.yml") {
		analysis.Stack.HasCI = true
		analysis.Stack.CITool = "gitlab-ci"
	} else if has("Jenkinsfile") {
		analysis.Stack.HasCI = true
		analysis.Stack.CITool = "jenkins"
	}

	if has("jest.config.js") || has("jest.config.ts") || has("jest.config.json") {
		analysis.Stack.HasTests = true
		analysis.Stack.TestFramework = "jest"
	} else if has("vitest.config.ts") || has("vitest.config.js") {
		analysis.Stack.HasTests = true
		analysis.Stack.TestFramework = "vitest"
	} else if has("pytest.ini") || has("setup.cfg") || hasPrefix("tests/") || hasPrefix("test/") {
		analysis.Stack.HasTests = true
		analysis.Stack.TestFramework = "pytest"
	} else if has("Cargo.toml") {
		if hasPrefix("tests/") || hasPrefix("src/tests/") {
			analysis.Stack.HasTests = true
			analysis.Stack.TestFramework = "rust-test"
		}
	}

	if has("db.json") || has("database.json") || hasPrefix("migrations/") || hasPrefix("seeds/") {
		analysis.Stack.DBEngine = "sqlite"
	}
	if has("postgres.env") || has("postgresql.env") || strings.Contains(strings.Join(analysis.RootFiles, ""), "postgres") {
		analysis.Stack.DBEngine = "postgresql"
	}
	if has("mysql.env") || strings.Contains(strings.Join(analysis.RootFiles, ""), "mysql") {
		analysis.Stack.DBEngine = "mysql"
	}
	if has("mongodb.env") || strings.Contains(strings.Join(analysis.RootFiles, ""), "mongodb") {
		analysis.Stack.DBEngine = "mongodb"
	}

	analysis.Stack.Runtime = analysis.Stack.Language
}

func (a *ProjectAnalyzer) detectArchitecture(analysis *ProjectAnalysis) {
	modules := make(map[string]bool)
	for _, m := range analysis.Modules {
		modules[m.Type] = true
	}

	if len(analysis.Modules) > 3 {
		analysis.Architecture.Type = "modular"
	} else if len(analysis.Modules) == 1 {
		analysis.Architecture.Type = "monolith"
	}

	analysis.Architecture.IsMonorepo = len(analysis.Modules) > 5

	hasPrefix := func(prefix string) bool {
		for _, f := range analysis.RootFiles {
			if strings.HasPrefix(filepath.ToSlash(f), prefix) {
				return true
			}
		}
		return false
	}

	scanForPatterns := func() {
		frontendPatterns := []string{
			"src/", "app/", "pages/", "components/", "views/",
			"frontend/src/", "frontend/components/", "frontend/pages/",
			"client/src/", "client/components/", "client/pages/",
			"web/src/", "web/components/", "web/pages/",
			"apps/web/src/", "apps/web/components/",
			"packages/web/src/", "packages/ui/src/",
			"ui/src/", "ui/components/",
		}

		reactFiles := []string{}
		vueFiles := []string{}
		svelteFiles := []string{}

		for _, f := range analysis.RootFiles {
			lower := strings.ToLower(f)
			ext := filepath.Ext(f)

			for _, pattern := range frontendPatterns {
				if strings.Contains(lower, pattern) {
					analysis.Architecture.HasFrontend = true
					break
				}
			}

			if ext == ".tsx" || ext == ".jsx" {
				reactFiles = append(reactFiles, f)
			} else if ext == ".vue" {
				vueFiles = append(vueFiles, f)
			} else if ext == ".svelte" {
				svelteFiles = append(svelteFiles, f)
			}
		}

		if len(reactFiles) > 0 {
			analysis.Architecture.HasFrontend = true
			analysis.Architecture.FrontendLib = "react"
		}
		if len(vueFiles) > 0 {
			analysis.Architecture.HasFrontend = true
			analysis.Architecture.FrontendLib = "vue"
		}
		if len(svelteFiles) > 0 {
			analysis.Architecture.HasFrontend = true
			analysis.Architecture.FrontendLib = "svelte"
		}

		if hasPrefix("src/api") || hasPrefix("api/") || hasPrefix("cmd/") || hasPrefix("backend/") || hasPrefix("server/") {
			analysis.Architecture.HasAPI = true
			analysis.Architecture.APIFramework = "rest"
		}
	}

	scanForPatterns()

	for _, m := range analysis.Modules {
		analysis.Architecture.Modules = append(analysis.Architecture.Modules, m.Name)
	}
}

func (a *ProjectAnalyzer) detectPatterns(analysis *ProjectAnalysis) {
	patterns := []*PatternInfo{
		{
			Type:        "testing",
			Name:        "Unit Testing",
			Detected:    analysis.Stack.HasTests,
			Confidence:  0.9,
			Description: "Unit tests are present in the project",
		},
		{
			Type:        "ci",
			Name:        "Continuous Integration",
			Detected:    analysis.Stack.HasCI,
			Confidence:  0.95,
			Description: "CI/CD pipeline is configured",
		},
		{
			Type:        "docker",
			Name:        "Containerization",
			Detected:    analysis.Stack.HasDocker,
			Confidence:  0.95,
			Description: "Docker is used for containerization",
		},
		{
			Type:        "typescript",
			Name:        "TypeScript",
			Detected:    analysis.Stack.Language == "typescript" || analysis.Stack.Language == "javascript/typescript",
			Confidence:  0.8,
			Description: "Project uses TypeScript",
		},
	}

	analysis.Patterns = append(analysis.Patterns, patterns...)
}

func GenerateSkillsConfig(analysis *ProjectAnalysis) map[string]interface{} {
	config := make(map[string]interface{})

	config["stack"] = analysis.Stack
	config["architecture"] = analysis.Architecture
	config["patterns"] = analysis.Patterns

	skills := []map[string]string{}

	if analysis.Stack.Framework != "" {
		skills = append(skills, map[string]string{
			"name":  strings.ToLower(strings.ReplaceAll(analysis.Stack.Framework, ".", "-")),
			"type":  "framework",
			"skill": analysis.Stack.Framework,
		})
	}

	if analysis.Stack.Language != "" {
		skills = append(skills, map[string]string{
			"name":  strings.ToLower(analysis.Stack.Language),
			"type":  "language",
			"skill": analysis.Stack.Language,
		})
	}

	if analysis.Stack.TestFramework != "" {
		skills = append(skills, map[string]string{
			"name":  strings.ToLower(analysis.Stack.TestFramework),
			"type":  "testing",
			"skill": analysis.Stack.TestFramework,
		})
	}

	config["suggested_skills"] = skills

	return config
}

func GenerateRulesConfig(analysis *ProjectAnalysis) []map[string]string {
	rules := []map[string]string{}

	if analysis.Stack.HasTests {
		rules = append(rules, map[string]string{
			"name":        "no-commit-without-tests",
			"category":    "quality",
			"description": "Commits that modify code must include or update tests",
			"severity":    "high",
		})
	}

	if analysis.Stack.Framework == "next.js" || analysis.Stack.Framework == "nuxt" {
		rules = append(rules, map[string]string{
			"name":        "use-api-routes",
			"category":    "architecture",
			"description": "Use framework API routes for backend endpoints",
			"severity":    "medium",
		})
	}

	if analysis.Stack.Language == "typescript" {
		rules = append(rules, map[string]string{
			"name":        "strict-typescript",
			"category":    "code-quality",
			"description": "Avoid 'any' types, use explicit interfaces",
			"severity":    "medium",
		})
	}

	if analysis.Stack.HasCI {
		rules = append(rules, map[string]string{
			"name":        "ci-must-pass",
			"category":    "process",
			"description": "All PRs must pass CI before merging",
			"severity":    "high",
		})
	}

	return rules
}

func GenerateRuleFingerprint(name, category, description string) string {
	input := strings.Join([]string{
		strings.ToLower(name),
		strings.ToLower(category),
		extractKeywords(description),
	}, "|")

	hash := sha256.Sum256([]byte(input))
	return hex.EncodeToString(hash[:])
}

func extractKeywords(text string) string {
	text = strings.ToLower(text)
	text = regexp.MustCompile(`[^a-z0-9\s]`).ReplaceAllString(text, "")
	words := strings.Fields(text)
	if len(words) <= 5 {
		return strings.Join(words, ",")
	}
	sort.Strings(words)
	return strings.Join(words[:5], ",")
}

func GenerateStandardsConfig(analysis *ProjectAnalysis) []map[string]string {
	standards := []map[string]string{}

	if analysis.Stack.TestFramework == "jest" || analysis.Stack.TestFramework == "vitest" {
		standards = append(standards, map[string]string{
			"name":        "test-pass",
			"command":     "npm test",
			"type":        "test",
			"block":       "true",
			"description": "All tests must pass",
		})
	}

	if analysis.Stack.TestFramework == "pytest" {
		standards = append(standards, map[string]string{
			"name":        "test-pass",
			"command":     "pytest",
			"type":        "test",
			"block":       "true",
			"description": "All pytest tests must pass",
		})
	}

	if analysis.Stack.Language == "go" {
		standards = append(standards, map[string]string{
			"name":        "go-test",
			"command":     "go test ./...",
			"type":        "test",
			"block":       "true",
			"description": "All Go tests must pass",
		})
	}

	if analysis.Stack.Language == "typescript" || analysis.Stack.Language == "javascript/typescript" {
		standards = append(standards, map[string]string{
			"name":        "lint",
			"command":     "npm run lint",
			"type":        "lint",
			"block":       "false",
			"description": "Run linter to check code style",
		})
	}

	return standards
}

func (a *ProjectAnalysis) ToJSON() string {
	b, _ := json.MarshalIndent(a, "", "  ")
	return string(b)
}

type PhaseTemplate struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	Tasks       []TaskTemplate `json:"tasks"`
	AgentType   string         `json:"agent_type"`
}

type TaskTemplate struct {
	Title       string   `json:"title"`
	Description string   `json:"description"`
	Priority    int      `json:"priority"`
	Milestone   bool     `json:"milestone"`
	AgentTypes  []string `json:"agent_types"`
}

func GeneratePhasesAndTasks(analysis *ProjectAnalysis) []PhaseTemplate {
	phases := []PhaseTemplate{}

	if analysis == nil {
		return phases
	}

	stack := analysis.Stack
	arch := analysis.Architecture

	safeGetStr := func(s string) string {
		if s == "" {
			return "unknown"
		}
		return s
	}

	if stack == nil {
		stack = &StackInfo{}
	}
	if arch == nil {
		arch = &ArchitectureInfo{}
	}

	phases = append(phases, PhaseTemplate{
		Name:        "Setup",
		Description: "Initialize project structure and dependencies",
		AgentType:   "backend",
		Tasks: []TaskTemplate{
			{Title: "Setup project structure", Description: "Create base directories and configuration files", Priority: 3, Milestone: false, AgentTypes: []string{"backend"}},
			{Title: "Install dependencies", Description: "Install all required dependencies for " + safeGetStr(stack.PackageMgr), Priority: 3, Milestone: false, AgentTypes: []string{"backend"}},
		},
	})

	if arch.HasAPI || stack.Framework != "" {
		phases = append(phases, PhaseTemplate{
			Name:        "Backend",
			Description: "Implement backend services with " + safeGetStr(stack.Framework),
			AgentType:   "backend",
			Tasks: []TaskTemplate{
				{Title: "Design database schema", Description: "Create database models and migrations", Priority: 3, Milestone: false, AgentTypes: []string{"backend"}},
				{Title: "Implement API endpoints", Description: "Create REST/GraphQL endpoints for " + safeGetStr(stack.Framework), Priority: 3, Milestone: false, AgentTypes: []string{"backend"}},
				{Title: "Implement business logic", Description: "Add service layer and business rules", Priority: 2, Milestone: false, AgentTypes: []string{"backend"}},
				{Title: "Add authentication", Description: "Implement auth endpoints and middleware", Priority: 3, Milestone: false, AgentTypes: []string{"backend", "security"}},
			},
		})
	}

	if arch.HasFrontend || stack.HasDocker {
		frontendFramework := arch.FrontendLib
		if frontendFramework == "" {
			frontendFramework = stack.Framework
		}
		phases = append(phases, PhaseTemplate{
			Name:        "Frontend",
			Description: "Implement frontend with " + frontendFramework,
			AgentType:   "frontend",
			Tasks: []TaskTemplate{
				{Title: "Setup " + frontendFramework + " project", Description: "Initialize frontend app and routing", Priority: 3, Milestone: false, AgentTypes: []string{"frontend"}},
				{Title: "Implement UI components", Description: "Create reusable UI components", Priority: 2, Milestone: false, AgentTypes: []string{"frontend"}},
				{Title: "Integrate API", Description: "Connect frontend to backend API", Priority: 3, Milestone: false, AgentTypes: []string{"frontend", "backend"}},
				{Title: "Add state management", Description: "Implement global state and data fetching", Priority: 2, Milestone: false, AgentTypes: []string{"frontend"}},
			},
		})
	}

	if stack.DBEngine != "" {
		phases = append(phases, PhaseTemplate{
			Name:        "Database",
			Description: "Setup and optimize database for " + safeGetStr(stack.DBEngine),
			AgentType:   "backend",
			Tasks: []TaskTemplate{
				{Title: "Create database schema", Description: "Design and create tables for " + safeGetStr(stack.DBEngine), Priority: 3, Milestone: false, AgentTypes: []string{"backend"}},
				{Title: "Add migrations", Description: "Setup database migration system", Priority: 2, Milestone: false, AgentTypes: []string{"backend"}},
				{Title: "Seed data", Description: "Add seed data for development", Priority: 1, Milestone: false, AgentTypes: []string{"backend"}},
			},
		})
	}

	if stack.HasTests {
		testFramework := stack.TestFramework
		if testFramework == "" {
			testFramework = "testing"
		}
		phases = append(phases, PhaseTemplate{
			Name:        "Testing",
			Description: "Implement tests with " + safeGetStr(testFramework),
			AgentType:   "qa",
			Tasks: []TaskTemplate{
				{Title: "Setup test infrastructure", Description: "Configure " + safeGetStr(testFramework) + " for the project", Priority: 3, Milestone: false, AgentTypes: []string{"qa", "backend"}},
				{Title: "Write unit tests", Description: "Add unit tests for core business logic", Priority: 2, Milestone: false, AgentTypes: []string{"qa", "backend"}},
				{Title: "Write integration tests", Description: "Add integration tests for API endpoints", Priority: 2, Milestone: false, AgentTypes: []string{"qa", "backend"}},
				{Title: "Setup E2E tests", Description: "Add end-to-end tests if applicable", Priority: 1, Milestone: false, AgentTypes: []string{"qa"}},
			},
		})
	}

	if stack.HasDocker || arch.IsMonorepo {
		phases = append(phases, PhaseTemplate{
			Name:        "DevOps",
			Description: "Setup deployment and CI/CD",
			AgentType:   "devops",
			Tasks: []TaskTemplate{
				{Title: "Create Dockerfile", Description: "Add container configuration", Priority: 3, Milestone: false, AgentTypes: []string{"devops"}},
				{Title: "Setup CI/CD pipeline", Description: "Configure " + safeGetStr(stack.CITool) + " for automated builds", Priority: 3, Milestone: false, AgentTypes: []string{"devops"}},
				{Title: "Add deployment config", Description: "Setup deployment scripts and configs", Priority: 2, Milestone: false, AgentTypes: []string{"devops"}},
			},
		})
	}

	phases = append(phases, PhaseTemplate{
		Name:        "Documentation",
		Description: "Create project documentation",
		AgentType:   "docs",
		Tasks: []TaskTemplate{
			{Title: "Write README", Description: "Document project setup and usage", Priority: 3, Milestone: false, AgentTypes: []string{"docs"}},
			{Title: "Document API", Description: "Create API documentation", Priority: 2, Milestone: false, AgentTypes: []string{"docs", "backend"}},
			{Title: "Add inline comments", Description: "Document complex code sections", Priority: 1, Milestone: false, AgentTypes: []string{"backend", "frontend"}},
		},
	})

	return phases
}

type LLMStackAnalysis struct {
	RecommendedAgents []map[string]string `json:"recommended_agents"`
	Reasoning         string              `json:"reasoning"`
	Complexity        string              `json:"complexity"`
	HasFrontend       bool                `json:"has_frontend"`
	HasBackend        bool                `json:"has_backend"`
	HasSecurity       bool                `json:"has_security"`
	HasDevops         bool                `json:"has_devops"`
	HasQA             bool                `json:"has_qa"`
	HasDocs           bool                `json:"has_docs"`
}

func GenerateLLMAnalysisPrompt(analysis *ProjectAnalysis, requirements []map[string]string) string {
	stackLang := "Unknown"
	framework := "Unknown"
	archType := "Unknown"
	hasDocker := false
	hasCI := false
	hasTests := false
	dbEngine := "None"

	if analysis != nil {
		if analysis.Stack != nil {
			stackLang = analysis.Stack.Language
			framework = analysis.Stack.Framework
			hasDocker = analysis.Stack.HasDocker
			hasCI = analysis.Stack.HasCI
			hasTests = analysis.Stack.HasTests
			dbEngine = analysis.Stack.DBEngine
		}
		if analysis.Architecture != nil {
			archType = analysis.Architecture.Type
		}
	}

	reqSummary := ""
	if len(requirements) > 0 {
		reqSummary = fmt.Sprintf("Total requirements: %d\n", len(requirements))
		for i, req := range requirements {
			if i >= 10 {
				reqSummary += fmt.Sprintf("... and %d more\n", len(requirements)-10)
				break
			}
			reqTitle := req["title"]
			reqType := req["type"]
			reqSummary += fmt.Sprintf("- [%s] %s\n", reqType, reqTitle)
		}
	} else {
		reqSummary = "No requirements provided"
	}

	prompt := fmt.Sprintf(`Analyze this software project and recommend the specialized AI agents needed for development.

PROJECT STACK:
- Language: %s
- Framework: %s
- Architecture: %s
- Database: %s
- Has Docker: %v
- Has CI/CD: %v
- Has Tests: %v

REQUIREMENTS:
%s

Based on this information, determine which specialized agents are needed. Consider:

1. Always needed:
   - backend-agent: For API, database, backend services
   - docs-agent: For documentation

2. Optional based on stack/features:
   - frontend-agent: If there's a UI, dashboard, or frontend components
   - security-agent: If handling auth, encryption, payments, or sensitive data
   - devops-agent: If has Docker, CI/CD, deployment infrastructure
   - qa-agent: If has testing requirements or quality gates
   - architect-agent: For complex multi-module systems or microservices

Respond with a JSON object containing:
{
  "recommended_agents": [
    {"name": "agent-name", "type": "agent-type", "role": "Agent Role", "scope": "what this agent handles"}
  ],
  "reasoning": "brief explanation of why these agents",
  "complexity": "low/medium/high",
  "has_frontend": true/false,
  "has_backend": true/false,
  "has_security": true/false,
  "has_devops": true/false,
  "has_qa": true/false,
  "has_docs": true/false
}`, stackLang, framework, archType, dbEngine, hasDocker, hasCI, hasTests, reqSummary)

	return prompt
}

func ParseLLMAnalysisResponse(response string) (*LLMStackAnalysis, error) {
	response = strings.TrimSpace(response)
	startIdx := strings.Index(response, "{")
	endIdx := strings.LastIndex(response, "}")
	if startIdx == -1 || endIdx == -1 {
		return nil, fmt.Errorf("no JSON found in LLM response")
	}

	jsonStr := response[startIdx : endIdx+1]
	var result LLMStackAnalysis
	if err := json.Unmarshal([]byte(jsonStr), &result); err != nil {
		return nil, fmt.Errorf("failed to parse LLM response as JSON: %w", err)
	}

	if len(result.RecommendedAgents) == 0 {
		return nil, fmt.Errorf("LLM returned no recommended agents")
	}

	return &result, nil
}

func GetRecommendedAgents(analysis *ProjectAnalysis, requirements []map[string]string) []map[string]string {
	agents := []map[string]string{}
	hasFrontend := false
	hasSecurity := false
	hasDevops := false
	hasQA := false

	if analysis != nil {
		if analysis.Architecture != nil && analysis.Architecture.HasFrontend {
			hasFrontend = true
		}
		if analysis.Stack != nil {
			if analysis.Stack.HasDocker {
				hasDevops = true
			}
			if analysis.Stack.HasTests {
				hasQA = true
			}
		}
	}

	if requirements != nil {
		for _, req := range requirements {
			title := strings.ToLower(req["title"])
			reqType := strings.ToLower(req["type"])

			if strings.Contains(title, "frontend") || strings.Contains(title, "react") ||
				strings.Contains(title, "ui") || strings.Contains(title, "interfaz") ||
				strings.Contains(title, "dashboard") || strings.Contains(title, "componente") {
				hasFrontend = true
			}

			if strings.Contains(title, "security") || strings.Contains(title, "owasp") ||
				strings.Contains(title, "seguridad") || strings.Contains(title, "autenticación") ||
				strings.Contains(title, "autorización") || strings.Contains(title, "cifrado") ||
				reqType == "non-functional" {
				hasSecurity = true
			}

			if strings.Contains(title, "docker") || strings.Contains(title, "contenedor") ||
				strings.Contains(title, "despliegue") || strings.Contains(title, "deployment") ||
				strings.Contains(title, "infraestructura") || strings.Contains(title, "ci/cd") {
				hasDevops = true
			}

			if strings.Contains(title, "test") || strings.Contains(title, "prueba") ||
				strings.Contains(title, "qa") || strings.Contains(title, "quality") ||
				strings.Contains(title, "calidad") || strings.Contains(title, "e2e") ||
				strings.Contains(title, "integration") {
				hasQA = true
			}
		}
	}

	agents = append(agents, map[string]string{
		"name":  "backend-agent",
		"type":  "backend",
		"role":  "Backend Developer",
		"scope": "API, database, backend services",
	})

	if hasFrontend {
		agents = append(agents, map[string]string{
			"name":  "frontend-agent",
			"type":  "frontend",
			"role":  "Frontend Developer",
			"scope": "UI, components, state management",
		})
	}

	if hasSecurity {
		agents = append(agents, map[string]string{
			"name":  "security-agent",
			"type":  "security",
			"role":  "Security Engineer",
			"scope": "Authentication, authorization, OWASP, encryption",
		})
	}
	if hasDevops {
		agents = append(agents, map[string]string{
			"name":  "devops-agent",
			"type":  "devops",
			"role":  "DevOps Engineer",
			"scope": "Infrastructure, CI/CD, Docker, deployment",
		})
	}
	if hasQA {
		agents = append(agents, map[string]string{
			"name":  "qa-agent",
			"type":  "qa",
			"role":  "QA Automation Engineer",
			"scope": "Unit testing, integration testing, E2E",
		})
	}

	// Nueva lógica: Agente de Arquitectura/Investigación si hay alta complejidad técnica
	if len(requirements) > 15 || hasSecurity && hasDevops {
		agents = append(agents, map[string]string{
			"name":  "architect-agent",
			"type":  "architect",
			"role":  "Software Architect",
			"scope": "System design, tech stack decisions, cross-module consistency",
		})
	}

	agents = append(agents, map[string]string{
		"name":  "docs-agent",
		"type":  "docs",
		"role":  "Technical Writer",
		"scope": "Documentation, guides",
	})

	return agents
}

func (a *ProjectAnalyzer) resolveDependencies(analysis *ProjectAnalysis) {
	for _, module := range analysis.Modules {
		a.scanModuleDependencies(module, analysis)
	}
}

type TaskExecutionPrompt struct {
	TaskID         string            `json:"task_id"`
	Title          string            `json:"title"`
	Description    string            `json:"description"`
	PhaseName      string            `json:"phase_name"`
	ProjectPath    string            `json:"project_path"`
	AgentType      string            `json:"agent_type"`
	Instructions   string            `json:"instructions"`
	ContextFiles   []string          `json:"context_files,omitempty"`
	PRDRequirement map[string]string `json:"prd_requirement,omitempty"`
}

func GenerateTaskExecutionPrompt(task map[string]interface{}, projectPath string, phaseName string) string {
	title, _ := task["title"].(string)
	description, _ := task["description"].(string)
	agentType, _ := task["assigned_agent_type"].(string)
	prdReqID, _ := task["prd_requirement_id"].(string)

	if agentType == "" {
		agentType = "backend"
	}

	var instructions string
	switch agentType {
	case "backend":
		instructions = fmt.Sprintf(`You are a Backend Developer agent. Implement the following task:

TASK: %s

DESCRIPTION: %s

PROJECT PATH: %s

Requirements:
- Write clean, maintainable code following best practices
- Include appropriate error handling
- Add comments for complex logic
- Ensure the code compiles without errors
- If creating new files, use appropriate naming conventions
- Update any relevant configuration files if needed

After completing the task, summarize what you did and any files you created or modified.`, title, description, projectPath)
	case "frontend":
		instructions = fmt.Sprintf(`You are a Frontend Developer agent. Implement the following task:

TASK: %s

DESCRIPTION: %s

PROJECT PATH: %s

Requirements:
- Write clean, maintainable UI code following best practices
- Use appropriate component structure
- Ensure responsive design if applicable
- Add appropriate styling
- Ensure the code has no syntax errors
- Follow the existing code style and patterns

After completing the task, summarize what you did and any files you created or modified.`, title, description, projectPath)
	case "security":
		instructions = fmt.Sprintf(`You are a Security Engineer agent. Implement the following task:

TASK: %s

DESCRIPTION: %s

PROJECT PATH: %s

Requirements:
- Follow OWASP security guidelines
- Implement proper authentication and authorization
- Use encryption where appropriate
- Validate all inputs
- Protect against common vulnerabilities (SQL injection, XSS, CSRF, etc.)
- Document any security considerations

After completing the task, summarize what you did and any files you created or modified.`, title, description, projectPath)
	case "devops":
		instructions = fmt.Sprintf(`You are a DevOps Engineer agent. Implement the following task:

TASK: %s

DESCRIPTION: %s

PROJECT PATH: %s

Requirements:
- Use Docker and containerization best practices
- Configure CI/CD pipelines appropriately
- Ensure infrastructure as code principles
- Monitor and log appropriately
- Follow deployment best practices

After completing the task, summarize what you did and any files you created or modified.`, title, description, projectPath)
	case "qa":
		instructions = fmt.Sprintf(`You are a QA Automation Engineer agent. Implement the following task:

TASK: %s

DESCRIPTION: %s

PROJECT PATH: %s

Requirements:
- Write comprehensive unit tests
- Write integration tests where appropriate
- Follow test automation best practices
- Ensure high code coverage for critical paths
- Use appropriate testing frameworks

After completing the task, summarize what you did and any files you created or modified.`, title, description, projectPath)
	case "architect":
		instructions = fmt.Sprintf(`You are a System Architect agent. Implement the following task:

TASK: %s

DESCRIPTION: %s

PROJECT PATH: %s

Requirements:
- Follow clean architecture principles
- Design for scalability and maintainability
- Consider microservices boundaries if applicable
- Document architectural decisions
- Ensure proper data modeling

After completing the task, summarize what you did and any files you created or modified.`, title, description, projectPath)
	case "docs":
		instructions = fmt.Sprintf(`You are a Technical Writer agent. Implement the following task:

TASK: %s

DESCRIPTION: %s

PROJECT PATH: %s

Requirements:
- Write clear, comprehensive documentation
- Use appropriate markdown formatting
- Include code examples where relevant
- Keep documentation up to date with code changes
- Follow documentation best practices

After completing the task, summarize what you did and any files you created or modified.`, title, description, projectPath)
	default:
		instructions = fmt.Sprintf(`You are a Developer agent. Implement the following task:

TASK: %s

DESCRIPTION: %s

PROJECT PATH: %s

Requirements:
- Write clean, maintainable code
- Follow best practices
- Ensure the code works correctly

After completing the task, summarize what you did.`, title, description, projectPath)
	}

	if prdReqID != "" {
		instructions += fmt.Sprintf("\n\nPRD Requirement ID: %s", prdReqID)
	}

	return instructions
}

func (a *ProjectAnalyzer) scanModuleDependencies(module *ModuleInfo, analysis *ProjectAnalysis) {
	moduleAbsPath := filepath.Join(a.projectPath, module.Path)

	// Limited scan for performance
	filepath.Walk(moduleAbsPath, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() || info.Size() > 100000 { // Skip large files
			return nil
		}

		ext := filepath.Ext(path)
		if ext != ".go" && ext != ".js" && ext != ".ts" && ext != ".py" {
			return nil
		}

		content, err := os.ReadFile(path)
		if err != nil {
			return nil
		}

		// Look for other module names in imports/requires
		for _, other := range analysis.Modules {
			if other.Path == module.Path {
				continue
			}

			// Simple check: does the name or path of the other module appear in this file?
			if strings.Contains(string(content), other.Path) || strings.Contains(string(content), other.Name) {
				alreadyAdded := false
				for _, dep := range module.DependsOn {
					if dep == other.Name {
						alreadyAdded = true
						break
					}
				}
				if !alreadyAdded {
					module.DependsOn = append(module.DependsOn, other.Name)
					module.Dependencies++
				}
			}
		}

		return nil
	})
}
