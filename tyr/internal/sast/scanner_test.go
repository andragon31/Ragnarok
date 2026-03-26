package sast

import (
	"testing"
)

func TestScanner_ScanFile_HardcodedSecret(t *testing.T) {
	scanner := NewScanner()

	tests := []struct {
		name     string
		content  string
		expected int
	}{
		{
			name:     "No secrets",
			content:  "package main\n\nfunc main() {}",
			expected: 0,
		},
		{
			name:     "Password detected",
			content:  `password := "PLACEHOLDER_VALUE"`,
			expected: 1,
		},
		{
			name:     "AWS access key detected",
			content:  `awsAccessKey := "AKIATESTEXAMPLE123"`,
			expected: 1,
		},
		{
			name:     "Bearer token detected",
			content:  `Authorization: Bearer SAMPLE_TOKEN_DATA`,
			expected: 1,
		},
		{
			name:     "Multiple secrets on same line",
			content:  `password := "PLACEHOLDER123"`,
			expected: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			findings := scanner.ScanFile("test.go", tt.content)
			if len(findings) != tt.expected {
				t.Errorf("expected %d findings, got %d", tt.expected, len(findings))
			}
		})
	}
}

func TestScanner_ScanFile_SQLInjection(t *testing.T) {
	scanner := NewScanner()

	tests := []struct {
		name     string
		content  string
		expected int
	}{
		{
			name:     "Safe query with parameter",
			content:  `query := "SELECT * FROM users WHERE id = ?", userID`,
			expected: 0,
		},
		{
			name:     "SELECT with concatenation",
			content:  `query := "SELECT * FROM users WHERE id = " + userID`,
			expected: 1,
		},
		{
			name:     "exec with string concatenation",
			content:  `exec("SELECT * FROM users WHERE name = '" + name + "'")`,
			expected: 1,
		},
		{
			name:     "INSERT with plus",
			content:  `INSERT INTO logs VALUES ('" + userInput + "')`,
			expected: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			findings := scanner.ScanFile("test.py", tt.content)
			if len(findings) != tt.expected {
				t.Errorf("expected %d findings, got %d", tt.expected, len(findings))
			}
		})
	}
}

func TestScanner_ScanFile_CommandInjection(t *testing.T) {
	scanner := NewScanner()

	tests := []struct {
		name           string
		content        string
		expectFindings bool
	}{
		{
			name:           "Safe system call",
			content:        `system("ls -la")`,
			expectFindings: false,
		},
		{
			name:           "subprocess call",
			content:        `subprocess.call(args)`,
			expectFindings: true,
		},
		{
			name:           "subprocess run",
			content:        `subprocess.run(args)`,
			expectFindings: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			findings := scanner.ScanFile("test.py", tt.content)
			hasFindings := len(findings) > 0
			if hasFindings != tt.expectFindings {
				t.Errorf("expected findings=%v, got %d findings", tt.expectFindings, len(findings))
			}
		})
	}
}

func TestScanner_ScanFile_XSS(t *testing.T) {
	scanner := NewScanner()

	tests := []struct {
		name           string
		content        string
		expectFindings bool
	}{
		{
			name:           "Safe textContent",
			content:        `element.textContent = userData`,
			expectFindings: false,
		},
		{
			name:           "innerHTML assignment",
			content:        `element.innerHTML = "<div>test</div>"`,
			expectFindings: false,
		},
		{
			name:           "dangerouslySetInnerHTML React",
			content:        `dangerouslySetInnerHTML={{__html: userContent}}`,
			expectFindings: true,
		},
		{
			name:           "v-html Vue directive",
			content:        `v-html="userContent"`,
			expectFindings: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			findings := scanner.ScanFile("test.js", tt.content)
			hasFindings := len(findings) > 0
			if hasFindings != tt.expectFindings {
				t.Errorf("expected findings=%v, got %d findings", tt.expectFindings, len(findings))
			}
		})
	}
}

func TestScanner_ScanFile_WeakCrypto(t *testing.T) {
	scanner := NewScanner()

	tests := []struct {
		name           string
		content        string
		expectFindings bool
	}{
		{
			name:           "Safe hashing with sha256",
			content:        `hashlib.sha256(data)`,
			expectFindings: false,
		},
		{
			name:           "MD5 usage",
			content:        `md5(password)`,
			expectFindings: true,
		},
		{
			name:           "SHA1 usage",
			content:        `sha1(data)`,
			expectFindings: true,
		},
		{
			name:           "RC4 cipher",
			content:        `Crypto.createCipher('rc4', key)`,
			expectFindings: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			findings := scanner.ScanFile("test.py", tt.content)
			hasFindings := len(findings) > 0
			if hasFindings != tt.expectFindings {
				t.Errorf("expected findings=%v, got %d findings", tt.expectFindings, len(findings))
			}
		})
	}
}

