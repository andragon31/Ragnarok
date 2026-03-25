package skills

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

type SkillInfo struct {
	Name          string   `json:"name"`
	Description   string   `json:"description"`
	Version       string   `json:"version,omitempty"`
	License       string   `json:"license,omitempty"`
	Compatibility string   `json:"compatibility,omitempty"`
	Framework     string   `json:"framework,omitempty"`
	MinVersion    string   `json:"min_version,omitempty"`
	MaxVersion    string   `json:"max_version,omitempty"`
	LastVerified  string   `json:"last_verified,omitempty"`
	AllowedTools  []string `json:"allowed_tools,omitempty"`
	HasScripts    bool     `json:"has_scripts"`
	HasReferences bool     `json:"has_references"`
	HasAssets     bool     `json:"has_assets"`
	Source        string   `json:"source"`
	Tags          []string `json:"tags,omitempty"`
	Trigger       string   `json:"trigger,omitempty"`
	Content       string   `json:"content,omitempty"`
	Path          string   `json:"path"`
}

type SkillFile struct {
	Path    string `json:"path"`
	Type    string `json:"type"`
	Content string `json:"content,omitempty"`
}

type Frontmatter struct {
	Name          string `yaml:"name"`
	Description   string `yaml:"description"`
	License       string `yaml:"license"`
	Compatibility string `yaml:"compatibility"`
	Metadata      struct {
		Author       string `yaml:"author"`
		Version      string `yaml:"version"`
		Framework    string `yaml:"framework"`
		MinVersion   string `yaml:"min_version"`
		MaxVersion   string `yaml:"max_version"`
		LastVerified string `yaml:"last_verified"`
	} `yaml:"metadata"`
	AllowedTools []string `yaml:"allowed-tools"`
}

type SkillLoader struct {
	skillsDir string
}

func (l *SkillLoader) GetSkillsDir() string {
	return l.skillsDir
}

func NewSkillLoader(skillsDir string) *SkillLoader {
	return &SkillLoader{skillsDir: skillsDir}
}

func (l *SkillLoader) ListSkills() ([]*SkillInfo, error) {
	var skills []*SkillInfo

	entries, err := os.ReadDir(l.skillsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return skills, nil
		}
		return nil, fmt.Errorf("failed to read skills directory: %w", err)
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		skillPath := filepath.Join(l.skillsDir, entry.Name())
		skill, err := l.LoadSkillIndex(entry.Name())
		if err != nil {
			continue
		}
		skill.Path = skillPath
		skills = append(skills, skill)
	}

	return skills, nil
}

func (l *SkillLoader) LoadSkillIndex(skillName string) (*SkillInfo, error) {
	skillDir := filepath.Join(l.skillsDir, skillName)
	skillMD := filepath.Join(skillDir, "SKILL.md")

	info := &SkillInfo{
		Name:   skillName,
		Source: "local",
		Path:   skillDir,
	}

	if _, err := os.Stat(skillDir); os.IsNotExist(err) {
		return info, nil
	}

	info.HasScripts = l.hasSubdir(skillDir, "scripts")
	info.HasReferences = l.hasSubdir(skillDir, "references")
	info.HasAssets = l.hasSubdir(skillDir, "assets")

	frontmatter, body, err := l.parseSKILLMD(skillMD)
	if err != nil {
		return info, nil
	}

	info.Description = frontmatter.Description
	info.License = frontmatter.License
	info.Compatibility = frontmatter.Compatibility
	info.Version = frontmatter.Metadata.Version
	info.Framework = frontmatter.Metadata.Framework
	info.MinVersion = frontmatter.Metadata.MinVersion
	info.MaxVersion = frontmatter.Metadata.MaxVersion
	info.LastVerified = frontmatter.Metadata.LastVerified
	info.AllowedTools = frontmatter.AllowedTools

	if info.Description == "" && body != "" {
		lines := strings.Split(body, "\n")
		for i, line := range lines {
			line = strings.TrimSpace(line)
			if line != "" && !strings.HasPrefix(line, "#") {
				info.Description = strings.Join(lines[i:], "\n")
				if len(info.Description) > 200 {
					info.Description = info.Description[:200] + "..."
				}
				break
			}
		}
	}

	return info, nil
}

