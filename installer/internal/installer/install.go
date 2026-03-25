package installer

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
)

type InstallOptions struct {
	ProjectName string
	MCPClient   string
	InitPlugins bool
}

func Install(opts *InstallOptions) {
	fmt.Println("Ragnarok Installer v1.0.0")
	fmt.Println("=====================")
	fmt.Println()

	if opts.ProjectName == "" {
		fmt.Println("Error: --project is required")
		return
	}

	home, _ := os.UserHomeDir()
	installDir := filepath.Join(home, ".local", "bin")

	if err := os.MkdirAll(installDir, 0755); err != nil {
		fmt.Printf("Error creating install directory: %v\n", err)
		return
	}

	fmt.Printf("Installing Ragnarok plugins to: %s\n", installDir)
	fmt.Println()

	plugins := []string{"fenrir", "hati", "skoll", "tyr"}

	for _, plugin := range plugins {
		fmt.Printf("  Installing %s... ", plugin)
		if err := installPlugin(plugin, installDir); err != nil {
			fmt.Printf("SKIP (not built yet)\n")
		} else {
			fmt.Printf("OK\n")
		}
	}

	fmt.Println()

	if opts.MCPClient != "" {
		fmt.Printf("Configuring MCP for: %s\n", opts.MCPClient)
		if err := configureMCP(opts.MCPClient); err != nil {
			fmt.Printf("Error configuring MCP: %v\n", err)
		}
	}

	if opts.InitPlugins {
		fmt.Println()
		fmt.Println("Initializing plugins...")
		initializePlugins(opts.ProjectName)
	}

	fmt.Println()
	fmt.Println("Installation complete!")
	fmt.Println()
	fmt.Println("Next steps:")
	fmt.Println("  1. Add the bin directory to your PATH if not already done")
	fmt.Printf("  2. Run: fenrir init --project %s\n", opts.ProjectName)
	fmt.Println("  3. Run: hati init --project " + opts.ProjectName)
	fmt.Println("  4. Run: skoll init --project " + opts.ProjectName)
	fmt.Println("  5. Run: tyr init --project " + opts.ProjectName)
}

func installPlugin(name, installDir string) error {
	platform := runtime.GOOS
	arch := runtime.GOARCH
	ext := ""
	if platform == "windows" {
		ext = ".exe"
	}

	binaryName := fmt.Sprintf("%s_%s_%s", name, platform, arch)
	_ = fmt.Sprintf("https://github.com/ragnarok-ecosystem/%s/releases/latest/download/%s", name, binaryName)

	targetPath := filepath.Join(installDir, name+ext)

	if _, err := os.Stat(targetPath); err == nil {
		return nil
	}

	_, err := exec.LookPath(name + ext)
	if err == nil {
		return nil
	}

	return fmt.Errorf("binary not found locally and download not implemented yet")
}

func configureMCP(client string) error {
	home, _ := os.UserHomeDir()

	switch client {
	case "opencode":
		configPath := filepath.Join(home, ".opencode", "mcp.json")
		return writeOpenCodeMCPConfig(configPath)
	case "cursor":
		configPath := filepath.Join(home, ".cursor", "mcp.json")
		return writeCursorMCPConfig(configPath)
	case "windsurf":
		configPath := filepath.Join(home, ".windsurf", "mcp.yaml")
		return writeWindsurfMCPConfig(configPath)
	default:
		return fmt.Errorf("unsupported MCP client: %s", client)
	}
}

func writeOpenCodeMCPConfig(path string) error {
	config := `{
  "mcpServers": {
    "fenrir": {
      "command": "fenrir",
      "args": ["mcp"]
    },
    "hati": {
      "command": "hati",
      "args": ["mcp"]
    },
    "skoll": {
      "command": "skoll",
      "args": ["mcp"]
    },
    "tyr": {
      "command": "tyr",
      "args": ["mcp"]
    }
  }
}`

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	return os.WriteFile(path, []byte(config), 0644)
}

func writeCursorMCPConfig(path string) error {
	config := `{
  "mcpServers": {
    "fenrir": {
      "command": "fenrir",
      "args": ["serve", "--stdio"]
    },
    "hati": {
      "command": "hati",
      "args": ["serve", "--stdio"]
    },
    "skoll": {
      "command": "skoll",
      "args": ["serve", "--stdio"]
    },
    "tyr": {
      "command": "tyr",
      "args": ["serve", "--stdio"]
    }
  }
}`

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	return os.WriteFile(path, []byte(config), 0644)
}

func writeWindsurfMCPConfig(path string) error {
	config := `mcp_servers:
  fenrir:
    command: fenrir
    args: [serve, --stdio]
  hati:
    command: hati
    args: [serve, --stdio]
  skoll:
    command: skoll
    args: [serve, --stdio]
  tyr:
    command: tyr
    args: [serve, --stdio]
`

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	return os.WriteFile(path, []byte(config), 0644)
}

func initializePlugins(projectName string) {
	plugins := map[string]int{
		"fenrir": 7438,
		"hati":   7439,
		"skoll":  7441,
		"tyr":    7440,
	}

	for plugin, port := range plugins {
		fmt.Printf("  Initializing %s on port %d... ", plugin, port)
		fmt.Printf("OK (stub)\n")
	}
}
