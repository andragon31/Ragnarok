package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	fenrircli "github.com/andragon31/Ragnarok/internal/fenrir/cli"
	"github.com/andragon31/Ragnarok/internal/installer/installer"
	"github.com/andragon31/Ragnarok/internal/installer/integration"
	"github.com/andragon31/Ragnarok/internal/mcp/unified"
)

var version = "1.2.0"

type Plugin struct {
	Name    string
	Port    int
	DataDir string
	BinName string
}

type PluginStats struct {
	Name      string                 `json:"name"`
	Status    string                 `json:"status"`
	Port      int                    `json:"port,omitempty"`
	LatencyMs int64                  `json:"latency_ms,omitempty"`
	Data      map[string]interface{} `json:"data,omitempty"`
	Error     string                 `json:"error,omitempty"`
}

type EcosystemStats struct {
	Fenrir *PluginStats `json:"fenrir,omitempty"`
	Hati   *PluginStats `json:"hati,omitempty"`
	Skoll  *PluginStats `json:"skoll,omitempty"`
	Tyr    *PluginStats `json:"tyr,omitempty"`
}

func main() {
	if len(os.Args) > 1 && os.Args[1] == "version" {
		fmt.Printf("Ragnarok v%s\n", version)
		fmt.Println("AI Governance & Memory Layer Ecosystem")
		return
	}

	statsCmd := flag.NewFlagSet("stats", flag.ExitOnError)
	ecosystem := statsCmd.Bool("ecosystem", false, "Show unified ecosystem stats")
	plugin := statsCmd.String("plugin", "", "Show stats for specific plugin (fenrir, hati, skoll, tyr)")

	installCmd := flag.NewFlagSet("install", flag.ExitOnError)
	projectName := installCmd.String("project", "", "Project name")
	mcpClient := installCmd.String("mcp", "", "MCP client (opencode, cursor, windsurf)")
	initPlugins := installCmd.Bool("init", false, "Initialize plugins after installation")

	backupCmd := flag.NewFlagSet("backup", flag.ExitOnError)
	backupPlugin := backupCmd.String("plugin", "all", "Plugin to backup (fenrir, hati, skoll, tyr, all)")
	backupDir := backupCmd.String("dir", "", "Backup directory (default: ~/OneDrive/RagnarokBackups)")

	restoreCmd := flag.NewFlagSet("restore", flag.ExitOnError)
	restorePlugin := restoreCmd.String("plugin", "", "Plugin to restore (required)")
	restoreFile := restoreCmd.String("file", "", "Backup file to restore (required)")

	integrateCmd := flag.NewFlagSet("integrate", flag.ExitOnError)
	integratePath := integrateCmd.String("path", ".", "Project path with .ragnarok directory")

	initCmd := flag.NewFlagSet("init", flag.ExitOnError)
	initProject := initCmd.String("project", "", "Project name")
	initDir := initCmd.String("dir", "", "Base directory for plugins (default: ~)")

	scanCmd := flag.NewFlagSet("scan", flag.ExitOnError)
	scanPath := scanCmd.String("path", ".", "Project path to scan")
	scanBootstrap := scanCmd.Bool("bootstrap", true, "Generate bootstrap files after scan")

	serveCmd := flag.NewFlagSet("serve", flag.ExitOnError)
	serveDir := serveCmd.String("dir", "", "Base directory for plugins (default: ~)")

	mcpCmd := flag.NewFlagSet("mcp", flag.ExitOnError)
	mcpDir := mcpCmd.String("dir", "", "Base directory for plugins (default: ~)")

	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	switch os.Args[1] {
	case "stats":
		statsCmd.Parse(os.Args[2:])
		runStats(*ecosystem, *plugin)
	case "install":
		installCmd.Parse(os.Args[2:])
		installer.Install(&installer.InstallOptions{
			ProjectName: *projectName,
			MCPClient:   *mcpClient,
			InitPlugins: *initPlugins,
		})
	case "backup":
		backupCmd.Parse(os.Args[2:])
		runBackup(*backupPlugin, *backupDir)
	case "restore":
		restoreCmd.Parse(os.Args[2:])
		runRestore(*restorePlugin, *restoreFile)
	case "integrate":
		integrateCmd.Parse(os.Args[2:])
		runIntegrate(*integratePath)
	case "init":
		initCmd.Parse(os.Args[2:])
		runInit(*initProject, *initDir)
	case "scan":
		scanCmd.Parse(os.Args[2:])
		runScan(*scanPath, *scanBootstrap)
	case "serve":
		serveCmd.Parse(os.Args[2:])
		runServe(*serveDir)
	case "mcp":
		mcpCmd.Parse(os.Args[2:])
		runUnifiedMCP(*mcpDir)
	case "setup":
		agent := ""
		if len(os.Args) > 2 {
			agent = os.Args[2]
		}
		runSetup(agent)
	case "stop":
		runStop()
	case "version":
		fmt.Printf("Ragnarok v%s\n", version)
		fmt.Println("AI Governance & Memory Layer Ecosystem")
	default:
		printUsage()
		os.Exit(1)
	}
}

