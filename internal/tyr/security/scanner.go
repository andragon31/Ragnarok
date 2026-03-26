package security

import (
	"regexp"
	"strings"
)

type ScanResult struct {
	HasFindings bool       `json:"has_findings"`
	Findings    []*Finding `json:"findings"`
	Safe        bool       `json:"safe"`
}

type Finding struct {
	Type     string `json:"type"`
	Severity string `json:"severity"`
	Message  string `json:"message"`
	Match    string `json:"match,omitempty"`
}

var (
	injectionPatterns = []struct {
		Pattern *regexp.Regexp
		Message string
		Type    string
	}{
		{
			Pattern: regexp.MustCompile(`(?i)<\s*script[^>]*>`),
			Message: "Potential script injection tag",
			Type:    "xss",
		},
		{
			Pattern: regexp.MustCompile(`(?i)javascript:`),
			Message: "Potential javascript: protocol injection",
			Type:    "xss",
		},
		{
			Pattern: regexp.MustCompile(`(?i)on\w+\s*=\s*["']?\s*[^"'\s]+`),
			Message: "Potential event handler injection",
			Type:    "xss",
		},
		{
			Pattern: regexp.MustCompile(`(?i)<iframe[^>]*>`),
			Message: "Potential iframe injection",
			Type:    "xss",
		},
		{
			Pattern: regexp.MustCompile(`(?i)<img[^>]+onerror`),
			Message: "Potential onerror injection",
			Type:    "xss",
		},
		{
			Pattern: regexp.MustCompile(`\{\{.*?\}\}`),
			Message: "Potential template injection",
			Type:    "ssti",
		},
		{
			Pattern: regexp.MustCompile(`\$\{.*?\}`),
			Message: "Potential template injection",
			Type:    "ssti",
		},
		{
			Pattern: regexp.MustCompile(`(?i)<%.*?%>`),
			Message: "Potential server-side template injection",
			Type:    "ssti",
		},
		{
			Pattern: regexp.MustCompile(`(?i)\[\[.*?\]\]`),
			Message: "Potential Angular template injection",
			Type:    "ssti",
		},
	}

	secretPatterns = []struct {
		Pattern *regexp.Regexp
		Message string
		Type    string
	}{
		{
			Pattern: regexp.MustCompile(`(?i)api[_-]?key\s*[=:]\s*['"][a-zA-Z0-9]{20,}['"]`),
			Message: "Potential API key hardcoded",
			Type:    "secret",
		},
		{
			Pattern: regexp.MustCompile(`(?i)password\s*[=:]\s*['"][^'"\s]{8,}['"]`),
			Message: "Potential password hardcoded",
			Type:    "secret",
		},
		{
			Pattern: regexp.MustCompile(`(?i)secret\s*[=:]\s*['"][a-zA-Z0-9+/=]{20,}['"]`),
			Message: "Potential secret hardcoded",
			Type:    "secret",
		},
		{
			Pattern: regexp.MustCompile(`(?i)token\s*[=:]\s*['"][a-zA-Z0-9+/=_\.-]{20,}['"]`),
			Message: "Potential token hardcoded",
			Type:    "secret",
		},
		{
			Pattern: regexp.MustCompile(`-----BEGIN\s+(RSA\s+)?PRIVATE\s+KEY-----`),
			Message: "Private key detected in content",
			Type:    "secret",
		},
		{
			Pattern: regexp.MustCompile(`(?i)aws[_-]?access[_-]?key[_-]?id\s*[=:]\s*['"][A-Z0-9]{20}['"]`),
			Message: "Potential AWS access key hardcoded",
			Type:    "secret",
		},
		{
			Pattern: regexp.MustCompile(`(?i)ghp_[a-zA-Z0-9]{36}`),
			Message: "Potential GitHub personal access token",
			Type:    "secret",
		},
		{
			Pattern: regexp.MustCompile(`xox[baprs]-[a-zA-Z0-9]{10,}`),
			Message: "Potential Slack token",
			Type:    "secret",
		},
	}

	pathTraversalPatterns = []struct {
		Pattern *regexp.Regexp
		Message string
		Type    string
	}{
		{
			Pattern: regexp.MustCompile(`\.\.[/\\]`),
			Message: "Potential path traversal sequence",
			Type:    "path-traversal",
		},
		{
			Pattern: regexp.MustCompile(`(?i)\.\.%2f`),
			Message: "Potential URL-encoded path traversal",
			Type:    "path-traversal",
		},
		{
			Pattern: regexp.MustCompile(`(?i)%2e%2e`),
			Message: "Potential double-encoded path traversal",
			Type:    "path-traversal",
		},
	}
)

func ScanContent(content string) *ScanResult {
	result := &ScanResult{
		Findings: []*Finding{},
		Safe:     true,
	}

	contentLower := strings.ToLower(content)

	for _, p := range injectionPatterns {
		matches := p.Pattern.FindAllString(content, -1)
		for _, match := range matches {
			result.Findings = append(result.Findings, &Finding{
				Type:     p.Type,
				Severity: "high",
				Message:  p.Message,
				Match:    match,
			})
		}
	}

	for _, p := range secretPatterns {
		matches := p.Pattern.FindAllString(content, -1)
		for _, match := range matches {
			result.Findings = append(result.Findings, &Finding{
				Type:     p.Type,
				Severity: "critical",
				Message:  p.Message,
				Match:    maskSecret(match),
			})
		}
	}

	for _, p := range pathTraversalPatterns {
		matches := p.Pattern.FindAllString(contentLower, -1)
		for _, match := range matches {
			result.Findings = append(result.Findings, &Finding{
				Type:     p.Type,
				Severity: "medium",
				Message:  p.Message,
				Match:    match,
			})
		}
	}

	if len(result.Findings) > 0 {
		result.Safe = false
		result.HasFindings = true
	}

	return result
}

func ScanFile(path string, content string) *ScanResult {
	return ScanContent(content)
}

func ScanDir(paths []string) *ScanResult {
	result := &ScanResult{
		Findings: []*Finding{},
		Safe:     true,
	}

	for _, p := range paths {
		scanResult := ScanContent(p)
		result.Findings = append(result.Findings, scanResult.Findings...)
	}

	if len(result.Findings) > 0 {
		result.Safe = false
		result.HasFindings = true
	}

	return result
}

func DetectInjection(content string) (bool, []*Finding) {
	result := ScanContent(content)
	return !result.Safe, result.Findings
}

func maskSecret(secret string) string {
	if len(secret) <= 8 {
		return "****"
	}
	return secret[:4] + "****" + secret[len(secret)-4:]
}
