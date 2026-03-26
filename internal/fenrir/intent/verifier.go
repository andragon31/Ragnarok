package intent

import (
	"regexp"
	"strings"
)

type Verifier struct {
	store *IntentStore
}

type VerificationResult struct {
	IntentID       string   `json:"intent_id"`
	PlanID         string   `json:"plan_id"`
	CoverageScore  float64  `json:"coverage_score"`
	AlignmentScore float64  `json:"alignment_score"`
	Covered        []string `json:"covered"`
	Missing        []string `json:"missing"`
	Partial        []string `json:"partial"`
	Suggestions    []string `json:"suggestions"`
	VerifiedAt     string   `json:"verified_at"`
}

type CodeAnalyzer struct{}

type CodeFeature struct {
	Type      string
	Name      string
	File      string
	Signature string
	Imports   []string
	Functions []string
}

func NewVerifier(store *IntentStore) *Verifier {
	return &Verifier{store: store}
}

func (v *Verifier) Verify(planID string, changedFiles []*FileInfo) (*VerificationResult, error) {
	intent, err := v.store.GetByPlanID(planID)
	if err != nil {
		return nil, err
	}

	codeFeatures := AnalyzeCodeFeatures(changedFiles)

	result := &VerificationResult{
		IntentID:    intent.ID,
		PlanID:      planID,
		Covered:     []string{},
		Missing:     []string{},
		Partial:     []string{},
		Suggestions: []string{},
	}

	for _, item := range intent.Items {
		matched := findBestMatch(item.Description, codeFeatures)

		if matched == nil {
			result.Missing = append(result.Missing, item.Description)
			result.Suggestions = append(result.Suggestions, "Consider implementing: "+item.Description)
			continue
		}

		if matched.Score >= 0.8 {
			result.Covered = append(result.Covered, item.Description)
			v.store.UpdateItemStatus(item.ID, "covered", matched.Score)
		} else if matched.Score >= 0.5 {
			result.Partial = append(result.Partial, item.Description)
			result.Suggestions = append(result.Suggestions, "Partially implemented: "+item.Description)
			v.store.UpdateItemStatus(item.ID, "partial", matched.Score)
		} else {
			result.Missing = append(result.Missing, item.Description)
			result.Suggestions = append(result.Suggestions, "Missing implementation for: "+item.Description)
			v.store.UpdateItemStatus(item.ID, "missing", matched.Score)
		}
	}

	total := len(intent.Items)
	if total > 0 {
		result.CoverageScore = float64(len(result.Covered)) / float64(total)
		result.AlignmentScore = float64(len(result.Covered)+len(result.Partial)) / float64(total)
	}

	return result, nil
}

type FileInfo struct {
	Path     string
	Content  string
	Language string
}

type Match struct {
	Item    *IntentItem
	Feature *CodeFeature
	Score   float64
}

func findBestMatch(itemDescription string, features []*CodeFeature) *Match {
	itemWords := extractWords(strings.ToLower(itemDescription))

	var best *Match
	bestScore := 0.0

	for _, feature := range features {
		featureWords := extractFeatureWords(feature)

		overlap := countWordOverlap(itemWords, featureWords)
		score := float64(overlap) / float64(max(len(itemWords), len(featureWords)))

		if score > bestScore && score > 0.3 {
			bestScore = score
			best = &Match{
				Feature: feature,
				Score:   score,
			}
		}
	}

	return best
}

func extractWords(s string) []string {
	re := regexp.MustCompile(`\w+`)
	return re.FindAllString(s, -1)
}

func extractFeatureWords(f *CodeFeature) []string {
	var words []string

	nameWords := extractWords(strings.ToLower(f.Name))
	words = append(words, nameWords...)

	typeWords := extractWords(strings.ToLower(f.Type))
	words = append(words, typeWords...)

	sigWords := extractWords(strings.ToLower(f.Signature))
	words = append(words, sigWords...)

	for _, fn := range f.Functions {
		fnWords := extractWords(strings.ToLower(fn))
		words = append(words, fnWords...)
	}

	return words
}

func countWordOverlap(a, b []string) int {
	bSet := make(map[string]bool)
	for _, w := range b {
		bSet[w] = true
	}

	count := 0
	for _, w := range a {
		if bSet[w] {
			count++
		}
	}
	return count
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func AnalyzeCodeFeatures(files []*FileInfo) []*CodeFeature {
	var features []*CodeFeature

	for _, file := range files {
		switch file.Language {
		case "go":
			features = append(features, analyzeGoFeatures(file)...)
		case "typescript", "javascript":
			features = append(features, analyzeTypeScriptFeatures(file)...)
		case "python":
			features = append(features, analyzePythonFeatures(file)...)
		default:
			features = append(features, &CodeFeature{
				Type: "file",
				Name: file.Path,
				File: file.Path,
			})
		}
	}

	return features
}

func analyzeGoFeatures(file *FileInfo) []*CodeFeature {
	var features []*CodeFeature

	funcRegex := regexp.MustCompile(`func\s+(\([^)]+\)\s+)?(\w+)\s*\(([^)]*)\)\s*(?:\*?(\w+))?`)
	matches := funcRegex.FindAllStringSubmatch(file.Content, -1)

	for _, match := range matches {
		feature := &CodeFeature{
			Type:      "function",
			Name:      match[2],
			File:      file.Path,
			Signature: match[3],
		}
		if len(match) > 4 && match[4] != "" {
			feature.Signature += " " + match[4]
		}
		features = append(features, feature)
	}

	structRegex := regexp.MustCompile(`type\s+(\w+)\s+struct\s*\{`)
	structMatches := structRegex.FindAllStringSubmatch(file.Content, -1)
	for _, match := range structMatches {
		features = append(features, &CodeFeature{
			Type: "struct",
			Name: match[1],
			File: file.Path,
		})
	}

	return features
}

func analyzeTypeScriptFeatures(file *FileInfo) []*CodeFeature {
	var features []*CodeFeature

	funcRegex := regexp.MustCompile(`(?:function\s+(\w+)|(?:const|let|var)\s+(\w+)\s*=\s*(?:async\s*)?(?:\([^)]*\)|[^=])*\s*=>)`)
	matches := funcRegex.FindAllStringSubmatch(file.Content, -1)

	for _, match := range matches {
		name := match[1]
		if name == "" {
			name = match[2]
		}
		if name != "" {
			features = append(features, &CodeFeature{
				Type: "function",
				Name: name,
				File: file.Path,
			})
		}
	}

	classRegex := regexp.MustCompile(`class\s+(\w+)`)
	classMatches := classRegex.FindAllStringSubmatch(file.Content, -1)
	for _, match := range classMatches {
		features = append(features, &CodeFeature{
			Type: "class",
			Name: match[1],
			File: file.Path,
		})
	}

	return features
}

func analyzePythonFeatures(file *FileInfo) []*CodeFeature {
	var features []*CodeFeature

	funcRegex := regexp.MustCompile(`def\s+(\w+)\s*\(([^)]*)\)`)
	matches := funcRegex.FindAllStringSubmatch(file.Content, -1)

	for _, match := range matches {
		features = append(features, &CodeFeature{
			Type:      "function",
			Name:      match[1],
			File:      file.Path,
			Signature: match[2],
		})
	}

	classRegex := regexp.MustCompile(`class\s+(\w+)`)
	classMatches := classRegex.FindAllStringSubmatch(file.Content, -1)
	for _, match := range classMatches {
		features = append(features, &CodeFeature{
			Type: "class",
			Name: match[1],
			File: file.Path,
		})
	}

	return features
}