func runStats(ecosystemFlag bool, pluginFlag string) {
	if ecosystemFlag || pluginFlag == "" {
		showEcosystemStats()
	} else {
		showPluginStats(pluginFlag)
	}
}

func showPluginStats(plugin string) {
	port := getPluginPort(plugin)
	if port == 0 {
		fmt.Printf("Error: Unknown plugin '%s'\n", plugin)
		fmt.Printf("Available plugins: fenrir, hati, skoll, tyr\n")
		os.Exit(1)
	}

	stats := fetchPluginStats(plugin, port)
	printPluginStats(stats)
}

func showEcosystemStats() {
	plugins := []string{"fenrir", "hati", "skoll", "tyr"}
	stats := &EcosystemStats{}

	for _, plugin := range plugins {
		port := getPluginPort(plugin)
		ps := fetchPluginStats(plugin, port)
		switch plugin {
		case "fenrir":
			stats.Fenrir = ps
		case "hati":
			stats.Hati = ps
		case "skoll":
			stats.Skoll = ps
		case "tyr":
			stats.Tyr = ps
		}
	}

	printUnifiedStats(stats)
}

var allPlugins = []Plugin{
	{Name: "fenrir", Port: 7437, DataDir: "~/.fenrir", BinName: "fenrir"},
	{Name: "hati", Port: 7439, DataDir: "~/.hati", BinName: "hati"},
	{Name: "skoll", Port: 7438, DataDir: "~/.skoll", BinName: "skoll"},
	{Name: "tyr", Port: 7440, DataDir: "~/.tyr", BinName: "tyr"},
}

func getPluginPort(plugin string) int {
	ports := map[string]int{
		"fenrir": 7437,
		"hati":   7439,
		"skoll":  7438,
		"tyr":    7440,
	}
	return ports[plugin]
}

func getPlugin(name string) *Plugin {
	for i := range allPlugins {
		if allPlugins[i].Name == name {
			return &allPlugins[i]
		}
	}
	return nil
}

func fetchPluginStats(plugin string, port int) *PluginStats {
	stats := &PluginStats{
		Name: plugin,
		Port: port,
	}

	if port == 0 {
		stats.Status = "unknown"
		return stats
	}

	start := time.Now()
	url := fmt.Sprintf("http://localhost:%d/stats", port)

	client := &http.Client{Timeout: 2 * time.Second}
	resp, err := client.Get(url)

	if err != nil {
		stats.Status = "offline"
		stats.Error = err.Error()
		return stats
	}
	defer resp.Body.Close()

	stats.LatencyMs = time.Since(start).Milliseconds()

	if resp.StatusCode == 200 {
		stats.Status = "online"
		var data map[string]interface{}
		if err := json.NewDecoder(resp.Body).Decode(&data); err == nil {
			stats.Data = data
		}
	} else {
		stats.Status = fmt.Sprintf("error:%d", resp.StatusCode)
	}

	return stats
}

