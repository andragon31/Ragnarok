package precommit

import (
	"regexp"
	"strings"
)

type ImportResolver struct {
	knownImports map[string]bool
}

func NewImportResolver() *ImportResolver {
	return &ImportResolver{
		knownImports: map[string]bool{
			"context":                     true,
			"encoding/json":               true,
			"fmt":                         true,
			"os":                          true,
			"time":                        true,
			"log":                         true,
			"database/sql":                true,
			"github.com/mattn/go-sqlite3": true,
		},
	}
}

type ImportError struct {
	Type       string
	ImportPath string
	File       string
	Line       int
	Message    string
	CanAutoFix bool
}

func (ir *ImportResolver) Resolve(file *FileChange) []*ImportError {
	var errors []*ImportError

	imports := ir.extractImports(file.Content, file.Language)

	for _, imp := range imports {
		if !ir.isValidImport(imp) {
			errors = append(errors, &ImportError{
				Type:       "missing",
				ImportPath: imp,
				File:       file.Path,
				Message:    "import path may not exist or is not accessible",
				CanAutoFix: false,
			})
		}
	}

	if file.Language == "go" {
		usedImports := ir.findUsedImports(file.Content)
		for _, used := range usedImports {
			found := false
			for _, imp := range imports {
				if imp == used || strings.HasPrefix(imp, used) {
					found = true
					break
				}
			}
			if !found && used != "" {
				errors = append(errors, &ImportError{
					Type:       "missing",
					ImportPath: used,
					File:       file.Path,
					Message:    "package imported but not used in code",
					CanAutoFix: false,
				})
			}
		}
	}

	return errors
}

func (ir *ImportResolver) extractImports(content string, language string) []string {
	var imports []string

	switch language {
	case "go":
		importRegex := regexp.MustCompile(`(?m)^import\s+(?:\(\n([\s\S]*?)\n\)|"?([\w/]+)"?)`)
		matches := importRegex.FindAllStringSubmatch(content, -1)
		for _, match := range matches {
			if match[1] != "" {
				lines := strings.Split(match[1], "\n")
				for _, line := range lines {
					imp := strings.TrimSpace(line)
					imp = strings.Trim(imp, `"`)
					if imp != "" {
						imports = append(imports, imp)
					}
				}
			} else if match[2] != "" {
				imports = append(imports, match[2])
			}
		}

	case "typescript", "javascript":
		importRegex := regexp.MustCompile(`(?m)^import\s+(?:(?:\{[^}]*\}|\*\s+as\s+\w+|\w+)\s+from\s+)?['"]([^'"]+)['"]`)
		matches := importRegex.FindAllStringSubmatch(content, -1)
		for _, match := range matches {
			if match[1] != "" {
				imports = append(imports, match[1])
			}
		}

	case "python":
		importRegex := regexp.MustCompile(`(?m)^(?:from\s+([\w.]+)\s+import|import\s+([\w.]+))`)
		matches := importRegex.FindAllStringSubmatch(content, -1)
		for _, match := range matches {
			if match[1] != "" {
				imports = append(imports, match[1])
			} else if match[2] != "" {
				imports = append(imports, match[2])
			}
		}
	}

	return imports
}

func (ir *ImportResolver) findUsedImports(content string) []string {
	var imports []string

	goUseRegex := regexp.MustCompile(`(?m)(?:^|\s)([A-Z][\w]*)\.`)
	matches := goUseRegex.FindAllStringSubmatch(content, -1)
	for _, match := range matches {
		if match[1] != "" && match[1] != "fmt" && match[1] != "os" && match[1] != "log" {
			imports = append(imports, match[1])
		}
	}

	return imports
}

func (ir *ImportResolver) isValidImport(importPath string) bool {
	if ir.knownImports[importPath] {
		return true
	}

	if strings.HasPrefix(importPath, "./") || strings.HasPrefix(importPath, "../") {
		return true
	}

	if strings.HasPrefix(importPath, "github.com/") ||
		strings.HasPrefix(importPath, "golang.org/") ||
		strings.HasPrefix(importPath, "go.uber.org/") {
		return true
	}

	return true
}

func (ir *ImportResolver) SuggestMissingImports(content string, language string) []string {
	var suggestions []string

	switch language {
	case "go":
		stdLibs := []string{"context", "encoding/json", "fmt", "os", "time", "log", "errors", "strings", "bytes", "io"}
		for _, lib := range stdLibs {
			if strings.Contains(content, lib) && !strings.Contains(content, `"`+lib+`"`) {
				suggestions = append(suggestions, lib)
			}
		}
	}

	return suggestions
}

func (ir *ImportResolver) FixMissingImport(content string, importPath string, language string) string {
	switch language {
	case "go":
		insertLine := `import "` + importPath + `"`
		importRegex := regexp.MustCompile(`(?m)^(import\s+\()\n`)
		if importRegex.MatchString(content) {
			return importRegex.ReplaceAllString(content, "$1\n\t"+insertLine+"\n")
		}
		importLineRegex := regexp.MustCompile(`(?m)^import\s+"[^"]+"\n`)
		if importLineRegex.MatchString(content) {
			return importLineRegex.ReplaceAllString(content, "$0\t"+insertLine+"\n")
		}
		return "import " + insertLine + "\n" + content

	case "typescript", "javascript":
		insertLine := `import "` + importPath + `";`
		importRegex := regexp.MustCompile(`(?m)^import\s+.*\n`)
		if importRegex.MatchString(content) {
			return importRegex.ReplaceAllString(content, "$0"+insertLine+"\n")
		}
		return insertLine + "\n" + content
	}

	return content
}
