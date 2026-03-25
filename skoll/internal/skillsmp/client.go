package skillsmp

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

type Client struct {
	httpClient *http.Client
}

func NewClient() *Client {
	return &Client{
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

type SkillInfo struct {
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Author      string   `json:"author"`
	Stars       int      `json:"stars"`
	License     string   `json:"license"`
	Topics      []string `json:"topics"`
	Readme      string   `json:"readme,omitempty"`
}

type SearchResult struct {
	Skills []*SkillInfo `json:"skills"`
	Total  int          `json:"total"`
	Query  string       `json:"query"`
}

func (c *Client) SearchSkills(query string, limit int) (*SearchResult, error) {
	if limit == 0 {
		limit = 10
	}

	url := fmt.Sprintf("https://api.github.com/search/repositories?q=%s+agentskills+OR+claude+OR+skill&sort=stars&per_page=%d",
		strings.ReplaceAll(query, " ", "+"), limit)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Accept", "application/vnd.github.v3+json")
	req.Header.Set("User-Agent", "Ragnarok-Skoll/1.0")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("GitHub API error: %d", resp.StatusCode)
	}

	var result struct {
		TotalCount int `json:"total_count"`
		Items      []struct {
			Name        string `json:"name"`
			Description string `json:"description"`
			Owner       struct {
				Login string `json:"login"`
			} `json:"owner"`
			StargazersCount int `json:"stargazers_count"`
			License         *struct {
				SPDXID string `json:"spdx_id"`
			} `json:"license"`
			Topics []string `json:"topics"`
		} `json:"items"`
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if err := json.Unmarshal(body, &result); err != nil {
		return nil, err
	}

	skills := make([]*SkillInfo, 0, len(result.Items))
	for _, item := range result.Items {
		license := ""
		if item.License != nil {
			license = item.License.SPDXID
		}
		skills = append(skills, &SkillInfo{
			Name:        item.Name,
			Description: item.Description,
			Author:      item.Owner.Login,
			Stars:       item.StargazersCount,
			License:     license,
			Topics:      item.Topics,
		})
	}

	return &SearchResult{
		Skills: skills,
		Total:  result.TotalCount,
		Query:  query,
	}, nil
}

func (c *Client) GetReadme(owner, repo string) (string, error) {
	url := fmt.Sprintf("https://api.github.com/repos/%s/%s/readme", owner, repo)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", err
	}

	req.Header.Set("Accept", "application/vnd.github.v3+json")
	req.Header.Set("User-Agent", "Ragnarok-Skoll/1.0")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return "", fmt.Errorf("README not found: %d", resp.StatusCode)
	}

	var result struct {
		Content string `json:"content"`
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	if err := json.Unmarshal(body, &result); err != nil {
		return "", err
	}

	return decodeBase64(result.Content), nil
}

func decodeBase64(s string) string {
	return s
}

func (c *Client) CloneOrDownloadSkill(url string, destDir string) error {
	url = strings.TrimSuffix(url, ".git")
	url = strings.TrimSuffix(url, "/")

	parts := strings.Split(url, "/")
	if len(parts) < 2 {
		return fmt.Errorf("invalid URL: %s", url)
	}

	owner := parts[len(parts)-2]
	repo := parts[len(parts)-1]

	apiURL := fmt.Sprintf("https://api.github.com/repos/%s/%s", owner, repo)

	req, err := http.NewRequest("GET", apiURL, nil)
	if err != nil {
		return err
	}

	req.Header.Set("Accept", "application/vnd.github.v3+json")
	req.Header.Set("User-Agent", "Ragnarok-Skoll/1.0")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return fmt.Errorf("repo not found: %d", resp.StatusCode)
	}

	var repoInfo struct {
		DefaultBranch string `json:"default_branch"`
		Description   string `json:"description"`
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	if err := json.Unmarshal(body, &repoInfo); err != nil {
		return err
	}

	contentsURL := fmt.Sprintf("https://api.github.com/repos/%s/%s/contents/SKILL.md", owner, repo)

	req2, err := http.NewRequest("GET", contentsURL, nil)
	if err != nil {
		return err
	}

	req2.Header.Set("Accept", "application/vnd.github.v3+json")
	req2.Header.Set("User-Agent", "Ragnarok-Skoll/1.0")

	resp2, err := c.httpClient.Do(req2)
	if err != nil {
		return err
	}
	defer resp2.Body.Close()

	if resp2.StatusCode == 200 {
		var skillMD struct {
			Content string `json:"content"`
		}

		body2, err := io.ReadAll(resp2.Body)
		if err != nil {
			return err
		}

		if err := json.Unmarshal(body2, &skillMD); err != nil {
			return err
		}

		skillPath := filepath.Join(destDir, repo, "SKILL.md")
		os.MkdirAll(filepath.Dir(skillPath), 0755)

		content := decodeBase64(skillMD.Content)
		if err := os.WriteFile(skillPath, []byte(content), 0644); err != nil {
			return err
		}
	}

	return nil
}

func ExtractSkillNameFromURL(url string) string {
	url = strings.TrimSuffix(url, ".git")
	url = strings.TrimSuffix(url, "/")

	parts := strings.Split(url, "/")
	if len(parts) < 2 {
		return ""
	}

	name := parts[len(parts)-1]
	name = strings.ToLower(name)
	name = regexp.MustCompile(`[^a-z0-9]+`).ReplaceAllString(name, "-")
	return name
}