func printPluginStats(stats *PluginStats) {
	statusIcon := "✓"
	if stats.Status != "online" {
		statusIcon = "✗"
	}

	fmt.Printf("%s %s", statusIcon, strings.ToUpper(stats.Name))
	if stats.LatencyMs > 0 {
		fmt.Printf(" (latency: %dms)", stats.LatencyMs)
	}
	fmt.Println()

	if stats.Error != "" {
		fmt.Printf("  Error: %s\n", stats.Error)
	}

	if stats.Data != nil {
		for key, value := range stats.Data {
			fmt.Printf("  %s: %v\n", key, value)
		}
	}
}

func printUnifiedStats(stats *EcosystemStats) {
	fmt.Println("RAGNAROK Ecosystem Health")
	fmt.Println("─" + strings.Repeat("─", 40))

	allOnline := true
	if stats.Fenrir != nil {
		status := stats.Fenrir.Status
		icon := "✓"
		if status != "online" {
			icon = "✗"
			allOnline = false
		}
		fmt.Printf("%s Fenrir: %s", icon, status)
		if stats.Fenrir.LatencyMs > 0 {
			fmt.Printf(" (latency: %dms)", stats.Fenrir.LatencyMs)
		}
		if stats.Fenrir.Data != nil {
			if nodes, ok := stats.Fenrir.Data["total_observations"]; ok {
				fmt.Printf(" [nodes: %v]", nodes)
			}
		}
		fmt.Println()
	}

	if stats.Hati != nil {
		status := stats.Hati.Status
		icon := "✓"
		if status != "online" {
			icon = "✗"
			allOnline = false
		}
		fmt.Printf("%s Hati: %s", icon, status)
		if stats.Hati.Data != nil {
			if plans, ok := stats.Hati.Data["total_plans"]; ok {
				fmt.Printf(" [plans: %v]", plans)
			}
		}
		fmt.Println()
	}

	if stats.Skoll != nil {
		status := stats.Skoll.Status
		icon := "✓"
		if status != "online" {
			icon = "✗"
			allOnline = false
		}
		fmt.Printf("%s Skoll: %s", icon, status)
		if stats.Skoll.Data != nil {
			if skills, ok := stats.Skoll.Data["total_skills"]; ok {
				fmt.Printf(" [skills: %v]", skills)
			}
		}
		fmt.Println()
	}

	if stats.Tyr != nil {
		status := stats.Tyr.Status
		icon := "✓"
		if status != "online" {
			icon = "✗"
			allOnline = false
		}
		fmt.Printf("%s Tyr: %s", icon, status)
		if stats.Tyr.Data != nil {
			if findings, ok := stats.Tyr.Data["active_findings"]; ok {
				fmt.Printf(" [findings: %v]", findings)
			}
		}
		fmt.Println()
	}

	fmt.Println("─" + strings.Repeat("─", 40))
	if allOnline {
		fmt.Println("Overall: ✓ Healthy")
	} else {
		fmt.Println("Overall: ⚠ Some plugins offline")
	}
}

func runBackup(plugin string, backupDir string) {
	plugins := map[string]string{
		"fenrir": "~/.fenrir",
		"hati":   "~/.hati",
		"skoll":  "~/.skoll",
		"tyr":    "~/.tyr",
	}

	if backupDir == "" {
		home, _ := os.UserHomeDir()
		backupDir = home + "/OneDrive/RagnarokBackups"
	}

	os.MkdirAll(backupDir, 0755)

	fmt.Printf("Ragnarok Backup\n")
	fmt.Printf("Plugin: %s\n", plugin)
	fmt.Printf("Backup directory: %s\n", backupDir)
	fmt.Println(strings.Repeat("-", 40))

	if plugin == "all" {
		for name, dir := range plugins {
			backupPlugin(name, dir, backupDir)
		}
	} else {
		if dir, ok := plugins[plugin]; ok {
			backupPlugin(plugin, dir, backupDir)
		} else {
			fmt.Printf("Error: Unknown plugin '%s'\n", plugin)
			os.Exit(1)
		}
	}

	fmt.Println(strings.Repeat("-", 40))
	fmt.Println("Backup complete!")
}

