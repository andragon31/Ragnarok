package audit

import (
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

type Package struct {
	Name         string   `json:"name"`
	Version      string   `json:"version"`
	ResolvedURL  string   `json:"resolved_url,omitempty"`
	Integrity    string   `json:"integrity,omitempty"`
	Dev          bool     `json:"dev,omitempty"`
	Dependencies []string `json:"dependencies,omitempty"`
	Ecosystem    string   `json:"ecosystem,omitempty"`
}

type AuditResult struct {
	Target        string     `json:"target"`
	PackageFile   string     `json:"package_file"`
	Packages      []*Package `json:"packages"`
	TotalPackages int        `json:"total_packages"`
	DevPackages   int        `json:"dev_packages"`
	AuditTime     string     `json:"audit_time"`
	Source        string     `json:"source"`
}

func AuditProject(projectPath string) (*AuditResult, error) {
	result := &AuditResult{
		Target: projectPath,
	}

	var lockFile string
	var lockFormat string

	files, _ := os.ReadDir(projectPath)
	for _, f := range files {
		name := f.Name()
		if strings.HasSuffix(name, ".lock") || strings.HasSuffix(name, "-lock.json") {
			switch name {
			case "package-lock.json":
				lockFile = filepath.Join(projectPath, name)
				lockFormat = "npm"
			case "yarn.lock":
				lockFile = filepath.Join(projectPath, name)
				lockFormat = "yarn"
			case "pnpm-lock.yaml":
				lockFile = filepath.Join(projectPath, name)
				lockFormat = "pnpm"
			case "Cargo.lock":
				lockFile = filepath.Join(projectPath, name)
				lockFormat = "cargo"
			case "go.sum":
				lockFile = filepath.Join(projectPath, name)
				lockFormat = "go"
			case "requirements.txt":
				lockFile = filepath.Join(projectPath, name)
				lockFormat = "pip"
			case "Pipfile.lock":
				lockFile = filepath.Join(projectPath, name)
				lockFormat = "pipenv"
			case "poetry.lock":
				lockFile = filepath.Join(projectPath, name)
				lockFormat = "poetry"
			case "composer.lock":
				lockFile = filepath.Join(projectPath, name)
				lockFormat = "composer"
			}
		}
	}

	if lockFile == "" {
		for _, f := range files {
			name := f.Name()
			switch name {
			case "package.json":
				if lockFile == "" {
					lockFile = filepath.Join(projectPath, name)
					lockFormat = "npm-manifest"
				}
			case "go.mod":
				if lockFile == "" {
					lockFile = filepath.Join(projectPath, name)
					lockFormat = "go-mod"
				}
			case "Cargo.toml":
				if lockFile == "" {
					lockFile = filepath.Join(projectPath, name)
					lockFormat = "cargo-manifest"
				}
			case "requirements.txt":
				if lockFile == "" {
					lockFile = filepath.Join(projectPath, name)
					lockFormat = "pip-manifest"
				}
			}
		}
	}

	result.PackageFile = lockFile
	result.Source = lockFormat

	if lockFile == "" {
		return result, nil
	}

	var err error
	switch lockFormat {
	case "npm", "pnpm":
		result.Packages, err = parseNPMLock(lockFile)
	case "cargo":
		result.Packages, err = parseCargoLock(lockFile)
	case "go", "go-mod":
		result.Packages, err = parseGoSum(lockFile)
	case "pip", "pipenv", "poetry":
		result.Packages, err = parseRequirements(lockFile)
	case "composer":
		result.Packages, err = parseComposerLock(lockFile)
	default:
		result.Packages, err = parseNPMLock(lockFile)
	}

	if err != nil {
		result.Packages = []*Package{}
	}

	for _, pkg := range result.Packages {
		result.TotalPackages++
		if pkg.Dev {
			result.DevPackages++
		}
	}

	return result, nil
}

func parseNPMLock(path string) ([]*Package, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	if filepath.Ext(path) == ".json" {
		var lock struct {
			Packages map[string]struct {
				Version      string   `json:"version"`
				Resolved     string   `json:"resolved"`
				Integrity    string   `json:"integrity"`
				Dev          bool     `json:"dev"`
				Dependencies []string `json:"dependencies"`
			} `json:"packages"`
		}

		if err := json.Unmarshal(data, &lock); err == nil {
			var packages []*Package
			for name, pkg := range lock.Packages {
				if name == "" {
					continue
				}
				name = strings.TrimPrefix(name, "node_modules/")
				packages = append(packages, &Package{
					Name:         name,
					Version:      pkg.Version,
					ResolvedURL:  pkg.Resolved,
					Integrity:    pkg.Integrity,
					Dev:          pkg.Dev,
					Dependencies: pkg.Dependencies,
				})
			}
			return packages, nil
		}
	}

	var packages []*Package
	lines := strings.Split(string(data), "\n")
	currentPkg := &Package{}

	for _, line := range lines {
		line = strings.TrimSpace(line)

		if strings.HasPrefix(line, "@") && !strings.Contains(line, "version") {
			if currentPkg.Name != "" {
				packages = append(packages, currentPkg)
			}
			nameParts := strings.SplitN(line, " ", 2)
			currentPkg = &Package{Name: nameParts[0]}
			continue
		}

		if strings.HasPrefix(line, "version") {
			parts := strings.SplitN(line, " ", 3)
			if len(parts) >= 2 {
				currentPkg.Version = strings.Trim(parts[1], `"'`)
			}
			continue
		}

		if strings.HasPrefix(line, "resolved") {
			parts := strings.SplitN(line, " ", 3)
			if len(parts) >= 2 {
				currentPkg.ResolvedURL = strings.Trim(parts[1], `"'`)
			}
			continue
		}

		if strings.HasPrefix(line, "integrity") {
			parts := strings.SplitN(line, " ", 3)
			if len(parts) >= 2 {
				currentPkg.Integrity = parts[1]
			}
			continue
		}
	}

	if currentPkg.Name != "" {
		packages = append(packages, currentPkg)
	}

	return packages, nil
}

func parseCargoLock(path string) ([]*Package, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var packages []*Package
	lines := strings.Split(string(data), "\n")
	inPackages := false
	var currentPkg *Package

	for _, line := range lines {
		if strings.TrimSpace(line) == "[[package]]" {
			inPackages = true
			currentPkg = &Package{}
			continue
		}

		if inPackages {
			if strings.TrimSpace(line) == "" || strings.HasPrefix(line, "[[package]]") {
				if currentPkg != nil && currentPkg.Name != "" {
					packages = append(packages, currentPkg)
				}
				currentPkg = &Package{}
				if strings.HasPrefix(line, "[[package]]") {
					continue
				}
				if line == "" {
					inPackages = false
				}
				continue
			}

			parts := strings.SplitN(line, " = ", 2)
			if len(parts) == 2 {
				key := strings.TrimSpace(parts[0])
				value := strings.Trim(parts[1], `"' `)

				switch key {
				case "name":
					currentPkg.Name = value
				case "version":
					currentPkg.Version = value
				case "source":
					if strings.Contains(value, "registry") {
						currentPkg.ResolvedURL = value
					}
				}
			}
		}
	}

	if currentPkg != nil && currentPkg.Name != "" {
		packages = append(packages, currentPkg)
	}

	return packages, nil
}

func parseGoSum(path string) ([]*Package, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var packages []*Package
	lines := strings.Split(string(data), "\n")

	re := regexp.MustCompile(`([^/\s]+)/v?\d+\.\d+\.\d+\s+([a-f0-9]+)`)

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		matches := re.FindStringSubmatch(line)
		if len(matches) >= 2 {
			name := matches[1]
			integrity := matches[2]

			version := ""
			parts := strings.Split(line, " ")
			if len(parts) >= 2 {
				versionParts := strings.Split(parts[0], "/")
				if len(versionParts) >= 2 {
					version = versionParts[len(versionParts)-1]
				}
			}

			packages = append(packages, &Package{
				Name:      name,
				Version:   version,
				Integrity: integrity,
				Ecosystem: "go",
			})
		}
	}

	return packages, nil
}

