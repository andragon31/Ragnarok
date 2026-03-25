package precommit

import (
	"os"
	"os/exec"
	"regexp"
	"strings"
	"time"
)

type SyntaxChecker struct {
	languageHandlers map[string]func(content string) []*ValidationError
}

func NewSyntaxChecker() *SyntaxChecker {
	sc := &SyntaxChecker{
		languageHandlers: make(map[string]func(content string) []*ValidationError),
	}
	sc.languageHandlers["go"] = sc.checkGoSyntax
	sc.languageHandlers["typescript"] = sc.checkTypeScriptSyntax
	sc.languageHandlers["javascript"] = sc.checkJavaScriptSyntax
	sc.languageHandlers["python"] = sc.checkPythonSyntax
	return sc
}

type ValidationError struct {
	Type       string
	File       string
	Line       int
	Column     int
	Message    string
	Fixed      bool
	CanAutoFix bool
}

type ValidationResult struct {
	Passed     bool
	DurationMs int64
	Errors     []*ValidationError
	Warnings   []*ValidationWarning
	FileCount  int
}

type ValidationWarning struct {
	Type    string
	File    string
	Line    int
	Message string
}

func (sc *SyntaxChecker) Check(file *FileChange) []*ValidationError {
	handler, ok := sc.languageHandlers[file.Language]
	if !ok {
		return nil
	}
	return handler(file.Content)
}

func (sc *SyntaxChecker) checkGoSyntax(content string) []*ValidationError {
	var errors []*ValidationError

	if strings.Contains(content, "undefined") {
		errors = append(errors, &ValidationError{
			Type:       "syntax",
			Message:    "undefined keyword not valid in Go",
			CanAutoFix: false,
		})
	}

	if strings.Contains(content, "== undefined") {
		errors = append(errors, &ValidationError{
			Type:       "syntax",
			Message:    "use '== nil' for nil comparison in Go",
			CanAutoFix: true,
		})
	}

	lines := strings.Split(content, "\n")
	for i, line := range lines {
		if strings.HasPrefix(strings.TrimSpace(line), "func (") && strings.Contains(line, "{") {
			if !strings.Contains(line, ") {") && !strings.Contains(line, ") {") {
				errors = append(errors, &ValidationError{
					Type:       "syntax",
					Line:       i + 1,
					Message:    "function receiver must be wrapped",
					CanAutoFix: true,
				})
			}
		}
	}

	return errors
}

func (sc *SyntaxChecker) checkTypeScriptSyntax(content string) []*ValidationError {
	var errors []*ValidationError

	if strings.Contains(content, "== undefined") {
		errors = append(errors, &ValidationError{
			Type:       "syntax",
			Message:    "use '=== undefined' for strict comparison",
			CanAutoFix: true,
		})
	}

	if strings.Contains(content, "require(") && !strings.Contains(content, "import ") {
		if !strings.Contains(content, "require.resolve") {
			errors = append(errors, &ValidationError{
				Type:       "import",
				Message:    "consider using ES6 import instead of require",
				CanAutoFix: false,
			})
		}
	}

	return errors
}

func (sc *SyntaxChecker) checkJavaScriptSyntax(content string) []*ValidationError {
	var errors []*ValidationError

	if strings.Contains(content, "== undefined") {
		errors = append(errors, &ValidationError{
			Type:       "syntax",
			Message:    "use '=== undefined' for strict comparison",
			CanAutoFix: true,
		})
	}

	return errors
}

func (sc *SyntaxChecker) checkPythonSyntax(content string) []*ValidationError {
	var errors []*ValidationError

	lines := strings.Split(content, "\n")
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "except:") && !strings.HasPrefix(trimmed, "except Exception:") && !strings.HasPrefix(trimmed, "except BaseException:") {
			errors = append(errors, &ValidationError{
				Type:       "syntax",
				Line:       i + 1,
				Message:    "bare except clause may catch unintended exceptions",
				CanAutoFix: false,
			})
		}
	}

	return errors
}

type FileChange struct {
	Path     string
	Content  string
	Language string
	Changes  []string
}

type SyntaxCheckResult struct {
	File   string
	Valid  bool
	Errors []*ValidationError
}

func CheckSyntaxWithGo(files []*FileChange) []*SyntaxCheckResult {
	var results []*SyntaxCheckResult

	for _, file := range files {
		result := &SyntaxCheckResult{File: file.Path, Valid: true, Errors: []*ValidationError{}}

		if file.Language == "go" {
			tmpFile := "/tmp/check_" + time.Now().Format("20060102150405") + ".go"
			if err := os.WriteFile(tmpFile, []byte(file.Content), 0644); err == nil {
				defer os.Remove(tmpFile)

				cmd := exec.Command("go", "fmt", tmpFile)
				if err := cmd.Run(); err == nil {
					if content, err := os.ReadFile(tmpFile); err == nil {
						if string(content) != file.Content {
							result.Valid = false
							result.Errors = append(result.Errors, &ValidationError{
								Type:       "format",
								Message:    "file is not gofmt compliant",
								CanAutoFix: true,
							})
						}
					}
				}
			}
		}

		if len(result.Errors) == 0 {
			result.Valid = true
		}
		results = append(results, result)
	}

	return results
}

var trailingWhitespaceRegex = regexp.MustCompile(`[ \t]+$`)
var duplicateImportRegex = regexp.MustCompile(`(?m)^import (.+)\nimport \1$`)

func CheckFormatting(content string, language string) []*ValidationWarning {
	var warnings []*ValidationWarning

	if matches := trailingWhitespaceRegex.FindAllStringIndex(content, -1); len(matches) > 0 {
		for _, match := range matches {
			line := strings.Count(content[:match[0]], "\n") + 1
			warnings = append(warnings, &ValidationWarning{
				Type:    "formatting",
				Line:    line,
				Message: "trailing whitespace",
			})
		}
	}

	if language == "python" || language == "go" {
		lines := strings.Split(content, "\n")
		for i, line := range lines {
			if len(line) > 120 {
				warnings = append(warnings, &ValidationWarning{
					Type:    "style",
					Line:    i + 1,
					Message: "line exceeds 120 characters",
				})
			}
		}
	}

	return warnings
}