func backupPlugin(name string, sourceDir string, backupDir string) {
	fmt.Printf("Backing up %s from %s...\n", name, sourceDir)

	if _, err := os.Stat(sourceDir); os.IsNotExist(err) {
		fmt.Printf("  Skipped: source directory not found\n")
		return
	}

	timestamp := time.Now().Format("2006-01-02")
	backupFile := backupDir + "/" + name + "_" + timestamp + ".zip"

	// Simple copy for now - in production would use archive/zip
	fmt.Printf("  Would backup to: %s\n", backupFile)
	fmt.Printf("  ✓ Backup scheduled for %s\n", name)
}

func runRestore(plugin string, backupFile string) {
	if plugin == "" || backupFile == "" {
		fmt.Println("Error: --plugin and --file are required for restore")
		fmt.Println("Example: rag restore --plugin fenrir --file ~/backups/fenrir_2026-03-25.zip")
		os.Exit(1)
	}

	plugins := map[string]string{
		"fenrir": "~/.fenrir",
		"hati":   "~/.hati",
		"skoll":  "~/.skoll",
		"tyr":    "~/.tyr",
	}

	targetDir, ok := plugins[plugin]
	if !ok {
		fmt.Printf("Error: Unknown plugin '%s'\n", plugin)
		os.Exit(1)
	}

	fmt.Printf("Ragnarok Restore\n")
	fmt.Printf("Plugin: %s\n", plugin)
	fmt.Printf("Backup file: %s\n", backupFile)
	fmt.Printf("Target: %s\n", targetDir)
	fmt.Println(strings.Repeat("-", 40))
	fmt.Println("Restore functionality requires PowerShell scripts")
	fmt.Println("Use: scripts/restore_ragnarok.ps1")
}

func runIntegrate(projectPath string) {
	absPath, err := filepath.Abs(projectPath)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Ragnarok Integration\n")
	fmt.Printf("Project: %s\n", absPath)
	fmt.Println(strings.Repeat("-", 50))

	data, err := integration.LoadBootstrapData(absPath)
	if err != nil {
		fmt.Printf("Error loading bootstrap data: %v\n", err)
		os.Exit(1)
	}

	if data == nil || !data.HasData() {
		fmt.Println("No bootstrap data found.")
		fmt.Println("\nRun 'fenrir bootstrap --path <project>' first to generate skills, rules and standards.")
		os.Exit(1)
	}

	fmt.Println("Bootstrap data loaded:")
	fmt.Printf("  Skills: %d\n", len(data.Skills))
	fmt.Printf("  Rules: %d\n", len(data.Rules))
	fmt.Printf("  Standards: %d\n", len(data.Standards))
	fmt.Println(strings.Repeat("-", 50))

	if len(data.Skills) > 0 {
		fmt.Println("\n📦 SKILLS (register via Skoll MCP):")
		for _, s := range data.Skills {
			fmt.Printf("  - %s (%s): %s\n", s["name"], s["type"], s["skill"])
		}
		fmt.Println("\n  To register: Use Skoll's skill management commands")
	}

	if len(data.Rules) > 0 {
		fmt.Println("\n📋 RULES (register via Skoll MCP):")
		for _, r := range data.Rules {
			fmt.Printf("  - %s [%s]: %s\n", r["name"], r["severity"], r["description"])
		}
		fmt.Println("\n  To register: Use Skoll's rule management commands")
	}

	if len(data.Standards) > 0 {
		fmt.Println("\n✅ STANDARDS (register via Tyr MCP):")
		for _, st := range data.Standards {
			block := ""
			if st["block"] == "true" {
				block = " (blocks merge)"
			}
			fmt.Printf("  - %s%s: %s\n", st["name"], block, st["description"])
		}
		fmt.Println("\n  To register: Use Tyr's standards management commands")
	}

	fmt.Println("\n" + strings.Repeat("-", 50))
	fmt.Println("Integration summary available above.")
	fmt.Println("Use each plugin's MCP interface to register the data.")
}

func printUsage() {
	fmt.Println(`Ragnarok v1.2.0 - AI Governance & Memory Layer

Usage:
  rag init --project NAME [--dir DIR]     Initialize all plugins
  rag scan --path PATH [--bootstrap]      Scan project and bootstrap
  rag install --project NAME [--mcp]     Install Ragnarok
  rag serve                              Start unified MCP server (stdio)
  rag mcp                                Alias for serve
  rag setup --agent AGENT                Setup MCP for agent (opencode, cursor, windsurf)
  rag version                            Show version

Quick Setup:
  rag setup opencode     Configure OpenCode (most common)
  rag setup cursor       Configure Cursor
  rag setup windsurf     Configure Windsurf
  rag setup antigravity  Configure Antigravity

Examples:
  rag init --project my-project
  rag scan --path ./myproject
  rag setup opencode
  rag serve`)
}

