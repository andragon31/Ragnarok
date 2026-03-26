package scanner

import (
	"encoding/json"
	"os"
	"path/filepath"
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
	Path         string `json:"path"`
	Name         string `json:"name"`
	Type         string `json:"type"`
	Language     string `json:"language"`
	Dependencies int    `json:"dependencies"`
	HasTests     bool   `json:"has_tests"`
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

	if has("package.json") {
		analysis.Stack.Language = "javascript/typescript"
		analysis.Stack.PackageMgr = "npm"

		if has("tsconfig.json") {
			analysis.Stack.Language = "typescript"
		}

		content := readFileContent("package.json")

		if has("next.config.js") || hasPrefix("app/") || has("next-env.d.ts") {
			analysis.Stack.Framework = "next.js"
		} else if has("nuxt.config.ts") || has("nuxt.config.js") {
			analysis.Stack.Framework = "nuxt"
		} else if has("astro.config.mjs") || has("astro.config.js") || has("astro.config.ts") {
			analysis.Stack.Framework = "astro"
		} else if has("svelte.config.js") || has("svelte.config.ts") {
			analysis.Stack.Framework = "sveltekit"
		} else if has("vite.config.ts") || has("vite.config.js") || has("vitest.config.ts") {
			if strings.Contains(content, "vue") {
				analysis.Stack.Framework = "vue"
			} else if strings.Contains(content, "react") {
				analysis.Stack.Framework = "react"
			} else {
				analysis.Stack.Framework = "vite"
			}
		} else if strings.Contains(content, "react") {
			analysis.Stack.Framework = "react"
		} else if strings.Contains(content, "vue") {
			analysis.Stack.Framework = "vue"
		} else if strings.Contains(content, "express") {
			analysis.Stack.Framework = "express"
		} else if strings.Contains(content, "fastify") {
			analysis.Stack.Framework = "fastify"
		} else if strings.Contains(content, "svelte") {
			analysis.Stack.Framework = "svelte"
		} else if strings.Contains(content, "@angular/core") {
			analysis.Stack.Framework = "angular"
		} else if strings.Contains(content, "remix") {
			analysis.Stack.Framework = "remix"
		} else if strings.Contains(content, "gatsby") {
			analysis.Stack.Framework = "gatsby"
		}
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

	if hasPrefix("src/api") || hasPrefix("api/") || hasPrefix("cmd/") {
		analysis.Architecture.HasAPI = true
		analysis.Architecture.APIFramework = "rest"
	}

	if hasPrefix("src/web") || hasPrefix("frontend/") || hasPrefix("app/") {
		analysis.Architecture.HasFrontend = true
	}

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
