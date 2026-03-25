package registry

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

type PackageInfo struct {
	Name              string `json:"name"`
	Version           string `json:"version,omitempty"`
	Exists            bool   `json:"exists"`
	Trusted           bool   `json:"trusted"`
	CVECount          int    `json:"cve_count"`
	AgeDays           int    `json:"age_days"`
	DownloadsMonthly  int64  `json:"downloads_monthly"`
	TyposquattingRisk bool   `json:"typosquatting_risk"`
	Description       string `json:"description,omitempty"`
	License           string `json:"license,omitempty"`
	LatestVersion     string `json:"latest_version,omitempty"`
	Source            string `json:"source"`
	Error             string `json:"error,omitempty"`
}

type RegistryClient struct {
	httpClient *http.Client
}

func NewRegistryClient() *RegistryClient {
	return &RegistryClient{
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

func (c *RegistryClient) CheckNPM(packageName string) (*PackageInfo, error) {
	url := fmt.Sprintf("https://registry.npmjs.org/%s", packageName)

	resp, err := c.httpClient.Get(url)
	if err != nil {
		return &PackageInfo{Name: packageName, Source: "npm", Error: err.Error()}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == 404 {
		return &PackageInfo{Name: packageName, Exists: false, Source: "npm"}, nil
	}

	if resp.StatusCode != 200 {
		return &PackageInfo{Name: packageName, Source: "npm", Error: fmt.Sprintf("status: %d", resp.StatusCode)}, fmt.Errorf("npm API error: %d", resp.StatusCode)
	}

	var result struct {
		Name        string `json:"name"`
		Description string `json:"description"`
		License     string `json:"license"`
		Time        struct {
			Created  string `json:"created"`
			Modified string `json:"modified"`
		} `json:"time"`
		DistTags struct {
			Latest string `json:"latest"`
		} `json:"dist-tags"`
		Downloads struct {
			Monthly int64 `json:"monthly"`
		} `json:"downloads"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return &PackageInfo{Name: packageName, Source: "npm", Error: err.Error()}, err
	}

	createdTime, _ := time.Parse("2006-01-02T15:04:05.000Z", result.Time.Created)
	ageDays := int(time.Since(createdTime).Hours() / 24)

	return &PackageInfo{
		Name:             packageName,
		Exists:           true,
		Trusted:          true,
		AgeDays:          ageDays,
		DownloadsMonthly: result.Downloads.Monthly,
		Description:      result.Description,
		License:          result.License,
		LatestVersion:    result.DistTags.Latest,
		Source:           "npm",
	}, nil
}

func (c *RegistryClient) CheckPyPI(packageName string) (*PackageInfo, error) {
	url := fmt.Sprintf("https://pypi.org/pypi/%s/json", packageName)

	resp, err := c.httpClient.Get(url)
	if err != nil {
		return &PackageInfo{Name: packageName, Source: "pypi", Error: err.Error()}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == 404 {
		return &PackageInfo{Name: packageName, Exists: false, Source: "pypi"}, nil
	}

	if resp.StatusCode != 200 {
		return &PackageInfo{Name: packageName, Source: "pypi", Error: fmt.Sprintf("status: %d", resp.StatusCode)}, fmt.Errorf("pypi API error: %d", resp.StatusCode)
	}

	var result struct {
		Info struct {
			Name           string `json:"name"`
			Summary        string `json:"summary"`
			License        string `json:"license"`
			Version        string `json:"version"`
			RequiresPython string `json:"requires_python"`
		} `json:"info"`
		Releases struct {
		} `json:"releases"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return &PackageInfo{Name: packageName, Source: "pypi", Error: err.Error()}, err
	}

	ageDays := 0

	summary := result.Info.Summary
	if len(summary) > 200 {
		summary = summary[:200] + "..."
	}

	return &PackageInfo{
		Name:          packageName,
		Exists:        true,
		Trusted:       true,
		AgeDays:       ageDays,
		Description:   summary,
		License:       result.Info.License,
		LatestVersion: result.Info.Version,
		Source:        "pypi",
	}, nil
}

func (c *RegistryClient) CheckCratesIO(packageName string) (*PackageInfo, error) {
	url := fmt.Sprintf("https://crates.io/api/v1/crates/%s", packageName)

	resp, err := c.httpClient.Get(url)
	if err != nil {
		return &PackageInfo{Name: packageName, Source: "crates.io", Error: err.Error()}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == 404 {
		return &PackageInfo{Name: packageName, Exists: false, Source: "crates.io"}, nil
	}

	if resp.StatusCode != 200 {
		return &PackageInfo{Name: packageName, Source: "crates.io", Error: fmt.Sprintf("status: %d", resp.StatusCode)}, fmt.Errorf("crates.io API error: %d", resp.StatusCode)
	}

	var result struct {
		Crate struct {
			Name        string `json:"name"`
			Description string `json:"description"`
			License     string `json:"license"`
			MaxVersion  string `json:"max_version"`
			CreatedAt   string `json:"created_at"`
			UpdatedAt   string `json:"updated_at"`
		} `json:"crate"`
		Versions []struct {
			Downloads int64 `json:"downloads"`
		} `json:"versions"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return &PackageInfo{Name: packageName, Source: "crates.io", Error: err.Error()}, err
	}

	createdTime, _ := time.Parse("2006-01-02T15:04:05.000Z", result.Crate.CreatedAt)
	ageDays := int(time.Since(createdTime).Hours() / 24)

	var totalDownloads int64
	for _, v := range result.Versions {
		totalDownloads += v.Downloads
	}

	return &PackageInfo{
		Name:             packageName,
		Exists:           true,
		Trusted:          true,
		AgeDays:          ageDays,
		DownloadsMonthly: totalDownloads,
		Description:      result.Crate.Description,
		License:          result.Crate.License,
		LatestVersion:    result.Crate.MaxVersion,
		Source:           "crates.io",
	}, nil
}

func (c *RegistryClient) CheckNuGet(packageName string) (*PackageInfo, error) {
	url := fmt.Sprintf("https://api.nuget.org/v3-flatcontainer/%s/index.json", strings.ToLower(packageName))

	resp, err := c.httpClient.Get(url)
	if err != nil {
		return &PackageInfo{Name: packageName, Source: "nuget", Error: err.Error()}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == 404 {
		return &PackageInfo{Name: packageName, Exists: false, Source: "nuget"}, nil
	}

	if resp.StatusCode != 200 {
		return &PackageInfo{Name: packageName, Source: "nuget", Error: fmt.Sprintf("status: %d", resp.StatusCode)}, fmt.Errorf("nuget API error: %d", resp.StatusCode)
	}

	var result struct {
		Versions []string `json:"versions"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return &PackageInfo{Name: packageName, Source: "nuget", Error: err.Error()}, err
	}

	latestVersion := ""
	if len(result.Versions) > 0 {
		latestVersion = result.Versions[len(result.Versions)-1]
	}

	return &PackageInfo{
		Name:          packageName,
		Exists:        true,
		Trusted:       true,
		LatestVersion: latestVersion,
		Source:        "nuget",
	}, nil
}

func (c *RegistryClient) CheckPackage(ecosystem, packageName string) (*PackageInfo, error) {
	switch strings.ToLower(ecosystem) {
	case "npm", "node":
		return c.CheckNPM(packageName)
	case "pypi", "pip", "python":
		return c.CheckPyPI(packageName)
	case "crates", "cargo", "rust":
		return c.CheckCratesIO(packageName)
	case "nuget", "dotnet":
		return c.CheckNuGet(packageName)
	default:
		return &PackageInfo{
			Name:  packageName,
			Error: fmt.Sprintf("unsupported ecosystem: %s", ecosystem),
		}, fmt.Errorf("unsupported ecosystem: %s", ecosystem)
	}
}

func (c *RegistryClient) CheckTyposquatting(packageName, ecosystem string) bool {
	commonTypos := []string{
		packageName + "-pkg", packageName + "-lib", packageName + "-dev",
		"lib" + packageName, "node-" + packageName, packageName + "-js",
	}

	for _, typo := range commonTypos {
		info, err := c.CheckPackage(ecosystem, typo)
		if err == nil && info.Exists {
			return true
		}
	}

	return false
}

func CheckOSV(packageName, ecosystem string) (int, error) {
	url := fmt.Sprintf("https://api.osv.dev/v1/query?package_name=%s&ecosystem=%s", packageName, ecosystem)

	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return 0, fmt.Errorf("OSV API error: %d", resp.StatusCode)
	}

	var result struct {
		Vulns []interface{} `json:"vulns"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return 0, err
	}

	return len(result.Vulns), nil
}
