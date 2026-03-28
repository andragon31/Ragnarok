package sast

import (
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"
)

type Rule struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Severity    string   `json:"severity"`
	Patterns    []string `json:"patterns"`
	Description string   `json:"description"`
	Language    string   `json:"language,omitempty"`
	CWE         string   `json:"cwe,omitempty"`
}

type Finding struct {
	ID        string    `json:"id"`
	RuleID    string    `json:"rule_id"`
	RuleName  string    `json:"rule_name"`
	Severity  string    `json:"severity"`
	Type      string    `json:"type"`
	File      string    `json:"file"`
	Line      int       `json:"line"`
	Column    int       `json:"column,omitempty"`
	Match     string    `json:"match,omitempty"`
	Message   string    `json:"message"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
}

type Scanner struct {
	Rules []Rule
}

func NewScanner() *Scanner {
	s := &Scanner{
		Rules: getDefaultRules(),
	}
	return s
}

func getDefaultRules() []Rule {
	return []Rule{
		{
			ID:          "hardcoded-secret",
			Name:        "Hardcoded Secret Detected",
			Severity:    "critical",
			Description: "Hardcoded credentials, API keys, or passwords detected",
			CWE:         "CWE-798",
			Patterns: []string{
				`(?i)api[_-]?key['":\s=]+['"][a-zA-Z0-9]{20,}['"]`,
				`(?i)password['":\s=]+['"][^'"\s]{8,}['"]`,
				`(?i)secret['":\s=]+['"][a-zA-Z0-9+/=]{20,}['"]`,
				`(?i)token['":\s=]+['"][a-zA-Z0-9+/=]{20,}['"]`,
				`(?i)bearer['":\s=]+[a-zA-Z0-9+/=_.=-]+`,
				`(?i)aws[_-]?access[_-]?key['":\s=]+[A-Z0-9]{20,}`,
				`(?i)private[_-]?key['":\s=]+['"]-----BEGIN`,
			},
		},
		{
			ID:          "sql-injection",
			Name:        "Potential SQL Injection",
			Severity:    "critical",
			Description: "SQL query built from user input without proper sanitization",
			CWE:         "CWE-89",
			Patterns: []string{
				`(?i)execute\s*\(\s*["'].*?\+.*?["']`,
				`(?i)query\s*\(\s*["'].*?\+.*?["']`,
				`(?i)exec\s*\(\s*["'].*?\%s.*?["']`,
				`(?i)sql\s*=\s*["'].*?\+`,
				`(?i)SELECT.*?FROM.*?\+`,
				`(?i)INSERT.*?INTO.*?\+`,
				`(?i)UPDATE.*?SET.*?\+`,
				`(?i)DELETE.*?FROM.*?\+`,
			},
		},
		{
			ID:          "command-injection",
			Name:        "Potential Command Injection",
			Severity:    "critical",
			Description: "System command executed with user-controlled input",
			CWE:         "CWE-78",
			Patterns: []string{
				`(?i)exec\s*\(\s*.*?\$`,
				`(?i)system\s*\(\s*.*?\$`,
				`(?i)shell_exec\s*\(\s*.*?\$`,
				`(?i)eval\s*\(\s*.*?\$`,
				`(?i)exec\s*\(\s*.*input`,
				`os\.system\s*\(.*?\)`,
				`os\.popen\s*\(.*?\)`,
				`subprocess\.call\s*\(.*?\)`,
				`subprocess\.run\s*\(.*?\)`,
			},
		},
		{
			ID:          "path-traversal",
			Name:        "Potential Path Traversal",
			Severity:    "high",
			Description: "File path constructed from user input without validation",
			CWE:         "CWE-22",
			Patterns: []string{
				`\.\./`,
				`\.\.\\`,
				`(?i)open\s*\(\s*.*?\%`,
				`(?i)readFile\s*\(\s*.*?\+`,
				`(?i)include\s*\(\s*.*?\$`,
				`(?i)require\s*\(\s*.*?\$`,
				`filepath\.Join.*?\.`,
				`path\.join.*?\.`,
			},
		},
		{
			ID:          "xss",
			Name:        "Potential Cross-Site Scripting (XSS)",
			Severity:    "high",
			Description: "User input rendered without proper escaping",
			CWE:         "CWE-79",
			Patterns: []string{
				`(?i)innerHTML\s*=.*?\$`,
				`(?i)outerHTML\s*=.*?\$`,
				`(?i)document\.write\s*\(.*?\$`,
				`(?i)\.html\s*\(.*?\)`,
				`(?i)dangerouslySetInnerHTML`,
				`(?i)v-html\s*=`,
				`(?i)render\s*\(\s*.*?\$`,
				`\{[^}]*\}\s*\+`,
				`(?i)echo.*?\$`,
			},
		},
		{
			ID:          "unsafe-deserialization",
			Name:        "Unsafe Deserialization",
			Severity:    "critical",
			Description: "Data deserialized from untrusted source",
			CWE:         "CWE-502",
			Patterns: []string{
				`(?i)pickle\.loads`,
				`(?i)yaml\.load\s*\([^,]*\)`,
				`(?i)json\.decode\s*\([^,]*\)`,
				`(?i)unserialize\s*\(`,
				`(?i)ObjectInputStream`,
				`(?i)readObject\s*\(\)`,
				`(?i)marshal\.Load`,
				`(?i)serde::de`,
			},
		},
		{
			ID:          "xxe",
			Name:        "XML External Entity (XXE)",
			Severity:    "critical",
			Description: "XML parsed with external entity resolution enabled",
			CWE:         "CWE-611",
			Patterns: []string{
				`(?i)DocumentBuilderFactory`,
				`(?i)SAXParserFactory`,
				`(?i)XMLInputFactory`,
				`(?i)setFeature\s*\(\s*["']http://apache.org/xml/features/`,
				`(?i)setProperty\s*\(\s*["']ENTITY`,
				`(?i)libxml_set_streams_context`,
				`(?i)simplexml_load_string.*?LIBXML_NOENT`,
			},
		},
		{
			ID:          "weak-crypto",
			Name:        "Weak Cryptographic Algorithm",
			Severity:    "medium",
			Description: "Weak cryptographic algorithm in use",
			CWE:         "CWE-327",
			Patterns: []string{
				`(?i)md5\s*\(`,
				`(?i)sha1\s*\(`,
				`(?i)des\s*\(`,
				`(?i)rc4\s*\(`,
				`(?i)Crypto\.createCipher\s*\(`,
				`(?i)hashlib\.md5`,
				`(?i)hashlib\.sha1`,
				`(?i)OpenSSL::Cipher::Cipher\.new\s*\(["']rc4`,
				`(?i)AES\.new\s*\([^,]*\s*,\s*["']ecb`,
			},
		},
		{
			ID:          "insecure-cookie",
			Name:        "Insecure Cookie Configuration",
			Severity:    "medium",
			Description: "Cookie without secure or httpOnly flag",
			CWE:         "CWE-614",
			Patterns: []string{
				`(?i)Cookie:\s*[^;]*;\s*[^H][^;]*`,
				`(?i)setcookie\s*\([^,]*,\s*[^,]*,\s*0`,
				`(?i)Secure\s*=\s*false`,
				`(?i)HttpOnly\s*=\s*false`,
				`(?i)cookie\s*=\s*[^;]*$`,
				`(?i)\.Cookie\s*\(`,
			},
		},
		{
			ID:          "csrf-missing",
			Name:        "Missing CSRF Protection",
			Severity:    "high",
			Description: "POST/PUT/DELETE without CSRF token verification",
			CWE:         "CWE-352",
			Patterns: []string{
				`(?i)app\.(post|put|delete|patch)\s*\(`,
				`(?i)@app\.route\(.*method.*POST`,
				`(?i)router\.(post|put|delete|patch)`,
				`(?i)@PostMapping|@PutMapping|@DeleteMapping`,
				`(?i)Route::(post|put|delete|patch)`,
			},
		},
		{
			ID:          "log-injection",
			Name:        "Log Injection",
			Severity:    "medium",
			Description: "User input logged without sanitization",
			CWE:         "CWE-117",
			Patterns: []string{
				`(?i)console\.log\s*\(.*?\)`,
				`(?i)logger\.info\s*\(.*?\)`,
				`(?i)logger\.debug\s*\(.*?\)`,
				`(?i)logging\.info\s*\(.*?\)`,
				`(?i)System\.out\.print\s*\(.*?\)`,
				`(?i)fmt\.Print\s*\(.*?\)`,
				`(?i)print\s*\(.*?\)`,
				`(?i)log\.Printf`,
				`(?i)log\.Println`,
			},
		},
		{
			ID:          "banner-detection",
			Name:        "Information Exposure Through Server Banner",
			Severity:    "low",
			Description: "Server version or technology information exposed",
			CWE:         "CWE-200",
			Patterns: []string{
				`(?i)Server:\s*[^\s]+`,
				`(?i)X-Powered-By:\s*[^\s]+`,
				`(?i)X-AspNet-Version:`,
				`(?i)Apache/\d+\.\d+`,
				`(?i)nginx/\d+\.\d+`,
				`(?i)Microsoft-IIS`,
			},
		},
		{
			ID:          "https-missing",
			Name:        "Missing HTTPS/SSL Configuration",
			Severity:    "high",
			Description: "Secure connection not enforced",
			CWE:         "CWE-295",
			Patterns: []string{
				`(?i)ssl_verify_mode\s*=\s*OpenSSL::SSL::VERIFY_NONE`,
				`(?i)\.check_server_certificate\s*=\s*false`,
				`(?i)InsecureRequestWarning`,
				`(?i)SSL_context\s*=\s*ssl\.create_default_context\s*\(\s*\)`,
				`(?i)verify=False`,
				`(?i)Skip-SSL`,
			},
		},
	}
}