func (l *SkillLoader) LoadSkillFull(skillName string) (*SkillInfo, error) {
	skillDir := filepath.Join(l.skillsDir, skillName)
	skillMD := filepath.Join(skillDir, "SKILL.md")

	info := &SkillInfo{
		Name:   skillName,
		Source: "local",
		Path:   skillDir,
	}

	if _, err := os.Stat(skillDir); os.IsNotExist(err) {
		return nil, fmt.Errorf("skill not found: %s", skillName)
	}

	info.HasScripts = l.hasSubdir(skillDir, "scripts")
	info.HasReferences = l.hasSubdir(skillDir, "references")
	info.HasAssets = l.hasSubdir(skillDir, "assets")

	frontmatter, body, err := l.parseSKILLMD(skillMD)
	if err != nil {
		return nil, fmt.Errorf("failed to parse SKILL.md: %w", err)
	}

	info.Description = frontmatter.Description
	info.License = frontmatter.License
	info.Compatibility = frontmatter.Compatibility
	info.Version = frontmatter.Metadata.Version
	info.Framework = frontmatter.Metadata.Framework
	info.MinVersion = frontmatter.Metadata.MinVersion
	info.MaxVersion = frontmatter.Metadata.MaxVersion
	info.LastVerified = frontmatter.Metadata.LastVerified
	info.AllowedTools = frontmatter.AllowedTools
	info.Content = body

	return info, nil
}

func (l *SkillLoader) LoadSkillFile(skillName, filePath string) (*SkillFile, error) {
	skillDir := filepath.Join(l.skillsDir, skillName)
	validTypes := map[string]bool{"scripts": true, "references": true, "assets": true}

	parts := strings.Split(filePath, "/")
	if len(parts) < 2 {
		return nil, fmt.Errorf("invalid file path: %s (must be relative to skill dir)", filePath)
	}

	dirType := parts[0]
	if !validTypes[dirType] {
		return nil, fmt.Errorf("invalid directory type: %s (allowed: scripts, references, assets)", dirType)
	}

	targetPath := filepath.Join(skillDir, filePath)

	if !strings.HasPrefix(targetPath, skillDir) {
		return nil, fmt.Errorf("path traversal detected: %s", filePath)
	}

	content, err := os.ReadFile(targetPath)
	if err != nil {
		return nil, fmt.Errorf("file not found: %s", filePath)
	}

	return &SkillFile{
		Path:    filePath,
		Type:    dirType,
		Content: string(content),
	}, nil
}

func (l *SkillLoader) ListSkillFiles(skillName string) (map[string][]string, error) {
	skillDir := filepath.Join(l.skillsDir, skillName)
	files := map[string][]string{
		"scripts":    {},
		"references": {},
		"assets":     {},
	}

	for dirType := range files {
		dirPath := filepath.Join(skillDir, dirType)
		entries, err := os.ReadDir(dirPath)
		if err != nil {
			continue
		}
		for _, entry := range entries {
			if !entry.IsDir() {
				files[dirType] = append(files[dirType], entry.Name())
			}
		}
	}

	return files, nil
}

func (l *SkillLoader) hasSubdir(dir, name string) bool {
	path := filepath.Join(dir, name)
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}

func (l *SkillLoader) parseSKILLMD(path string) (*Frontmatter, string, error) {
	file, err := os.Open(path)
	if err != nil {
		return &Frontmatter{}, "", err
	}
	defer file.Close()

	var frontmatter Frontmatter
	var bodyLines []string
	inFrontmatter := false
	frontmatterLines := []string{}
	bodyStart := false

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()

		if strings.TrimSpace(line) == "---" {
			if !inFrontmatter && !bodyStart {
				inFrontmatter = true
				continue
			}
			if inFrontmatter {
				if err := yaml.Unmarshal([]byte(strings.Join(frontmatterLines, "\n")), &frontmatter); err != nil {
					return &Frontmatter{}, "", err
				}
				inFrontmatter = false
				bodyStart = true
				continue
			}
		}

		if inFrontmatter {
			frontmatterLines = append(frontmatterLines, line)
		} else if bodyStart || (len(frontmatterLines) == 0 && strings.TrimSpace(line) != "") {
			bodyStart = true
			bodyLines = append(bodyLines, line)
		}
	}

	return &frontmatter, strings.Join(bodyLines, "\n"), nil
}

