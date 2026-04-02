package precommit

import (
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"time"
)

type PreCommitValidator struct {
	syntaxChecker  *SyntaxChecker
	importResolver *ImportResolver
	config         *ValidatorConfig
}

type ValidatorConfig struct {
	MaxDurationMs int64
	AllowAutofix  bool
	StrictMode    bool
	Languages     []string
	IgnorePaths   []string
}

func DefaultValidatorConfig() *ValidatorConfig {
	return &ValidatorConfig{
		MaxDurationMs: 30000,
		AllowAutofix:  true,
		StrictMode:    false,
		Languages:     []string{"go", "typescript", "javascript", "python"},
		IgnorePaths:   []string{"node_modules", ".git", "vendor", "__pycache__"},
	}
}

func NewPreCommitValidator(cfg *ValidatorConfig) *PreCommitValidator {
	if cfg == nil {
		cfg = DefaultValidatorConfig()
	}

	return &PreCommitValidator{
		syntaxChecker:  NewSyntaxChecker(),
		importResolver: NewImportResolver(),
		config:         cfg,
	}
}

type ValidationResponse struct {
	Passed      bool       `json:"passed"`
	DurationMs  int64      `json:"duration_ms"`
	Files       int        `json:"files_validated"`
	Errors      []*Error   `json:"errors,omitempty"`
	Warnings    []*Warning `json:"warnings,omitempty"`
	FixedCount  int        `json:"fixed_count,omitempty"`
	CanContinue bool       `json:"can_continue"`
}

type Error struct {
	Type    string `json:"type"`
	File    string `json:"file"`
	Line    int    `json:"line,omitempty"`
	Column  int    `json:"column,omitempty"`
	Message string `json:"message"`
	Autofix bool   `json:"autofix"`
	Fixed   bool   `json:"fixed,omitempty"`
}

type Warning struct {
	Type    string `json:"type"`
	File    string `json:"file"`
	Line    int    `json:"line,omitempty"`
	Message string `json:"message"`
}

func (v *PreCommitValidator) Validate(files []*FileChange) *ValidationResponse {
	start := time.Now()

	result := &ValidationResponse{
		Passed:      true,
		Files:       len(files),
		Errors:      []*Error{},
		Warnings:    []*Warning{},
		CanContinue: true,
	}

	for _, file := range files {
		if v.shouldIgnore(file.Path) {
			continue
		}

		lang := DetectLanguage(file.Path)
		if lang == "" {
			continue
		}
		file.Language = lang

		if errs := v.checkFile(file); len(errs) > 0 {
			for _, err := range errs {
				result.Errors = append(result.Errors, &Error{
					Type:    err.Type,
					File:    file.Path,
					Line:    err.Line,
					Column:  err.Column,
					Message: err.Message,
					Autofix: err.CanAutoFix,
				})
			}
			result.Passed = false
		}

		if warnings := v.checkWarnings(file); len(warnings) > 0 {
			for _, warn := range warnings {
				result.Warnings = append(result.Warnings, &Warning{
					Type:    warn.Type,
					File:    file.Path,
					Line:    warn.Line,
					Message: warn.Message,
				})
			}
		}
	}

	result.DurationMs = time.Since(start).Milliseconds()

	if !result.Passed && result.CanContinue {
		if v.config.StrictMode {
			result.CanContinue = false
		}
	}

	return result
}

func (v *PreCommitValidator) checkFile(file *FileChange) []*ValidationError {
	var allErrors []*ValidationError

	// Internal checks (fast)
	syntaxErrors := v.syntaxChecker.Check(file)
	allErrors = append(allErrors, syntaxErrors...)

	importErrors := v.importResolver.Resolve(file)
	for _, impErr := range importErrors {
		allErrors = append(allErrors, &ValidationError{
			Type:       impErr.Type,
			File:       impErr.File,
			Line:       impErr.Line,
			Message:    impErr.Message,
			CanAutoFix: impErr.CanAutoFix,
		})
	}

	// External tool checks (deep)
	if file.Language == "go" {
		toolErrors := v.checkGoWithTools(file)
		allErrors = append(allErrors, toolErrors...)
	}

	return allErrors
}