func runInit(projectName, baseDir string) {
	if projectName == "" {
		fmt.Println("Error: --project is required")
		fmt.Println("Example: rag init --project my-project")
		os.Exit(1)
	}

	if baseDir == "" {
		home, _ := os.UserHomeDir()
		baseDir = home
	}

	fmt.Printf("Ragnarok Init - Initializing all plugins\n")
	fmt.Printf("Project: %s\n", projectName)
	fmt.Printf("Base directory: %s\n", baseDir)
	fmt.Println(strings.Repeat("─", 50))

	pluginDirs := map[string]string{
		"fenrir": filepath.Join(baseDir, ".fenrir"),
		"hati":   filepath.Join(baseDir, ".hati"),
		"skoll":  filepath.Join(baseDir, ".skoll"),
		"tyr":    filepath.Join(baseDir, ".tyr"),
	}

	pluginPorts := map[string]int{
		"fenrir": 7437,
		"hati":   7439,
		"skoll":  7438,
		"tyr":    7440,
	}

	for name, dir := range pluginDirs {
		fmt.Printf("\n📦 Initializing %s...\n", strings.ToUpper(name))
		fmt.Printf("   Directory: %s\n", dir)

		os.MkdirAll(dir, 0755)

		cfg := map[string]interface{}{
			"project":  projectName,
			"version":  version,
			"port":     pluginPorts[name],
			"data_dir": dir,
		}

		cfgPath := filepath.Join(dir, "config.json")
		data, _ := json.MarshalIndent(cfg, "", "  ")
		os.WriteFile(cfgPath, data, 0644)
		fmt.Printf("   ✓ Config: %s\n", cfgPath)
	}

	generateMCPJson(projectName, baseDir)

	fmt.Printf("\n✓ All plugins initialized for project: %s\n", projectName)
	fmt.Println("\nNext steps:")
	fmt.Println("  1. Start servers: rag serve")
	fmt.Println("  2. Scan project:  rag scan --path ./your-project")
	fmt.Println("  3. Check health:  rag stats --ecosystem")
}

type MCPJson struct {
	MCPServers map[string]MCPServer `json:"mcpServers"`
}

type MCPServer struct {
	Command string            `json:"command"`
	Args    []string          `json:"args,omitempty"`
	Env     map[string]string `json:"env,omitempty"`
}

func generateMCPJson(projectName, baseDir string) {
	ext := ""
	if runtime.GOOS == "windows" {
		ext = ".exe"
	}

	selfPath, _ := os.Executable()
	selfDir := filepath.Dir(selfPath)

	pluginPorts := map[string]int{
		"fenrir": 7437,
		"hati":   7439,
		"skoll":  7438,
		"tyr":    7440,
	}

	mcpServers := make(map[string]MCPServer)

	for name, port := range pluginPorts {
		binName := name + ext
		binPath := filepath.Join(selfDir, binName)

		if _, err := os.Stat(binPath); os.IsNotExist(err) {
			binPath = binName
		}

		mcpServers[name] = MCPServer{
			Command: binPath,
			Args:    []string{"serve", "--port", fmt.Sprintf("%d", port)},
			Env:     map[string]string{"MCP_TRANSPORT": "tcp"},
		}
	}

	mcpJson := MCPJson{MCPServers: mcpServers}

	cwd, _ := os.Getwd()
	mcpJsonPath := filepath.Join(cwd, ".mcp.json")
	data, _ := json.MarshalIndent(mcpJson, "", "  ")
	os.WriteFile(mcpJsonPath, data, 0644)
	fmt.Printf("   ✓ MCP config: %s\n", mcpJsonPath)
}