func (l *SkillLoader) CreateSkill(name string, description string) error {
	skillDir := filepath.Join(l.skillsDir, name)
	if err := os.MkdirAll(skillDir, 0755); err != nil {
		return fmt.Errorf("failed to create skill directory: %w", err)
	}

	skillMD := filepath.Join(skillDir, "SKILL.md")
	content := fmt.Sprintf(`---
name: %s
description: |
  %s
license: MIT
allowed-tools:
---

## Cuándo aplicar

[instrucciones...]

## Proceso

### Paso 1

[instrucciones...]

## Checklist

- [ ]

## Anti-patrones

-

## Ver también

`, name, description)

	if err := os.WriteFile(skillMD, []byte(content), 0644); err != nil {
		return fmt.Errorf("failed to create SKILL.md: %w", err)
	}

	scriptsDir := filepath.Join(skillDir, "scripts")
	referencesDir := filepath.Join(skillDir, "references")
	assetsDir := filepath.Join(skillDir, "assets")

	os.MkdirAll(scriptsDir, 0755)
	os.MkdirAll(referencesDir, 0755)
	os.MkdirAll(assetsDir, 0755)

	return nil
}

func (l *SkillLoader) SearchSkills(query string) ([]*SkillInfo, error) {
	allSkills, err := l.ListSkills()
	if err != nil {
		return nil, err
	}

	queryLower := strings.ToLower(query)
	var results []*SkillInfo

	for _, skill := range allSkills {
		if strings.Contains(strings.ToLower(skill.Name), queryLower) ||
			strings.Contains(strings.ToLower(skill.Description), queryLower) ||
			strings.Contains(strings.ToLower(skill.Framework), queryLower) {
			results = append(results, skill)
		}
	}

	return results, nil
}

func SkillToJSON(skills []*SkillInfo) string {
	data, _ := json.MarshalIndent(skills, "", "  ")
	return string(data)
}

func (s *SkillInfo) ToMap() map[string]interface{} {
	return map[string]interface{}{
		"name":                s.Name,
		"description":         s.Description,
		"version":             s.Version,
		"framework":           s.Framework,
		"has_scripts":         s.HasScripts,
		"has_references":      s.HasReferences,
		"has_assets":          s.HasAssets,
		"source":              s.Source,
		"allowed_tools":       s.AllowedTools,
		"allowed_tools_count": len(s.AllowedTools),
		"license":             s.License,
		"compatibility":       s.Compatibility,
		"last_verified":       s.LastVerified,
		"path":                s.Path,
	}
}

func (s *SkillInfo) ToFullMap(files map[string][]string) map[string]interface{} {
	m := s.ToMap()
	m["content"] = s.Content
	m["available_files"] = files
	return m
}

type SkillIndexEntry struct {
	Name              string `json:"name"`
	Description       string `json:"description"`
	HasScripts        bool   `json:"has_scripts"`
	HasReferences     bool   `json:"has_references"`
	HasAssets         bool   `json:"has_assets"`
	VersionStatus     string `json:"version_status"`
	AllowedToolsCount int    `json:"allowed_tools_count"`
}

func BuildSkillIndex(skills []*SkillInfo) []*SkillIndexEntry {
	var index []*SkillIndexEntry
	for _, s := range skills {
		versionStatus := "current"
		if s.LastVerified != "" {
			if t, err := time.Parse("2006-01-02", s.LastVerified); err == nil {
				if time.Since(t).Hours() > 24*30 {
					versionStatus = "stale"
				}
			}
		}
		index = append(index, &SkillIndexEntry{
			Name:              s.Name,
			Description:       s.Description,
			HasScripts:        s.HasScripts,
			HasReferences:     s.HasReferences,
			HasAssets:         s.HasAssets,
			VersionStatus:     versionStatus,
			AllowedToolsCount: len(s.AllowedTools),
		})
	}
	return index
}