func (s *Scanner) ScanFile(path string, content string) []*Finding {
	var findings []*Finding
	ext := filepath.Ext(path)
	lang := detectLangByExt(ext)

	for _, rule := range s.Rules {
		if rule.Language != "" && rule.Language != lang {
			continue
		}

		for lineNum, line := range strings.Split(content, "\n") {
			for _, pattern := range rule.Patterns {
				re := regexp.MustCompile(pattern)
				matches := re.FindAllStringIndex(line, -1)
				for _, match := range matches {
					finding := &Finding{
						ID:        generateID(" finding"),
						RuleID:    rule.ID,
						RuleName:  rule.Name,
						Severity:  rule.Severity,
						Type:      rule.ID,
						File:      path,
						Line:      lineNum + 1,
						Column:    match[0] + 1,
						Match:     line[match[0]:match[1]],
						Message:   rule.Description,
						Status:    "open",
						CreatedAt: time.Now(),
					}
					findings = append(findings, finding)
				}
			}
		}
	}

	return findings
}

func (s *Scanner) ScanDir(rootPath string, ignorePaths []string) ([]*Finding, error) {
	var findings []*Finding

	ignoreMap := make(map[string]bool)
	for _, p := range ignorePaths {
		ignoreMap[p] = true
	}

	err := filepath.Walk(rootPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}

		for _, ignore := range ignorePaths {
			if strings.Contains(path, ignore) {
				if info.IsDir() {
					return filepath.SkipDir
				}
				return nil
			}
		}

		if info.IsDir() {
			return nil
		}

		ext := filepath.Ext(path)
		if !isScannableExt(ext) {
			return nil
		}

		content, err := os.ReadFile(path)
		if err != nil {
			return nil
		}

		fileFindings := s.ScanFile(path, string(content))
		findings = append(findings, fileFindings...)

		return nil
	})

	return findings, err
}