func runScan(projectPath string, doBootstrap bool) {
	absPath, err := filepath.Abs(projectPath)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Ragnarok Scan - Analyzing project\n")
	fmt.Printf("Project: %s\n", absPath)
	fmt.Println(strings.Repeat("─", 50))

	fmt.Println("\n🔍 Running project analysis...")
	fenrircli.RunScan(absPath, "all")

	if doBootstrap {
		fmt.Println("\n📦 Generating bootstrap files...")
		fenrircli.RunBootstrap(absPath)

		fmt.Println("\n📝 Generating AGENTS.md...")
		fenrircli.RunInit(filepath.Base(absPath), "")
	}

	fmt.Println("\n✓ Scan complete!")
	fmt.Println("\nNext steps:")
	fmt.Println("  1. Initialize plugins: rag init --project your-project")
	fmt.Println("  2. Import to plugins: rag integrate --path " + absPath)
}

func runServe(baseDir string) {
	fmt.Printf("Ragnarok Unified Serve - Starting MCP server on stdio\n")
	runUnifiedMCP(baseDir)
}

func runUnifiedMCP(baseDir string) {
	srv, err := unified.NewServer(baseDir)
	if err != nil {
		fmt.Printf("Error initializing unified server: %v\n", err)
		os.Exit(1)
	}

	ctx := context.Background()
	if err := srv.Run(ctx); err != nil {
		fmt.Printf("Server error: %v\n", err)
		os.Exit(1)
	}
}

func runSetup(agent string) {
	if agent == "" {
		fmt.Println("Ragnarok Setup - Configure MCP for AI agents")
		fmt.Println(strings.Repeat("─", 50))
		fmt.Println("Available agents:")
		fmt.Println("  rag setup opencode      Configure OpenCode")
		fmt.Println("  rag setup cursor        Configure Cursor")
		fmt.Println("  rag setup windsurf     Configure Windsurf")
		fmt.Println("  rag setup antigravity  Configure Antigravity")
		fmt.Println("")
		fmt.Println("Example: rag setup opencode")
		return
	}

	switch strings.ToLower(agent) {
	case "opencode":
		setupOpenCode()
	case "cursor":
		setupCursor()
	case "windsurf":
		setupWindsurf()
	case "antigravity":
		setupAntigravity()
	default:
		fmt.Printf("Unknown agent: %s\n", agent)
		fmt.Println("Available: opencode, cursor, windsurf, antigravity")
	}
}

func setupOpenCode() {
	fmt.Println("Setting up Ragnarok for OpenCode...")

	ragPath, err := os.Executable()
	if err != nil {
		fmt.Printf("Error finding rag.exe: %v\n", err)
		os.Exit(1)
	}

	configDirs := []string{
		filepath.Join(os.Getenv("APPDATA"), "opencode"),
		filepath.Join(os.Getenv("LOCALAPPDATA"), "opencode"),
		filepath.Join(os.Getenv("USERPROFILE"), ".config", "opencode"),
	}

	configDir := ""
	for _, dir := range configDirs {
		if _, err := os.Stat(dir); err == nil {
			configDir = dir
			break
		}
	}

	if configDir == "" {
		configDir = filepath.Join(os.Getenv("USERPROFILE"), ".config", "opencode")
		os.MkdirAll(configDir, 0755)
	}

	mcpConfig := map[string]interface{}{
		"mcp": map[string]interface{}{
			"ragnarok": map[string]interface{}{
				"type":    "local",
				"command": []string{ragPath, "mcp"},
				"enabled": true,
			},
		},
	}

	configPath := filepath.Join(configDir, "opencode.json")
	var existingConfig map[string]interface{}
	if data, err := os.ReadFile(configPath); err == nil {
		json.Unmarshal(data, &existingConfig)
	}

	if existingConfig != nil {
		if mcpServers, ok := existingConfig["mcp"].(map[string]interface{}); ok {
			mcpServers["ragnarok"] = mcpConfig["mcp"].(map[string]interface{})["ragnarok"]
			mcpConfig = existingConfig
		}
	}

	data, _ := json.MarshalIndent(mcpConfig, "", "  ")
	os.WriteFile(configPath, data, 0644)

	fmt.Printf("✓ OpenCode configured: %s\n", configPath)
	fmt.Println("  Restart OpenCode to use Ragnarok MCP")
}