func TestScanner_ScanFile_PathTraversal(t *testing.T) {
	scanner := NewScanner()

	tests := []struct {
		name           string
		content        string
		expectFindings bool
	}{
		{
			name:           "Safe file open",
			content:        `open("/safe/path/file.txt")`,
			expectFindings: false,
		},
		{
			name:           "Path traversal sequence",
			content:        `open(userPath + "/../../etc/passwd")`,
			expectFindings: true,
		},
		{
			name:           "open with format string",
			content:        `open("/path/%s" % userInput)`,
			expectFindings: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			findings := scanner.ScanFile("test.py", tt.content)
			hasFindings := len(findings) > 0
			if hasFindings != tt.expectFindings {
				t.Errorf("expected findings=%v, got %d findings", tt.expectFindings, len(findings))
			}
		})
	}
}

func TestIsScannableExt(t *testing.T) {
	tests := []struct {
		ext      string
		expected bool
	}{
		{".go", true},
		{".js", true},
		{".ts", true},
		{".py", true},
		{".java", true},
		{".cs", true},
		{".rb", true},
		{".rs", true},
		{".php", true},
		{".swift", true},
		{".kt", true},
		{".jpg", false},
		{".png", false},
		{".mp4", false},
		{".exe", false},
		{".txt", false},
		{".md", false},
	}

	for _, tt := range tests {
		t.Run(tt.ext, func(t *testing.T) {
			result := isScannableExt(tt.ext)
			if result != tt.expected {
				t.Errorf("isScannableExt(%s) = %v, expected %v", tt.ext, result, tt.expected)
			}
		})
	}
}

func TestDetectLangByExt(t *testing.T) {
	tests := []struct {
		ext      string
		expected string
	}{
		{".go", "go"},
		{".js", "javascript"},
		{".ts", "typescript"},
		{".py", "python"},
		{".java", "java"},
		{".cs", "csharp"},
		{".rb", "ruby"},
		{".rs", "rust"},
		{".kt", "kotlin"},
		{".swift", "swift"},
		{".scala", "scala"},
		{".c", "c"},
		{".cpp", "cpp"},
		{".vue", "vue"},
		{".svelte", "svelte"},
		{".unknown", ""},
	}

	for _, tt := range tests {
		t.Run(tt.ext, func(t *testing.T) {
			result := detectLangByExt(tt.ext)
			if result != tt.expected {
				t.Errorf("detectLangByExt(%s) = %s, expected %s", tt.ext, result, tt.expected)
			}
		})
	}
}

func TestFinding_ToMap(t *testing.T) {
	finding := &Finding{
		ID:       "test-1",
		RuleID:   "hardcoded-secret",
		RuleName: "Hardcoded Secret",
		Severity: "critical",
		Type:     "hardcoded-secret",
		File:     "config.go",
		Line:     10,
		Column:   5,
		Match:    "password123",
		Message:  "Hardcoded secret detected",
		Status:   "open",
	}

	result := finding.ToMap()

	if result["id"] != "test-1" {
		t.Errorf("expected id=test-1, got %v", result["id"])
	}
	if result["rule_id"] != "hardcoded-secret" {
		t.Errorf("expected rule_id=hardcoded-secret, got %v", result["rule_id"])
	}
	if result["file"] != "config.go" {
		t.Errorf("expected file=config.go, got %v", result["file"])
	}
	if result["line"] != 10 {
		t.Errorf("expected line=10, got %v", result["line"])
	}
}

func TestScanner_ToJSON(t *testing.T) {
	scanner := NewScanner()
	findings := []*Finding{
		{
			ID:       "test-1",
			RuleID:   "hardcoded-secret",
			RuleName: "Hardcoded Secret",
			Severity: "critical",
			Type:     "hardcoded-secret",
			File:     "config.go",
			Line:     10,
			Column:   5,
			Match:    "secret",
			Message:  "Hardcoded secret detected",
			Status:   "open",
		},
	}

	json := scanner.ToJSON(findings)
	if json == "" {
		t.Error("expected non-empty JSON output")
	}
}

func BenchmarkScanFile(b *testing.B) {
	scanner := NewScanner()
	content := `
	package main
	
	import "fmt"
	
	func main() {
		testValue := "PLACEHOLDER"
		secretValue := "PLACEHOLDER"
		fmt.Println("Hello")
	}
	`

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		scanner.ScanFile("test.go", content)
	}
}