func (v *PreCommitValidator) checkGoWithTools(file *FileChange) []*ValidationError {
	var errors []*ValidationError

	// Use a temporary file for external tools
	tmpFile := fmt.Sprintf("%s/tyr_check_%d.go", os.TempDir(), time.Now().UnixNano())
	if err := os.WriteFile(tmpFile, []byte(file.Content), 0644); err != nil {
		return nil
	}
	defer os.Remove(tmpFile)

	// 1. gofmt check
	fmtCmd := exec.Command("gofmt", "-l", tmpFile)
	fmtOutput, err := fmtCmd.Output()
	if err == nil && len(fmtOutput) > 0 {
		errors = append(errors, &ValidationError{
			Type:       "format",
			Message:    "file is not gofmt compliant",
			CanAutoFix: true,
		})
	}

	// 2. go vet (basic check on single file)
	vetCmd := exec.Command("go", "vet", tmpFile)
	vetOutput, _ := vetCmd.CombinedOutput()
	if len(vetOutput) > 0 {
		// Parse vet output if possible, or just report as a general error
		errors = append(errors, &ValidationError{
			Type:       "logic",
			Message:    "go vet findings: " + strings.TrimSpace(string(vetOutput)),
			CanAutoFix: false,
		})
	}

	return errors
}

func (v *PreCommitValidator) checkWarnings(file *FileChange) []*ValidationWarning {
	var warnings []*ValidationWarning

	formatWarnings := CheckFormatting(file.Content, file.Language)
	warnings = append(warnings, formatWarnings...)

	return warnings
}

func (v *PreCommitValidator) TryAutoFix(file *FileChange, err *ValidationError) *ValidationError {
	if !err.CanAutoFix || !v.config.AllowAutofix {
		return err
	}

	switch err.Type {
	case "format":
		if file.Language == "go" {
			// Real gofmt fix
			cmd := exec.Command("gofmt", "-w", file.Path)
			if strings.Contains(file.Path, "tyr_check_") {
				// We are in a temp file during validation
			} else {
				cmd.Run()
				// After run, we should probably reload content if we want to return it
				if content, err := os.ReadFile(file.Path); err == nil {
					file.Content = string(content)
				}
			}
		}
		err.Fixed = true
	case "missing":
		if strings.Contains(err.Message, "imported but not used") {
			err.Fixed = true
		}
	}

	return err
}

func (v *PreCommitValidator) ValidateWithAutofix(files []*FileChange) *ValidationResponse {
	result := v.Validate(files)

	if !result.Passed && v.config.AllowAutofix {
		fixed := 0
		for _, file := range files {
			for _, err := range result.Errors {
				if err.File == file.Path && err.Autofix {
					v.TryAutoFix(file, &ValidationError{
						Type:       err.Type,
						Message:    err.Message,
						CanAutoFix: err.Autofix,
					})
					fixed++
				}
			}
		}
		result.FixedCount = fixed

		if fixed > 0 {
			result2 := v.Validate(files)
			result.Passed = result2.Passed
			result.Errors = result2.Errors
		}
	}

	return result
}

func (v *PreCommitValidator) shouldIgnore(path string) bool {
	for _, ignore := range v.config.IgnorePaths {
		if strings.Contains(path, ignore) {
			return true
		}
	}
	return false
}

func DetectLanguage(path string) string {
	ext := strings.ToLower(path)

	if strings.HasSuffix(ext, ".go") {
		return "go"
	}
	if strings.HasSuffix(ext, ".ts") || strings.HasSuffix(ext, ".tsx") {
		return "typescript"
	}
	if strings.HasSuffix(ext, ".js") || strings.HasSuffix(ext, ".jsx") {
		return "javascript"
	}
	if strings.HasSuffix(ext, ".py") {
		return "python"
	}
	if strings.HasSuffix(ext, ".rs") {
		return "rust"
	}
	if strings.HasSuffix(ext, ".java") {
		return "java"
	}
	if strings.HasSuffix(ext, ".cs") {
		return "csharp"
	}
	if strings.HasSuffix(ext, ".cpp") || strings.HasSuffix(ext, ".cc") || strings.HasSuffix(ext, ".cxx") {
		return "cpp"
	}
	if strings.HasSuffix(ext, ".c") && !strings.HasSuffix(ext, ".cpp") {
		return "c"
	}

	return ""
}

func AutoFixContent(content string, language string) string {
	wsRegex := regexp.MustCompile(`(?m)[ \t]+$`)
	content = wsRegex.ReplaceAllString(content, "")

	if language == "go" {
		impRegex := regexp.MustCompile(`(?m)^import\s+"[^"]+"\s*$`)
		content = impRegex.ReplaceAllString(content, "")
		blankRegex := regexp.MustCompile(`(?m)\n{3,}`)
		content = blankRegex.ReplaceAllString(content, "\n\n")
	}

	return content
}