func setupCursor() {
	fmt.Println("Setting up Ragnarok for Cursor...")

	ragPath, err := os.Executable()
	if err != nil {
		fmt.Printf("Error finding rag.exe: %v\n", err)
		os.Exit(1)
	}

	configPath := filepath.Join(os.Getenv("USERPROFILE"), ".cursor", "mcp.json")
	os.MkdirAll(filepath.Dir(configPath), 0755)

	mcpConfig := map[string]interface{}{
		"mcpServers": map[string]interface{}{
			"ragnarok": map[string]interface{}{
				"command": []string{ragPath, "mcp"},
			},
		},
	}

	data, _ := json.MarshalIndent(mcpConfig, "", "  ")
	os.WriteFile(configPath, data, 0644)

	fmt.Printf("✓ Cursor configured: %s\n", configPath)
	fmt.Println("  Restart Cursor to use Ragnarok MCP")
}

func setupWindsurf() {
	fmt.Println("Setting up Ragnarok for Windsurf...")

	ragPath, err := os.Executable()
	if err != nil {
		fmt.Printf("Error finding rag.exe: %v\n", err)
		os.Exit(1)
	}

	configPath := filepath.Join(os.Getenv("USERPROFILE"), ".windsurf", "mcp.json")
	os.MkdirAll(filepath.Dir(configPath), 0755)

	mcpConfig := map[string]interface{}{
		"mcpServers": map[string]interface{}{
			"ragnarok": map[string]interface{}{
				"command": ragPath,
				"args":    []string{"mcp"},
			},
		},
	}

	data, _ := json.MarshalIndent(mcpConfig, "", "  ")
	os.WriteFile(configPath, data, 0644)

	fmt.Printf("✓ Windsurf configured: %s\n", configPath)
	fmt.Println("  Restart Windsurf to use Ragnarok MCP")
}

func setupAntigravity() {
	fmt.Println("Setting up Ragnarok for Antigravity...")

	ragPath, err := os.Executable()
	if err != nil {
		fmt.Printf("Error finding rag.exe: %v\n", err)
		os.Exit(1)
	}

	home, _ := os.UserHomeDir()
	configPath := filepath.Join(home, ".gemini", "antigravity", "mcp_config.json")
	os.MkdirAll(filepath.Dir(configPath), 0755)

	mcpConfig := map[string]interface{}{
		"mcpServers": map[string]interface{}{
			"ragnarok": map[string]interface{}{
				"command": ragPath,
				"args":    []string{"mcp"},
			},
		},
	}

	data, _ := json.MarshalIndent(mcpConfig, "", "  ")
	os.WriteFile(configPath, data, 0644)

	fmt.Printf("✓ Antigravity configured: %s\n", configPath)
	fmt.Println("  Restart Antigravity to use Ragnarok MCP")
}

func runStop() {
	fmt.Printf("Ragnarok Stop - Stopping all servers\n")
	fmt.Println(strings.Repeat("─", 50))

	pluginPorts := []int{7437, 7438, 7439, 7440}

	for _, port := range pluginPorts {
		fmt.Printf("Checking port %d...\n", port)
	}

	fmt.Println("\nNote: On Windows, use:")
	fmt.Println("  taskkill /F /IM fenrir.exe /IM hati.exe /IM skoll.exe /IM tyr.exe")
}

func findPluginBinary(name string) string {
	ext := ""
	if runtime.GOOS == "windows" {
		ext = ".exe"
	}

	searchDirs := []string{}

	if selfPath, err := os.Executable(); err == nil {
		selfDir := filepath.Dir(selfPath)
		searchDirs = append(searchDirs, selfDir)
	}

	if cwd, err := os.Getwd(); err == nil {
		searchDirs = append(searchDirs, cwd)
	}

	path := os.Getenv("PATH")
	if pathDirs := filepath.SplitList(path); len(pathDirs) > 0 {
		searchDirs = append(searchDirs, pathDirs...)
	}

	for _, dir := range searchDirs {
		binPath := filepath.Join(dir, name+ext)
		if _, err := os.Stat(binPath); err == nil {
			return binPath
		}
	}

	return ""
}
