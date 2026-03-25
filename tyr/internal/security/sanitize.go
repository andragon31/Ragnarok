package security

import (
	"regexp"
	"strings"
)

type SanitizeResult struct {
	Content    string   `json:"content"`
	Redacted   int      `json:"redacted_count"`
	Redactions []string `json:"redactions"`
}

func Sanitize(content string) *SanitizeResult {
	result := &SanitizeResult{
		Content:    content,
		Redactions: []string{},
	}

	result.Content = sanitizeSecrets(result.Content, result)
	result.Content = sanitizeEmails(result.Content, result)
	result.Content = sanitizeIPs(result.Content, result)
	result.Content = sanitizeCreditCards(result.Content, result)
	result.Content = sanitizePhoneNumbers(result.Content, result)

	return result
}

func sanitizeSecrets(content string, result *SanitizeResult) string {
	patterns := []*regexp.Regexp{
		regexp.MustCompile(`(?i)(api[_-]?key\s*[=:]\s*)['"]([a-zA-Z0-9]{20,})['"]`),
		regexp.MustCompile(`(?i)(password\s*[=:]\s*)['"]([^'"\s]{8,})['"]`),
		regexp.MustCompile(`(?i)(secret\s*[=:]\s*)['"]([a-zA-Z0-9+/=]{20,})['"]`),
		regexp.MustCompile(`(?i)(token\s*[=:]\s*)['"]([a-zA-Z0-9+/=_\.-]{20,})['"]`),
		regexp.MustCompile(`(?i)(aws[_-]?access[_-]?key[_-]?id\s*[=:]\s*)['"]([A-Z0-9]{20})['"]`),
		regexp.MustCompile(`ghp_[a-zA-Z0-9]{36}`),
		regexp.MustCompile(`xox[baprs]-[a-zA-Z0-9]{10,}`),
	}

	for _, p := range patterns {
		content = p.ReplaceAllStringFunc(content, func(match string) string {
			if len(match) <= 8 {
				result.Redactions = append(result.Redactions, "secret")
				result.Redacted++
				return "[REDACTED]"
			}
			result.Redactions = append(result.Redactions, "secret")
			result.Redacted++
			return match[:4] + "[REDACTED]" + match[len(match)-4:]
		})
	}

	return content
}

func sanitizeEmails(content string, result *SanitizeResult) string {
	pattern := regexp.MustCompile(`[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}`)
	return pattern.ReplaceAllStringFunc(content, func(match string) string {
		parts := strings.Split(match, "@")
		if len(parts) == 2 {
			domain := parts[1]
			local := parts[0]
			if len(local) > 2 {
				local = local[:2] + "***"
			}
			result.Redactions = append(result.Redactions, "email")
			result.Redacted++
			return local + "@" + domain
		}
		return match
	})
}

func sanitizeIPs(content string, result *SanitizeResult) string {
	ipv4 := regexp.MustCompile(`\b\d{1,3}\.\d{1,3}\.\d{1,3}\.\d{1,3}\b`)
	content = ipv4.ReplaceAllStringFunc(content, func(match string) string {
		result.Redactions = append(result.Redactions, "ip")
		result.Redacted++
		return "[REDACTED-IP]"
	})

	ipv6 := regexp.MustCompile(`([0-9a-fA-F]{1,4}:){7}[0-9a-fA-F]{1,4}`)
	content = ipv6.ReplaceAllStringFunc(content, func(match string) string {
		result.Redactions = append(result.Redactions, "ip")
		result.Redacted++
		return "[REDACTED-IPV6]"
	})

	return content
}

func sanitizeCreditCards(content string, result *SanitizeResult) string {
	patterns := []*regexp.Regexp{
		regexp.MustCompile(`\b\d{4}[\s-]?\d{4}[\s-]?\d{4}[\s-]?\d{4}\b`),
		regexp.MustCompile(`\b\d{15,16}\b`),
	}

	for _, p := range patterns {
		content = p.ReplaceAllStringFunc(content, func(match string) string {
			digits := regexp.MustCompile(`\d`).FindAllString(match, -1)
			if len(digits) >= 4 {
				result.Redactions = append(result.Redactions, "credit_card")
				result.Redacted++
				return "****-****-****-" + strings.Join(digits[len(digits)-4:], "")
			}
			return match
		})
	}

	return content
}

func sanitizePhoneNumbers(content string, result *SanitizeResult) string {
	patterns := []*regexp.Regexp{
		regexp.MustCompile(`\+?\d{1,3}[\s.-]?\(?\d{3}\)?[\s.-]?\d{3}[\s.-]?\d{4}`),
		regexp.MustCompile(`\(\d{3}\)\s*\d{3}[-.]?\d{4}`),
	}

	for _, p := range patterns {
		content = p.ReplaceAllStringFunc(content, func(match string) string {
			digits := regexp.MustCompile(`\d`).FindAllString(match, -1)
			if len(digits) >= 4 {
				result.Redactions = append(result.Redactions, "phone")
				result.Redacted++
				return "(***) ***-" + strings.Join(digits[len(digits)-4:], "")
			}
			return match
		})
	}

	return content
}

func SanitizeHTML(html string) string {
	dangerousTags := []string{
		"<script",
		"</script",
		"<iframe",
		"</iframe",
		"javascript:",
		"onerror=",
		"onclick=",
		"onload=",
	}

	for _, tag := range dangerousTags {
		html = strings.ReplaceAll(strings.ToLower(html), tag, "&lt;"+tag)
	}

	return html
}

func RemovePrivateTags(content string) string {
	privateTags := []string{
		`<!--.*?-->`,
		`/\*.*?\*/`,
	}

	for _, tag := range privateTags {
		re := regexp.MustCompile(tag)
		content = re.ReplaceAllString(content, "")
	}

	return content
}
