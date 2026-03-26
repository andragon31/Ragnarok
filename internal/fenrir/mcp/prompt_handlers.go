package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func (s *Server) handlePromptAnalyze(ctx context.Context, req *Request) (*Response, error) {
	var params struct {
		Prompt string `json:"prompt"`
		Module string `json:"module,omitempty"`
	}

	if err := json.Unmarshal(req.Params, &params); err != nil {
		return nil, fmt.Errorf("failed to parse params: %w", err)
	}

	if params.Prompt == "" {
		return nil, fmt.Errorf("prompt is required")
	}

	analysis := analyzePromptRelevance(params.Prompt, params.Module)

	return &Response{
		Result: map[string]interface{}{
			"prompt":           params.Prompt,
			"module":           params.Module,
			"is_relevant":      analysis.IsRelevant,
			"confidence":       analysis.Confidence,
			"reason":           analysis.Reason,
			"suggested_action": analysis.SuggestedAction,
			"keywords_found":   analysis.KeywordsFound,
			"keywords_missing": analysis.KeywordsMissing,
		},
	}, nil
}

type PromptAnalysis struct {
	IsRelevant      bool     `json:"is_relevant"`
	Confidence      float64  `json:"confidence"`
	Reason          string   `json:"reason"`
	SuggestedAction string   `json:"suggested_action"`
	KeywordsFound   []string `json:"keywords_found"`
	KeywordsMissing []string `json:"keywords_missing"`
}

var projectKeywords = map[string][]string{
	"go":         {"func", "package", "struct", "interface", "goroutine", "channel", "module", "import"},
	"typescript": {"interface", "type", "class", "async", "await", "promise", "npm", "import", "export"},
	"javascript": {"function", "const", "let", "var", "async", "await", "npm", "import", "export"},
	"python":     {"def", "class", "import", "from", "pip", "venv", "decorator", "async"},
	"react":      {"component", "jsx", "tsx", "useState", "useEffect", "props", "state"},
	"next":       {"page", "getServerSideProps", "getStaticProps", "api", "route"},
	"docker":     {"docker", "container", "image", "dockerfile", "compose", "volume"},
	"database":   {"query", "sql", "select", "insert", "update", "delete", "schema", "migration"},
	"api":        {"endpoint", "rest", "graphql", "http", "request", "response", "json"},
	"testing":    {"test", "spec", "mock", "assert", "expect", "coverage", "jest", "pytest"},
}

var outOfScopeKeywords = []string{
	"政治", "politics", "宗教", "religion",
	"crypto", "bitcoin", "mining",
	"gambling", "casino", "betting",
	"adult", "porn", "nsfw",
}

func analyzePromptRelevance(prompt, module string) *PromptAnalysis {
	result := &PromptAnalysis{
		IsRelevant:      true,
		Confidence:      0.5,
		Reason:          "neutral",
		SuggestedAction: "proceed",
		KeywordsFound:   []string{},
		KeywordsMissing: []string{},
	}

	promptLower := strings.ToLower(prompt)

	for _, keyword := range outOfScopeKeywords {
		if strings.Contains(promptLower, keyword) {
			result.IsRelevant = false
			result.Confidence = 0.95
			result.Reason = "prompt contains out-of-scope content: " + keyword
			result.SuggestedAction = "reject"
			return result
		}
	}

	detectedTech := []string{}
	for tech, keywords := range projectKeywords {
		foundCount := 0
		for _, kw := range keywords {
			if strings.Contains(promptLower, strings.ToLower(kw)) {
				foundCount++
			}
		}
		if foundCount >= 2 {
			detectedTech = append(detectedTech, tech)
			result.KeywordsFound = append(result.KeywordsFound, tech)
		}
	}

	if len(detectedTech) > 0 {
		result.IsRelevant = true
		result.Confidence = 0.8
		result.Reason = "detected technology-related keywords: " + strings.Join(detectedTech, ", ")
		result.SuggestedAction = "proceed"
		return result
	}

	actionWords := []string{"create", "add", "fix", "update", "delete", "remove", "implement", "refactor", "test", "build", "run", "debug", "deploy", "configure", "setup"}
	hasAction := false
	for _, action := range actionWords {
		if strings.Contains(promptLower, action) {
			hasAction = true
			break
		}
	}

	if hasAction {
		result.IsRelevant = true
		result.Confidence = 0.6
		result.Reason = "prompt contains development action words"
		result.SuggestedAction = "proceed_with_caution"
		return result
	}

	if len(prompt) < 20 {
		result.IsRelevant = false
		result.Confidence = 0.7
		result.Reason = "prompt too short to determine relevance"
		result.SuggestedAction = "request_clarification"
		return result
	}

	result.IsRelevant = true
	result.Confidence = 0.4
	result.Reason = "generic prompt, proceeding with standard workflow"
	result.SuggestedAction = "proceed"
	return result
}

func (s *Server) handleAgentsMdGet(ctx context.Context, req *Request) (*Response, error) {
	var params struct {
		ProjectPath string `json:"project_path,omitempty"`
	}

	if err := json.Unmarshal(req.Params, &params); err != nil {
		return nil, fmt.Errorf("failed to parse params: %w", err)
	}

	if params.ProjectPath == "" {
		home, _ := os.UserHomeDir()
		params.ProjectPath = home
	}

	agentsMdPath := filepath.Join(params.ProjectPath, "AGENTS.md")
	content, err := os.ReadFile(agentsMdPath)
	if err != nil {
		return &Response{
			Result: map[string]interface{}{
				"found":   false,
				"path":    agentsMdPath,
				"content": "",
				"note":    "AGENTS.md not found in project root",
			},
		}, nil
	}

	return &Response{
		Result: map[string]interface{}{
			"found":   true,
			"path":    agentsMdPath,
			"content": string(content),
			"size":    len(content),
		},
	}, nil
}