func parseRequirements(path string) ([]*Package, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var packages []*Package
	lines := strings.Split(string(data), "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)

		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		line = strings.SplitN(line, "#", 2)[0]

		var name, version string

		if strings.Contains(line, "==") {
			parts := strings.SplitN(line, "==", 2)
			name = strings.TrimSpace(parts[0])
			version = strings.TrimSpace(parts[1])
		} else if strings.Contains(line, ">=") {
			parts := strings.SplitN(line, ">=", 2)
			name = strings.TrimSpace(parts[0])
			version = ">=" + strings.TrimSpace(parts[1])
		} else if strings.Contains(line, "<=") {
			parts := strings.SplitN(line, "<=", 2)
			name = strings.TrimSpace(parts[0])
			version = "<=" + strings.TrimSpace(parts[1])
		} else if strings.Contains(line, "~=") {
			parts := strings.SplitN(line, "~=", 2)
			name = strings.TrimSpace(parts[0])
			version = "~=" + strings.TrimSpace(parts[1])
		} else if strings.Contains(line, "!=") {
			parts := strings.SplitN(line, "!=", 2)
			name = strings.TrimSpace(parts[0])
			version = "!=" + strings.TrimSpace(parts[1])
		} else if strings.Contains(line, " @ ") {
			parts := strings.SplitN(line, " @ ", 2)
			name = strings.TrimSpace(parts[0])
			version = ""
			url := strings.TrimSpace(parts[1])
			if strings.HasPrefix(url, "http") {
				packages = append(packages, &Package{
					Name:        name,
					Version:     version,
					ResolvedURL: url,
					Ecosystem:   "pip",
				})
				continue
			}
		} else {
			name = line
			version = "latest"
		}

		name = regexp.MustCompile(`\[.*?\]`).ReplaceAllString(name, "")

		if name != "" {
			packages = append(packages, &Package{
				Name:      name,
				Version:   version,
				Ecosystem: "pip",
			})
		}
	}

	return packages, nil
}

func parseComposerLock(path string) ([]*Package, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var lock struct {
		Packages []struct {
			Name       string `json:"name"`
			Version    string `json:"version"`
			DistURL    string `json:"dist_url"`
			RequireDev bool   `json:"require-dev"`
		} `json:"packages"`
	}

	if err := json.Unmarshal(data, &lock); err != nil {
		return nil, err
	}

	var packages []*Package
	for _, p := range lock.Packages {
		packages = append(packages, &Package{
			Name:        p.Name,
			Version:     p.Version,
			ResolvedURL: p.DistURL,
			Dev:         p.RequireDev,
			Ecosystem:   "composer",
		})
	}

	return packages, nil
}

func (r *AuditResult) ToMap() map[string]interface{} {
	return map[string]interface{}{
		"target":         r.Target,
		"package_file":   r.PackageFile,
		"format":         r.Source,
		"total_packages": r.TotalPackages,
		"dev_packages":   r.DevPackages,
		"audit_time":     r.AuditTime,
	}
}