func isScannableExt(ext string) bool {
	scannable := map[string]bool{
		".go":     true,
		".js":     true,
		".ts":     true,
		".jsx":    true,
		".tsx":    true,
		".py":     true,
		".java":   true,
		".cs":     true,
		".php":    true,
		".rb":     true,
		".swift":  true,
		".kt":     true,
		".scala":  true,
		".c":      true,
		".cpp":    true,
		".h":      true,
		".rs":     true,
		".vue":    true,
		".svelte": true,
	}
	return scannable[ext]
}

func detectLangByExt(ext string) string {
	langs := map[string]string{
		".go":     "go",
		".js":     "javascript",
		".ts":     "typescript",
		".jsx":    "javascript",
		".tsx":    "typescript",
		".py":     "python",
		".java":   "java",
		".cs":     "csharp",
		".php":    "php",
		".rb":     "ruby",
		".swift":  "swift",
		".kt":     "kotlin",
		".scala":  "scala",
		".c":      "c",
		".cpp":    "cpp",
		".rs":     "rust",
		".vue":    "vue",
		".svelte": "svelte",
	}
	return langs[ext]
}

func (f *Finding) ToMap() map[string]interface{} {
	return map[string]interface{}{
		"id":         f.ID,
		"rule_id":    f.RuleID,
		"rule_name":  f.RuleName,
		"severity":   f.Severity,
		"file":       f.File,
		"line":       f.Line,
		"column":     f.Column,
		"match":      f.Match,
		"message":    f.Message,
		"status":     f.Status,
		"created_at": f.CreatedAt,
	}
}

func (s *Scanner) ToJSON(findings []*Finding) string {
	data, _ := json.MarshalIndent(findings, "", "  ")
	return string(data)
}

var idCounter = 0
var idMutex sync.Mutex

func generateID(prefix string) string {
	idMutex.Lock()
	defer idMutex.Unlock()
	idCounter++
	return prefix + "_" + time.Now().Format("20060102150405") + "_" + string(rune('a'+idCounter%26))
}
