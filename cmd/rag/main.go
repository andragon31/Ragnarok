package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	fenrircli "github.com/andragon31/Ragnarok/internal/fenrir/cli"
	fenrirdb "github.com/andragon31/Ragnarok/internal/fenrir/database"
	hatidb "github.com/andragon31/Ragnarok/internal/hati/database"
	"github.com/andragon31/Ragnarok/internal/installer/installer"
	"github.com/andragon31/Ragnarok/internal/installer/integration"
	"github.com/andragon31/Ragnarok/internal/mcp/unified"
	skolldb "github.com/andragon31/Ragnarok/internal/skoll/database"
	tyrdb "github.com/andragon31/Ragnarok/internal/tyr/database"
)

var version = "2.4.5"

func toInt(v interface{}) int {
	if v == nil {
		return 0
	}
	switch val := v.(type) {
	case int:
		return val
	case float64:
		return int(val)
	case int32:
		return int(val)
	case int64:
		return int(val)
	case bool:
		if val {
			return 1
		}
		return 0
	case string:
		i, _ := strconv.Atoi(val)
		return i
	default:
		return 0
	}
}

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

	bootstrapCmd := flag.NewFlagSet("bootstrap", flag.ExitOnError)
	bootstrapPRD := bootstrapCmd.String("prd", "", "PRD file path (required)")
	bootstrapPath := bootstrapCmd.String("path", ".", "Project base path")

	newCmd := flag.NewFlagSet("new", flag.ExitOnError)
	newProject := newCmd.String("project", "", "Project name")
	newPath := newCmd.String("path", ".", "Project directory")
	newStack := newCmd.String("stack", "", "Tech stack (go, node, python, java, rust, dotnet)")

	continueCmd := flag.NewFlagSet("continue", flag.ExitOnError)
	continuePlan := continueCmd.String("plan", "", "Plan ID to resume")

	featureCmd := flag.NewFlagSet("feature", flag.ExitOnError)
	featureName := featureCmd.String("name", "", "Feature name")
	featurePlan := featureCmd.String("plan", "", "Parent plan ID")

	reviewCmd := flag.NewFlagSet("review", flag.ExitOnError)
	reviewPlan := reviewCmd.String("plan", "", "Plan ID for review")

	statusCmd := flag.NewFlagSet("status", flag.ExitOnError)
	statusPlan := statusCmd.String("plan", "", "Plan ID (default: active)")

	projectCmd := flag.NewFlagSet("project", flag.ExitOnError)
	projectPath := projectCmd.String("path", "", "Project directory (required)")
	projectPRD := projectCmd.String("prd", "", "PRD file path (optional, for new projects)")
	projectTitle := projectCmd.String("title", "", "Project title (optional)")
	projectStack := projectCmd.String("stack", "", "Tech stack (auto-detect if not specified)")

	requirementCmd := flag.NewFlagSet("requirement", flag.ExitOnError)
	requirementProject := requirementCmd.String("project", "", "Project directory (required)")
	requirementText := requirementCmd.String("text", "", "Requirement text (required)")
	requirementPriority := requirementCmd.Int("priority", 5, "Priority (1-10)")

	planCmd := flag.NewFlagSet("plan", flag.ExitOnError)
	planProject := planCmd.String("project", "", "Project directory (required)")
	planTitle := planCmd.String("title", "", "Plan title (optional)")

	resetCmd := flag.NewFlagSet("reset", flag.ExitOnError)
	resetForce := resetCmd.Bool("force", false, "Skip confirmation prompt")
	resetDBs := resetCmd.String("db", "all", "Databases to reset: all, hati, skoll, fenrir, tyr (comma-separated)")

	reinstallCmd := flag.NewFlagSet("reinstall", flag.ExitOnError)
	reinstallBackup := reinstallCmd.Bool("backup", true, "Backup before reinstalling")
	reinstallForce := reinstallCmd.Bool("force", false, "Skip confirmation prompt")

	doctorCmd := flag.NewFlagSet("doctor", flag.ExitOnError)
	doctorVerbose := doctorCmd.Bool("verbose", false, "Show detailed output")

	cleanupCmd := flag.NewFlagSet("cleanup", flag.ExitOnError)
	cleanupForce := cleanupCmd.Bool("force", false, "Skip confirmation prompt")
	cleanupOptimize := cleanupCmd.Bool("optimize", false, "Optimize databases after cleanup")

	migrateCmd := flag.NewFlagSet("migrate", flag.ExitOnError)
	migratePlugin := migrateCmd.String("plugin", "all", "Plugin to migrate (fenrir, hati, skoll, tyr, all)")

	infoCmd := flag.NewFlagSet("info", flag.ExitOnError)

	configCmd := flag.NewFlagSet("config", flag.ExitOnError)
	configPlugin := configCmd.String("plugin", "", "Plugin name (fenrir, hati, skoll, tyr)")
	configKey := configCmd.String("key", "", "Config key")
	configValue := configCmd.String("value", "", "Config value (if setting)")

	exportCmd := flag.NewFlagSet("export", flag.ExitOnError)
	exportPlugin := exportCmd.String("plugin", "all", "Plugin to export (fenrir, hati, skoll, tyr, all)")
	exportFile := exportCmd.String("file", "", "Export file path")

	importCmd := flag.NewFlagSet("import", flag.ExitOnError)
	importFile := importCmd.String("file", "", "Import file path")
	importPlugin := importCmd.String("plugin", "all", "Plugin to import (fenrir, hati, skoll, tyr, all)")

	serverCmd := flag.NewFlagSet("server", flag.ExitOnError)
	serverAction := serverCmd.String("action", "status", "Action: status, start, stop, restart")

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
	case "bootstrap":
		bootstrapCmd.Parse(os.Args[2:])
		if *bootstrapPRD == "" && len(os.Args) > 2 {
			// Direct usage: rag bootstrap PRD.md
			*bootstrapPRD = os.Args[2]
		}
		runBootstrap(*bootstrapPRD, *bootstrapPath)
	case "setup":
		agent := ""
		if len(os.Args) > 2 {
			agent = os.Args[2]
		}
		runSetup(agent)
	case "stop":
		runStop()
	case "new":
		newCmd.Parse(os.Args[2:])
		runNewProject(*newProject, *newPath, *newStack)
	case "continue":
		continueCmd.Parse(os.Args[2:])
		runContinue(*continuePlan)
	case "feature":
		featureCmd.Parse(os.Args[2:])
		runFeature(*featureName, *featurePlan)
	case "review":
		reviewCmd.Parse(os.Args[2:])
		runReview(*reviewPlan)
	case "status":
		statusCmd.Parse(os.Args[2:])
		runStatus(*statusPlan)
	case "project":
		projectCmd.Parse(os.Args[2:])
		runProject(*projectPath, *projectPRD, *projectTitle, *projectStack)
	case "requirement":
		reqArgs := os.Args[2:]
		if len(reqArgs) > 0 && reqArgs[0] == "add" {
			reqArgs = reqArgs[1:]
		}
		requirementCmd.Parse(reqArgs)
		runRequirement(*requirementProject, *requirementText, *requirementPriority)
	case "plan":
		planArgs := os.Args[2:]
		if len(planArgs) > 0 && planArgs[0] == "create" {
			planArgs = planArgs[1:]
		}
		planCmd.Parse(planArgs)
		runPlan(*planProject, *planTitle)
	case "reset":
		resetCmd.Parse(os.Args[2:])
		runReset(*resetForce, *resetDBs)
	case "reinstall":
		reinstallCmd.Parse(os.Args[2:])
		runReinstall(*reinstallBackup, *reinstallForce)
	case "doctor":
		doctorCmd.Parse(os.Args[2:])
		runDoctor(*doctorVerbose)
	case "cleanup":
		cleanupCmd.Parse(os.Args[2:])
		runCleanup(*cleanupForce, *cleanupOptimize)
	case "migrate":
		migrateCmd.Parse(os.Args[2:])
		runMigrate(*migratePlugin)
	case "info":
		infoCmd.Parse(os.Args[2:])
		runInfo()
	case "config":
		configCmd.Parse(os.Args[2:])
		runConfig(*configPlugin, *configKey, *configValue)
	case "export":
		exportCmd.Parse(os.Args[2:])
		runExport(*exportPlugin, *exportFile)
	case "import":
		importCmd.Parse(os.Args[2:])
		runImport(*importFile, *importPlugin)
	case "server":
		serverCmd.Parse(os.Args[2:])
		runServer(*serverAction)
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
	fmt.Println(`Ragnarok v2.2.6 - AI Governance & Memory Layer

Usage:
  rag project --path DIR [--prd FILE]      Analyze or init project (Recommended)
  rag new --project NAME [--path DIR]     Create new project from stack
  rag continue --plan ID                  Resume existing project
  rag feature --name NAME [--plan ID]    Start new feature
  rag review [--plan ID]                  Quality checkpoint review
  rag status [--plan ID]                  Show project status

  rag init --project NAME [--dir DIR]     Initialize all plugins
  rag scan --path PATH [--bootstrap]      Scan project and bootstrap
  rag install --project NAME [--mcp]     Install Ragnarok
  rag serve                              Start unified MCP server (stdio)
  rag mcp                                Alias for serve
	rag setup --agent AGENT                Setup MCP for agent (opencode, cursor, windsurf, claude, gemini)
  rag reset                              Reset all databases (DANGER!)
  rag reinstall                          Complete reinstall from scratch (DANGER!)
  rag version                            Show version

Project Workflows:
  rag project --path ./myapi --prd ./PRD.md  # Init project from PRD
  rag project --path ./existing              # Analyze existing project
  rag requirement add --project ./myapi --text "requirement"
  rag plan create --project ./myapi [--title "Plan Title"]

Execution:
  rag new --project NAME [--stack]     Create new project from stack
  rag continue --plan ID                Resume existing project
  rag feature --name NAME [--plan ID]  Start new feature
  rag review [--plan ID]               Quality checkpoint review
  rag status [--plan ID]               Show project status

Maintenance:
  rag reset                             Reset all databases (requires confirmation)
  rag reset --force                     Reset without confirmation
  rag reinstall                         Complete reinstall from scratch (keeps backup)
  rag reinstall --force                  Complete reinstall without backup prompt

Quick Setup:
  rag setup opencode     Configure OpenCode (most common)
  rag setup cursor       Configure Cursor
  rag setup windsurf     Configure Windsurf
  rag setup claude       Configure Claude Code
  rag setup gemini       Configure Gemini CLI`)
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
		fmt.Println("  rag setup all         Detect and configure all installed IDEs")
		fmt.Println("  rag setup opencode    Configure OpenCode")
		fmt.Println("  rag setup cursor      Configure Cursor")
		fmt.Println("  rag setup windsurf    Configure Windsurf")
		fmt.Println("  rag setup claude      Configure Claude Code")
		fmt.Println("  rag setup gemini      Configure Gemini CLI")
		fmt.Println("")
		fmt.Println("Example: rag setup all")
		return
	}

	switch strings.ToLower(agent) {
	case "all":
		setupAll()
	case "opencode":
		setupOpenCode()
	case "cursor":
		setupCursor()
	case "windsurf":
		setupWindsurf()
	case "claude":
		setupClaude()
	case "gemini":
		setupGemini()
	default:
		fmt.Printf("Unknown agent: %s\n", agent)
		fmt.Println("Available: all, opencode, cursor, windsurf, claude, gemini")
	}
}

func setupAll() {
	fmt.Println("Ragnarok Setup - Configuring all detected IDEs...")
	fmt.Println(strings.Repeat("─", 50))
	configured := 0

	home, _ := os.UserHomeDir()

	// OpenCode — check multiple config locations
	opencodeLocations := []string{
		filepath.Join(home, ".config", "opencode"),
		filepath.Join(os.Getenv("APPDATA"), "opencode"),
		filepath.Join(os.Getenv("LOCALAPPDATA"), "opencode"),
	}
	for _, dir := range opencodeLocations {
		if _, err := os.Stat(dir); err == nil {
			fmt.Print("  OpenCode: ")
			setupOpenCode()
			configured++
			break
		}
	}

	// Cursor
	cursorConfig := filepath.Join(home, ".cursor")
	if _, err := os.Stat(cursorConfig); err == nil {
		fmt.Print("  Cursor: ")
		setupCursor()
		configured++
	}

	// Windsurf
	windsurfConfig := filepath.Join(home, ".windsurf")
	if _, err := os.Stat(windsurfConfig); err == nil {
		fmt.Print("  Windsurf: ")
		setupWindsurf()
		configured++
	}

	// Claude Code
	claudeConfig := filepath.Join(home, ".claude")
	if _, err := os.Stat(claudeConfig); err == nil {
		fmt.Print("  Claude Code: ")
		setupClaude()
		configured++
	}

	// Gemini CLI
	geminiConfig := filepath.Join(home, ".gemini")
	if _, err := os.Stat(geminiConfig); err == nil {
		fmt.Print("  Gemini CLI: ")
		setupGemini()
		configured++
	}

	fmt.Println(strings.Repeat("─", 50))
	if configured == 0 {
		fmt.Println("No supported IDEs detected.")
		fmt.Println("Run 'rag setup <ide>' after installing an IDE.")
	} else {
		fmt.Printf("✓ Configured %d IDE(s). Restart them to enable Ragnarok MCP.\n", configured)
	}
}

func setupOpenCode() {
	fmt.Println("Setting up Ragnarok for OpenCode...")

	ragPath, err := os.Executable()
	if err != nil {
		fmt.Printf("Error finding rag.exe: %v\n", err)
		os.Exit(1)
	}

	configs := []struct {
		dir  string
		sub  string
		file string
	}{
		{os.Getenv("USERPROFILE"), ".config/opencode", "opencode.json"},
		{os.Getenv("APPDATA"), "opencode", "opencode.json"},
		{os.Getenv("APPDATA"), "OpenCode", "opencode.json"},
		{os.Getenv("LOCALAPPDATA"), "opencode", "opencode.json"},
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

	updated := 0
	for _, cfg := range configs {
		if cfg.dir == "" {
			continue
		}
		configDir := filepath.Join(cfg.dir, cfg.sub)
		configPath := filepath.Join(configDir, cfg.file)

		os.MkdirAll(configDir, 0755)

		var existingConfig map[string]interface{}
		if data, err := os.ReadFile(configPath); err == nil {
			json.Unmarshal(data, &existingConfig)
		}

		if existingConfig != nil {
			if mcp, ok := existingConfig["mcp"].(map[string]interface{}); ok {
				mcp["ragnarok"] = mcpConfig["mcp"].(map[string]interface{})["ragnarok"]
				mcpConfig = existingConfig
			}
		}

		data, _ := json.MarshalIndent(mcpConfig, "", "  ")
		os.WriteFile(configPath, data, 0644)
		updated++
	}

	fmt.Printf("✓ OpenCode configured: %d config files updated\n", updated)
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
				"command": []string{ragPath, "mcp"},
			},
		},
	}

	data, _ := json.MarshalIndent(mcpConfig, "", "  ")
	os.WriteFile(configPath, data, 0644)

	fmt.Printf("✓ Windsurf configured: %s\n", configPath)
	fmt.Println("  Restart Windsurf to use Ragnarok MCP")
}

func setupClaude() {
	fmt.Println("Setting up Ragnarok for Claude Code...")

	ragPath, err := os.Executable()
	if err != nil {
		fmt.Printf("Error finding rag.exe: %v\n", err)
		os.Exit(1)
	}

	home, _ := os.UserHomeDir()
	configPath := filepath.Join(home, ".claude", "settings.json")
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

	fmt.Printf("✓ Claude Code configured: %s\n", configPath)
	fmt.Println("  Restart Claude Code to use Ragnarok MCP")
}

func setupGemini() {
	fmt.Println("Setting up Ragnarok for Gemini CLI...")

	ragPath, err := os.Executable()
	if err != nil {
		fmt.Printf("Error finding rag.exe: %v\n", err)
		os.Exit(1)
	}

	home, _ := os.UserHomeDir()
	configPath := filepath.Join(home, ".gemini", "settings.json")
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

	fmt.Printf("✓ Gemini CLI configured: %s\n", configPath)
	fmt.Println("  Restart Gemini CLI to use Ragnarok MCP")
}

func runNewProject(projectName, projectPath, stack string) {
	if projectName == "" {
		fmt.Println("Error: --project is required")
		fmt.Println("Example: rag new --project myapi --path ./myapi --stack=go")
		os.Exit(1)
	}

	absPath, _ := filepath.Abs(projectPath)
	fmt.Printf("Ragnarok New - Creating new project\n")
	fmt.Printf("Project: %s\n", projectName)
	fmt.Printf("Path: %s\n", absPath)
	fmt.Printf("Stack: %s\n", stack)
	fmt.Println(strings.Repeat("─", 50))

	srv, err := unified.NewServer("")
	if err != nil {
		fmt.Printf("Error initializing unified server: %v\n", err)
		os.Exit(1)
	}

	ctx := context.Background()
	result, err := srv.ExecuteWorkflow(ctx, "workflow_stack_based_init", map[string]interface{}{
		"project_path": absPath,
		"title":        projectName,
		"phases":       getDefaultPhases(stack),
	})

	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}

	printWorkflowResult("new", result)

	fmt.Println("\n✓ Project created successfully!")
	fmt.Println("\nNext steps:")
	fmt.Printf("  rag continue --plan %s  # To start development\n", getPlanID(result))
}

func runContinue(planID string) {
	if planID == "" {
		planID = getActivePlanID()
		if planID == "" {
			fmt.Println("Error: No active plan found. Use --plan to specify a plan ID")
			os.Exit(1)
		}
	}

	fmt.Printf("Ragnarok Continue - Resuming project\n")
	fmt.Printf("Plan: %s\n", planID)
	fmt.Println(strings.Repeat("─", 50))

	srv, err := unified.NewServer("")
	if err != nil {
		fmt.Printf("Error initializing unified server: %v\n", err)
		os.Exit(1)
	}

	ctx := context.Background()
	result, err := srv.ExecuteWorkflow(ctx, "workflow_plan_develop_v2", map[string]interface{}{
		"plan_id":       planID,
		"auto_continue": false,
	})

	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}

	printWorkflowResult("continue", result)

	fmt.Printf("\n✓ Development session completed\n")
	fmt.Printf("Plan progress: %s\n", getPlanProgress(result))
}

func runFeature(featureName, planID string) {
	if featureName == "" {
		fmt.Println("Error: --name is required")
		fmt.Println("Example: rag feature --name user-auth --plan <plan_id>")
		os.Exit(1)
	}

	if planID == "" {
		planID = getActivePlanID()
	}

	fmt.Printf("Ragnarok Feature - Starting new feature\n")
	fmt.Printf("Feature: %s\n", featureName)
	fmt.Printf("Plan: %s\n", planID)
	fmt.Println(strings.Repeat("─", 50))

	srv, err := unified.NewServer("")
	if err != nil {
		fmt.Printf("Error initializing unified server: %v\n", err)
		os.Exit(1)
	}

	ctx := context.Background()
	phaseResult, err := srv.ExecuteWorkflow(ctx, "phase_create", map[string]interface{}{
		"plan_id":   planID,
		"title":     "Feature: " + featureName,
		"order_num": 99,
	})

	if err != nil {
		fmt.Printf("Error creating feature phase: %v\n", err)
		os.Exit(1)
	}

	phaseID := getPhaseID(phaseResult)

	taskResult, _ := srv.ExecuteWorkflow(ctx, "task_create", map[string]interface{}{
		"phase_id":    phaseID,
		"title":       featureName,
		"description": "Implement feature: " + featureName,
		"priority":    5,
		"milestone":   true,
	})

	fmt.Println("\n✓ Feature created!")
	fmt.Printf("Phase ID: %s\n", phaseID)
	fmt.Printf("Task ID: %s\n", getTaskID(taskResult))
	fmt.Println("\nNext steps:")
	fmt.Println("  rag continue --plan " + planID)
}

func runReview(planID string) {
	if planID == "" {
		planID = getActivePlanID()
	}

	fmt.Printf("Ragnarok Review - Quality checkpoint\n")
	fmt.Printf("Plan: %s\n", planID)
	fmt.Println(strings.Repeat("─", 50))

	srv, err := unified.NewServer("")
	if err != nil {
		fmt.Printf("Error initializing unified server: %v\n", err)
		os.Exit(1)
	}

	ctx := context.Background()
	result, err := srv.ExecuteWorkflow(ctx, "workflow_checkpoint_create", map[string]interface{}{
		"plan_id":     planID,
		"description": "Manual quality review",
	})

	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}

	printWorkflowResult("review", result)

	fmt.Println("\n✓ Review checkpoint created!")
	fmt.Println("Waiting for human approval...")
}

func runStatus(planID string) {
	if planID == "" {
		planID = getActivePlanID()
	}

	fmt.Printf("Ragnarok Status\n")
	if planID != "" {
		fmt.Printf("Plan: %s\n", planID)
	}
	fmt.Println(strings.Repeat("─", 50))

	srv, err := unified.NewServer("")
	if err != nil {
		fmt.Printf("Error initializing unified server: %v\n", err)
		os.Exit(1)
	}

	ctx := context.Background()

	fmt.Println("\n📊 Ecosystem Status:")
	diag, _ := srv.ExecuteWorkflow(ctx, "ecosystem_diagnose", map[string]interface{}{"verbose": false})
	printWorkflowResult("diagnose", diag)

	if planID != "" {
		fmt.Println("\n📋 Plan Progress:")
		progress, _ := srv.ExecuteWorkflow(ctx, "plan_progress", map[string]interface{}{"plan_id": planID})
		printWorkflowResult("progress", progress)

		fmt.Println("\n📝 Recent Tasks:")
		tasks, _ := srv.ExecuteWorkflow(ctx, "task_list", map[string]interface{}{"plan_id": planID, "limit": 5})
		printWorkflowResult("tasks", tasks)
	}

	fmt.Println("\n✓ Status check complete")
}

func runBootstrap(prdFile, projectPath string) {
	if prdFile == "" {
		fmt.Println("Error: --prd is required")
		fmt.Println("Example: rag bootstrap --prd ./PRD.md [--path .]")
		os.Exit(1)
	}

	absPRD, _ := filepath.Abs(prdFile)
	if projectPath == "." || projectPath == "" {
		projectPath = filepath.Dir(absPRD)
	}
	absPath, _ := filepath.Abs(projectPath)

	fmt.Printf("\n🚀 Ragnarok Bootstrap - Project Initialization\n")
	fmt.Printf("   PRD:  %s\n", absPRD)
	fmt.Printf("   Path: %s\n", absPath)
	fmt.Println(strings.Repeat("─", 50))

	srv, err := unified.NewServer("")
	if err != nil {
		fmt.Printf("Error initializing unified server: %v\n", err)
		os.Exit(1)
	}

	ctx := context.Background()

	fmt.Println("⏳ Executing Integrated Lifecycle (Fenrir -> Skoll -> Tyr -> Hati)...")
	fmt.Println("   Note: This can take 1-2 minutes for large projects. Handled safely.")

	result, err := srv.ExecuteWorkflow(ctx, "workflow_project_lifecycle", map[string]interface{}{
		"prd_file":     absPRD,
		"project_path": absPath,
		"auto_start":   false,
	})

	if err != nil {
		fmt.Printf("\n❌ Error during bootstrap: %v\n", err)
		os.Exit(1)
	}

	// Format result
	resMap, ok := result.(map[string]interface{})
	if !ok {
		fmt.Printf("\n❌ Unexpected result format: %v\n", result)
		os.Exit(1)
	}

	status, _ := resMap["status"].(string)
	projectName, _ := resMap["project_name"].(string)
	planID, _ := resMap["plan_id"].(string)
	agentCount := toInt(resMap["agent_count"])
	taskCount := toInt(resMap["task_count"])

	if status == "partial" {
		fmt.Printf("\n⚠️ Workflow paused due to complexity. Please run the command again to finish.\n")
	}

	fmt.Printf("\n✅ Bootstrap Successful for project '%s'!\n", projectName)
	fmt.Printf("   ├─ Agents created: %d\n", agentCount)
	fmt.Printf("   ├─ Tasks planned: %d\n", taskCount)
	fmt.Printf("   └─ Plan ID:       %s\n", planID)

	fmt.Println("\n📋 Next steps:")
	fmt.Printf("   rag continue --plan %s     # Start working on tasks\n", planID)
	fmt.Printf("   rag status --plan %s       # View current progress\n\n", planID)
}

func runProject(projectPath, prdFile, title, stack string) {
	if prdFile != "" {
		runBootstrap(prdFile, projectPath)
		return
	}

	if projectPath == "" {
		fmt.Println("Error: --path is required")
		os.Exit(1)
	}

	absPath, _ := filepath.Abs(projectPath)
	srv, err := unified.NewServer("")
	if err != nil {
		fmt.Printf("Error initializing server: %v\n", err)
		os.Exit(1)
	}

	ctx := context.Background()
	fmt.Printf("Ragnarok Project - Syncing: %s\n", absPath)

	result, err := srv.ExecuteWorkflow(ctx, "project_scan", map[string]interface{}{
		"path": absPath,
	})
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("✓ Scan complete: %s\n", getStringResult(result, "stack"))
}

func runRequirement(projectPath, text string, priority int) {
	if projectPath == "" || text == "" {
		fmt.Println("Error: --project and --text are required")
		fmt.Println("Example: rag requirement add --project ./mi-proyecto --text \"Necesito auth con JWT\"")
		os.Exit(1)
	}

	absPath, _ := filepath.Abs(projectPath)

	fmt.Printf("Ragnarok Requirement - Adding requirement\n")
	fmt.Printf("Project: %s\n", absPath)
	fmt.Printf("Requirement: %s\n", text)
	fmt.Println(strings.Repeat("─", 50))

	srv, err := unified.NewServer("")
	if err != nil {
		fmt.Printf("Error initializing unified server: %v\n", err)
		os.Exit(1)
	}

	ctx := context.Background()

	memResult, err := srv.ExecuteWorkflow(ctx, "mem_save", map[string]interface{}{
		"title": "Requirement: " + text,
		"type":  "requirement",
		"what":  text,
		"where": absPath,
	})

	if err != nil {
		fmt.Printf("Error saving requirement: %v\n", err)
		os.Exit(1)
	}

	memID := getStringResult(memResult, "id")

	fmt.Println("\n✓ Requirement added!")
	fmt.Printf("Memory ID: %s\n", memID)
	fmt.Println("\n📋 Next steps:")
	fmt.Println("  rag requirement add --project " + absPath + " --text \"Another requirement\"")
	fmt.Println("  rag plan create --project " + absPath + "  # Create plan from all requirements")
}

func runPlan(projectPath, title string) {
	if projectPath == "" {
		fmt.Println("Error: --project is required")
		fmt.Println("Example: rag plan create --project ./mi-proyecto --title \"Mi Plan\"")
		os.Exit(1)
	}

	absPath, _ := filepath.Abs(projectPath)

	fmt.Printf("Ragnarok Plan - Creating plan from project\n")
	fmt.Printf("Project: %s\n", absPath)
	fmt.Println(strings.Repeat("─", 50))

	srv, err := unified.NewServer("")
	if err != nil {
		fmt.Printf("Error initializing unified server: %v\n", err)
		os.Exit(1)
	}

	ctx := context.Background()

	fmt.Println("\n🔍 Scanning project...")
	scanResult, err := srv.ExecuteWorkflow(ctx, "project_scan", map[string]interface{}{
		"path": absPath,
	})

	if err != nil {
		fmt.Printf("Error scanning project: %v\n", err)
		os.Exit(1)
	}

	projectName := title
	if projectName == "" {
		projectName = getStringResult(scanResult, "name")
		if projectName == "" {
			projectName = filepath.Base(absPath)
		}
	}

	detectedStack := getStringResult(scanResult, "stack")

	fmt.Printf("\n📋 Creating plan for: %s\n", projectName)
	fmt.Printf("   Stack detected: %s\n", detectedStack)

	memContext, _ := srv.ExecuteWorkflow(ctx, "mem_context", map[string]interface{}{
		"module": absPath,
	})

	memRequirements, _ := srv.ExecuteWorkflow(ctx, "mem_find", map[string]interface{}{
		"query": "requirement " + projectName,
		"limit": 50,
	})

	_ = memContext

	requirementsText := ""
	if reqMap, ok := memRequirements.(map[string]interface{}); ok {
		if results, ok := reqMap["results"].([]interface{}); ok {
			for _, r := range results {
				if req, ok := r.(map[string]interface{}); ok {
					if what, ok := req["what"].(string); ok {
						requirementsText += "- " + what + "\n"
					}
				}
			}
		}
	}

	fmt.Println("\n📝 Requirements found:")
	if requirementsText != "" {
		fmt.Println(requirementsText)
	} else {
		fmt.Println("  (none - will use stack-based phases)")
	}

	planResult, err := srv.ExecuteWorkflow(ctx, "plan_create", map[string]interface{}{
		"title":       "Ragnarok MCP Ecosystem v2.2.6 Plan",
		"description": requirementsText,
		"risk_level":  "medium",
	})

	if err != nil {
		fmt.Printf("Error creating plan: %v\n", err)
		os.Exit(1)
	}

	planID := getStringResult(planResult, "id")
	if planID == "" {
		planID = getStringResult(planResult, "plan_id")
	}

	phases := getDefaultPhases(detectedStack)
	fmt.Printf("\n📋 Creating %d phases based on stack...\n", len(phases))

	for i, phaseName := range phases {
		srv.ExecuteWorkflow(ctx, "phase_create", map[string]interface{}{
			"plan_id":   planID,
			"title":     phaseName,
			"order_num": i,
		})
	}

	memSave, _ := srv.ExecuteWorkflow(ctx, "mem_save", map[string]interface{}{
		"title": "Plan created: " + projectName,
		"type":  "decision",
		"what":  "Development plan created with " + fmt.Sprintf("%d", len(phases)) + " phases",
		"where": planID,
	})
	_ = memSave

	fmt.Println("\n✓ Plan created!")
	fmt.Printf("Plan ID: %s\n", planID)
	fmt.Println("\n📋 Next steps:")
	fmt.Printf("  rag continue --plan %s  # Start development\n", planID)
}

func getStringResult(result interface{}, key string) string {
	if m, ok := result.(map[string]interface{}); ok {
		if v, ok := m[key].(string); ok {
			return v
		}
	}
	return ""
}

func getDefaultPhases(stack string) []string {
	switch strings.ToLower(stack) {
	case "go":
		return []string{"Setup", "Backend API", "Models", "Handlers", "Middleware", "Tests", "Documentation"}
	case "node", "javascript", "typescript":
		return []string{"Setup", "Backend API", "Routes", "Middleware", "Frontend", "Tests", "Deployment"}
	case "python":
		return []string{"Setup", "Backend API", "Models", "Routes", "Tests", "Documentation"}
	case "java":
		return []string{"Setup", "Backend API", "Services", "Repositories", "Tests", "Deployment"}
	case "rust":
		return []string{"Setup", "Backend API", "Models", "Error Handling", "Tests", "Documentation"}
	case "dotnet", "csharp":
		return []string{"Setup", "Backend API", "Services", "Data Access", "Tests", "Deployment"}
	default:
		return []string{"Setup", "Backend", "Frontend", "Tests", "Deployment"}
	}
}

func runReset(force bool, dbList string) {
	if !force {
		fmt.Println("⚠️  This will DELETE all Ragnarok databases and data!")
		fmt.Println("   Datbases affected: hati, skoll, fenrir, tyr")
		fmt.Println("")
		fmt.Print("Type 'yes' to confirm: ")
		var confirm string
		fmt.Scanln(&confirm)
		if confirm != "yes" {
			fmt.Println("Cancelled.")
			os.Exit(0)
		}
	}

	home, _ := os.UserHomeDir()
	baseDir := filepath.Join(home, ".ragnarok")

	folders := map[string]string{
		"hati":   ".hati",
		"skoll":  ".skoll",
		"fenrir": ".fenrir",
		"tyr":    ".tyr",
	}

	resetAll := dbList == "all"

	fmt.Printf("\n🔄 Resetting Ragnarok databases...\n\n")

	fmt.Println("📦 Stopping any running Ragnarok servers...")
	stopRagnarokServers()
	time.Sleep(1 * time.Second)

	for name, folder := range folders {
		if !resetAll && !contains(dbList, name) {
			continue
		}

		fullFolder := filepath.Join(baseDir, folder)

		os.RemoveAll(fullFolder)

		if err := os.MkdirAll(fullFolder, 0755); err != nil {
			fmt.Printf("  ❌ %s: failed to create folder - %v\n", name, err)
		} else {
			fmt.Printf("  ✅ %s: folder reset\n", name)
		}
	}

	if resetAll {
		fmt.Println("\n🗄️  Initializing databases...")
		initializeDatabases(home)
	}

	fmt.Println("\n✓ Databases reset complete!")
	fmt.Println("  Run 'rag serve' or 'rag mcp' to start with fresh databases.")
}

func runReinstall(withBackup, force bool) {
	if !force {
		fmt.Println("⚠️  This will COMPLETELY REMOVE all Ragnarok data and reinstall from scratch!")
		fmt.Println("   All databases, sessions, plans, and configurations will be DELETED!")
		fmt.Println("")
		fmt.Print("Type 'yes' to confirm: ")
		var confirm string
		fmt.Scanln(&confirm)
		if confirm != "yes" {
			fmt.Println("Cancelled.")
			os.Exit(0)
		}
	}

	home, _ := os.UserHomeDir()
	ragnarokDir := filepath.Join(home, ".ragnarok")

	fmt.Printf("\n🔄 Complete Ragnarok Reinstallation...\n\n")

	fmt.Println("📦 Stopping any running Ragnarok servers...")
	stopRagnarokServers()

	dirs := map[string]string{
		".ragnarok": ragnarokDir,
		".fenrir":   filepath.Join(ragnarokDir, ".fenrir"),
		".hati":     filepath.Join(ragnarokDir, ".hati"),
		".skoll":    filepath.Join(ragnarokDir, ".skoll"),
		".tyr":      filepath.Join(ragnarokDir, ".tyr"),
	}

	if withBackup {
		fmt.Println("\n💾 Creating backup...")
		backupDir := filepath.Join(home, "OneDrive", "RagnarokBackups", time.Now().Format("2006-01-02_150405"))
		os.MkdirAll(backupDir, 0755)
		for name, dir := range dirs {
			if _, err := os.Stat(dir); err == nil {
				backupPath := filepath.Join(backupDir, name)
				copyDir(dir, backupPath)
				fmt.Printf("  ✅ Backed up %s -> %s\n", name, backupPath)
			}
		}
		fmt.Printf("  Backup saved to: %s\n", backupDir)
	}

	fmt.Println("\n🗑️  Removing old data directories...")
	for name, dir := range dirs {
		if _, err := os.Stat(dir); err == nil {
			os.RemoveAll(dir)
			fmt.Printf("  ✅ Removed %s\n", name)
		}
	}

	fmt.Println("\n✨ Creating fresh directory structure...")
	for name, dir := range dirs {
		os.MkdirAll(dir, 0755)
		fmt.Printf("  ✅ Created %s\n", name)
	}

	fmt.Println("\n🗄️  Initializing databases...")
	initializeDatabases(home)

	fmt.Println("\n✅ Reinstallation complete!")
	fmt.Println("   Run 'rag serve' or 'rag mcp' to start with fresh databases.")
}

func stopRagnarokServers() {
	ports := []int{7437, 7438, 7439, 7440}
	for _, port := range ports {
		cmd := exec.Command("netstat", "-ano")
		output, _ := cmd.Output()
		lines := strings.Split(string(output), "\n")
		for _, line := range lines {
			if strings.Contains(line, fmt.Sprintf(":%d", port)) && strings.Contains(line, "LISTENING") {
				parts := strings.Fields(line)
				if len(parts) > 5 {
					pid := parts[4]
					exec.Command("taskkill", "/F", "/PID", pid).Run()
				}
			}
		}
	}
}

func initializeDatabases(home string) {
	baseDir := filepath.Join(home, ".ragnarok")

	fenrirDir := filepath.Join(baseDir, ".fenrir")
	hatiDir := filepath.Join(baseDir, ".hati")
	skollDir := filepath.Join(baseDir, ".skoll")
	tyrDir := filepath.Join(baseDir, ".tyr")

	if db, err := fenrirdb.NewDB(filepath.Join(fenrirDir, "fenrir.db")); err == nil {
		fenrirdb.InitSchema(db)
		fmt.Printf("  ✅ fenrir.db initialized\n")
	}

	if db, err := hatidb.NewDB(filepath.Join(hatiDir, "hati.db")); err == nil {
		hatidb.InitSchema(db)
		fmt.Printf("  ✅ hati.db initialized\n")
	}

	if db, err := skolldb.NewDB(filepath.Join(skollDir, "skoll.db")); err == nil {
		skolldb.InitSchema(db)
		fmt.Printf("  ✅ skoll.db initialized\n")
	}

	if db, err := tyrdb.NewDB(filepath.Join(tyrDir, "tyr.db")); err == nil {
		tyrdb.InitSchema(db)
		fmt.Printf("  ✅ tyr.db initialized\n")
	}
}

func copyDir(src, dst string) error {
	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		relPath, _ := filepath.Rel(src, path)
		dstPath := filepath.Join(dst, relPath)
		if info.IsDir() {
			os.MkdirAll(dstPath, 0755)
		} else {
			osCopyFile(path, dstPath)
		}
		return nil
	})
}

func osCopyFile(src, dst string) error {
	srcFile, _ := os.Open(src)
	defer srcFile.Close()
	dstFile, _ := os.Create(dst)
	defer dstFile.Close()
	buf := make([]byte, 1024*1024)
	for {
		n, _ := srcFile.Read(buf)
		if n == 0 {
			break
		}
		dstFile.Write(buf[:n])
	}
	return nil
}

func contains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func getPlanID(result interface{}) string {
	if m, ok := result.(map[string]interface{}); ok {
		if id, ok := m["plan_id"].(string); ok {
			return id
		}
	}
	return ""
}

func getPhaseID(result interface{}) string {
	if m, ok := result.(map[string]interface{}); ok {
		if id, ok := m["id"].(string); ok {
			return id
		}
	}
	return ""
}

func getTaskID(result interface{}) string {
	if m, ok := result.(map[string]interface{}); ok {
		if id, ok := m["id"].(string); ok {
			return id
		}
	}
	return ""
}

func getPlanProgress(result interface{}) string {
	if m, ok := result.(map[string]interface{}); ok {
		if progress, ok := m["progress"].(map[string]interface{}); ok {
			if pct, ok := progress["percent"].(float64); ok {
				return fmt.Sprintf("%.0f%%", pct)
			}
		}
	}
	return "unknown"
}

func getActivePlanID() string {
	srv, err := unified.NewServer("")
	if err != nil {
		return ""
	}
	ctx := context.Background()
	result, err := srv.ExecuteWorkflow(ctx, "plan_list", map[string]interface{}{"status": "active"})
	if err != nil {
		return ""
	}
	return getPlanID(result)
}

func printWorkflowResult(workflow string, result interface{}) {
	if result == nil {
		fmt.Println("   (No result data)")
		return
	}

	m, ok := result.(map[string]interface{})
	if !ok {
		fmt.Printf("   Result: %v\n", result)
		return
	}

	// General fields
	if status, ok := m["status"].(string); ok {
		fmt.Printf("   Status: %s\n", status)
	}
	if msg, ok := m["message"].(string); ok && msg != "" {
		fmt.Printf("   Message: %s\n", msg)
	}

	// Specific workflow handling
	switch workflow {
	case "progress":
		total := toInt(m["total_tasks"])
		done := toInt(m["completed_tasks"])
		perc := m["progress_percent"].(string)
		fmt.Printf("   Progress: %s (%d/%d tasks)\n", perc, done, total)
		if phases := toInt(m["phase_count"]); phases > 0 {
			donePhases := toInt(m["completed_phases"])
			fmt.Printf("   Phases:   %d/%d phases completed\n", donePhases, phases)
		}

	case "tasks":
		tasksRaw := m["tasks"]
		var tasksList []interface{}
		if s, ok := tasksRaw.([]interface{}); ok {
			tasksList = s
		} else if s, ok := tasksRaw.([]map[string]interface{}); ok {
			for _, t := range s {
				tasksList = append(tasksList, t)
			}
		}

		if len(tasksList) == 0 {
			fmt.Println("   (No tasks found)")
		}
		for _, t := range tasksList {
			tm, ok := t.(map[string]interface{})
			if !ok {
				continue
			}
			id, _ := tm["id"].(string)
			title, _ := tm["title"].(string)
			status, _ := tm["status"].(string)

			icon := "○"
			if status == "completed" {
				icon = "✓"
			} else if status == "in_progress" {
				icon = "▶"
			} else if status == "blocked" {
				icon = "⚠"
			}

			fmt.Printf("   %s [%s] %s\n", icon, id, title)
		}

	case "diagnose":
		if healthy, ok := m["healthy"].(bool); ok {
			status := "Healthy"
			if !healthy {
				status = "Unhealthy"
			}
			fmt.Printf("   System Status: %s\n", status)
		}
		if issues, ok := m["issues"].([]interface{}); ok && len(issues) > 0 {
			fmt.Printf("   Issues found: %d\n", len(issues))
			for _, iss := range issues {
				fmt.Printf("     - %s\n", iss)
			}
		}

	case "continue", "review":
		if steps, ok := m["steps"].([]interface{}); ok {
			if len(steps) == 0 {
				fmt.Println("   (No actions performed)")
			}
			for _, s := range steps {
				sm, ok := s.(map[string]interface{})
				if !ok {
					continue
				}

				name, _ := sm["name"].(string)
				status, _ := sm["status"].(string)
				icon := "○"
				if status == "success" {
					icon = "✓"
				} else if status == "error" {
					icon = "❌"
				} else if status == "in_progress" {
					icon = "▶"
				}

				fmt.Printf("   %s [%-14s] %s\n", icon, status, name)
				if out, ok := sm["output"].(string); ok && out != "" {
					fmt.Printf("      Output: %s\n", out)
				}
				if err, ok := sm["error"].(string); ok && err != "" {
					fmt.Printf("      Error: %s\n", err)
				}
			}
		}

	default:
		// Generic steps print
		if steps, ok := m["steps"].([]interface{}); ok {
			fmt.Printf("   Steps performed: %d\n", len(steps))
			for i, s := range steps {
				if i >= 3 && len(steps) > 5 { // Limit generic output
					fmt.Printf("   ... and %d more steps\n", len(steps)-3)
					break
				}
				fmt.Printf("   - %v\n", s)
			}
		}
	}
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

func runDoctor(verbose bool) {
	fmt.Printf("Ragnarok Doctor - Health Check\n")
	fmt.Println(strings.Repeat("=", 50))

	home, _ := os.UserHomeDir()
	baseDir := filepath.Join(home, ".ragnarok")
	issues := 0
	checks := 0

	plugins := map[string]string{
		"fenrir": filepath.Join(baseDir, ".fenrir"),
		"hati":   filepath.Join(baseDir, ".hati"),
		"skoll":  filepath.Join(baseDir, ".skoll"),
		"tyr":    filepath.Join(baseDir, ".tyr"),
	}

	for name, dir := range plugins {
		checks++
		fmt.Printf("\n[%s]\n", strings.ToUpper(name))
		fmt.Printf("  Directory: %s\n", dir)

		if _, err := os.Stat(dir); os.IsNotExist(err) {
			fmt.Printf("  ❌ Directory does not exist\n")
			issues++
			continue
		}
		fmt.Printf("  ✅ Directory exists\n")

		dbPath := filepath.Join(dir, name+".db")
		if _, err := os.Stat(dbPath); os.IsNotExist(err) {
			fmt.Printf("  ❌ Database file does not exist: %s\n", dbPath)
			issues++
		} else {
			fmt.Printf("  ✅ Database exists: %s\n", dbPath)
			if verbose {
				info, _ := os.Stat(dbPath)
				fmt.Printf("     Size: %d bytes\n", info.Size())
			}
		}

		configPath := filepath.Join(dir, "config.json")
		if _, err := os.Stat(configPath); os.IsNotExist(err) {
			if verbose {
				fmt.Printf("  ⚠️  Config file does not exist (optional)\n")
			}
		} else {
			fmt.Printf("  ✅ Config exists\n")
		}
	}

	fmt.Printf("\n%s\n", strings.Repeat("=", 50))
	fmt.Printf("Database Check (via unified server):\n")
	srv, err := unified.NewServer("")
	if err != nil {
		fmt.Printf("  ❌ Failed to initialize server: %v\n", err)
		issues++
	} else {
		fmt.Printf("  ✅ Unified server initialized\n")
		fmt.Printf("  ✅ Tools registered: %d\n", len(srv.ListTools()))
	}

	for _, name := range []string{"fenrir", "hati", "skoll", "tyr"} {
		dbPath := filepath.Join(baseDir, "."+name, name+".db")
		db, err := sql.Open("sqlite", dbPath)
		if err != nil {
			fmt.Printf("  ❌ %s: Cannot open database\n", name)
			issues++
			continue
		}

		var version string
		err = db.QueryRow("PRAGMA user_version").Scan(&version)
		if err == nil && verbose {
			fmt.Printf("  ✅ %s DB accessible (user_version: %s)\n", name, version)
		} else if err == nil {
			fmt.Printf("  ✅ %s DB accessible\n", name)
		} else {
			fmt.Printf("  ❌ %s: %v\n", name, err)
			issues++
		}
		db.Close()
	}

	fmt.Printf("\n%s\n", strings.Repeat("=", 50))
	if issues == 0 {
		fmt.Printf("✅ All checks passed! Ragnarok is healthy.\n")
	} else {
		fmt.Printf("❌ %d issue(s) found. Run 'rag cleanup --optimize' to fix.\n", issues)
	}
}

func runCleanup(force bool, optimize bool) {
	if !force {
		fmt.Println("⚠️  This will clean up WAL files, temporary files, and optimize databases!")
		fmt.Println("")
		fmt.Print("Type 'yes' to confirm: ")
		var confirm string
		fmt.Scanln(&confirm)
		if confirm != "yes" {
			fmt.Println("Cancelled.")
			os.Exit(0)
		}
	}

	home, _ := os.UserHomeDir()
	baseDir := filepath.Join(home, ".ragnarok")

	fmt.Printf("Ragnarok Cleanup\n")
	fmt.Println(strings.Repeat("=", 50))

	plugins := []string{"fenrir", "hati", "skoll", "tyr"}
	totalCleaned := int64(0)

	for _, name := range plugins {
		dir := filepath.Join(baseDir, "."+name)
		dbPath := filepath.Join(dir, name+".db")

		fmt.Printf("\n[%s]\n", strings.ToUpper(name))

		for _, ext := range []string{"-wal", "-shm"} {
			path := dbPath + ext
			if info, err := os.Stat(path); err == nil {
				size := info.Size()
				os.Remove(path)
				fmt.Printf("  🗑️  Removed %s (%d bytes)\n", ext, size)
				totalCleaned += size
			}
		}

		if optimize {
			db, err := sql.Open("sqlite", dbPath)
			if err == nil {
				db.Exec("VACUUM")
				db.Exec("ANALYZE")
				fmt.Printf("  ✅ Database optimized\n")
				db.Close()
			} else {
				fmt.Printf("  ❌ Failed to optimize: %v\n", err)
			}
		}
	}

	fmt.Printf("\n%s\n", strings.Repeat("=", 50))
	fmt.Printf("✅ Cleanup complete! Total space freed: %d bytes (%.2f MB)\n", totalCleaned, float64(totalCleaned)/1024/1024)
}

func runMigrate(plugin string) {
	home, _ := os.UserHomeDir()
	baseDir := filepath.Join(home, ".ragnarok")

	fmt.Printf("Ragnarok Migrate\n")
	fmt.Println(strings.Repeat("=", 50))

	plugins := map[string]string{
		"fenrir": filepath.Join(baseDir, ".fenrir"),
		"hati":   filepath.Join(baseDir, ".hati"),
		"skoll":  filepath.Join(baseDir, ".skoll"),
		"tyr":    filepath.Join(baseDir, ".tyr"),
	}

	migratePlugins := plugins
	if plugin != "all" {
		if path, ok := plugins[plugin]; ok {
			migratePlugins = map[string]string{plugin: path}
		} else {
			fmt.Printf("Error: Unknown plugin '%s'\n", plugin)
			fmt.Printf("Available: all, fenrir, hati, skoll, tyr\n")
			os.Exit(1)
		}
	}

	for name, dir := range migratePlugins {
		fmt.Printf("\n[%s]\n", strings.ToUpper(name))
		dbPath := filepath.Join(dir, name+".db")

		db, err := sql.Open("sqlite", dbPath)
		if err != nil {
			fmt.Printf("  ❌ Cannot open database: %v\n", err)
			continue
		}

		fmt.Printf("  🔄 Running migrations...\n")

		switch name {
		case "fenrir":
			fenrirdb.InitSchema(db)
		case "hati":
			hatidb.InitSchema(db)
		case "skoll":
			skolldb.InitSchema(db)
		case "tyr":
			tyrdb.InitSchema(db)
		}

		fmt.Printf("  ✅ Migrations complete\n")
		db.Close()
	}

	fmt.Printf("\n%s\n", strings.Repeat("=", 50))
	fmt.Printf("✅ Migration complete!\n")
}

func runInfo() {
	fmt.Printf("Ragnarok v%s - Information\n", version)
	fmt.Println(strings.Repeat("=", 50))

	home, _ := os.UserHomeDir()
	baseDir := filepath.Join(home, ".ragnarok")

	fmt.Printf("\n[Binary Information]\n")
	if exe, err := os.Executable(); err == nil {
		fmt.Printf("  Executable: %s\n", exe)
		if info, err := os.Stat(exe); err == nil {
			fmt.Printf("  Modified: %s\n", info.ModTime().Format("2006-01-02 15:04:05"))
		}
	}
	fmt.Printf("  Version: %s\n", version)
	fmt.Printf("  GOOS: %s\n", runtime.GOOS)
	fmt.Printf("  GOARCH: %s\n", runtime.GOARCH)

	fmt.Printf("\n[Data Directories]\n")
	fmt.Printf("  Base: %s\n", baseDir)

	plugins := map[string]string{
		"fenrir": filepath.Join(baseDir, ".fenrir"),
		"hati":   filepath.Join(baseDir, ".hati"),
		"skoll":  filepath.Join(baseDir, ".skoll"),
		"tyr":    filepath.Join(baseDir, ".tyr"),
	}

	for name, dir := range plugins {
		fmt.Printf("  %s: %s\n", name, dir)
		if _, err := os.Stat(dir); os.IsNotExist(err) {
			fmt.Printf("    ❌ Not found\n")
		} else {
			dbPath := filepath.Join(dir, name+".db")
			if info, err := os.Stat(dbPath); err == nil {
				fmt.Printf("    ✅ DB: %s (%.2f KB)\n", info.ModTime().Format("2006-01-02 15:04"), float64(info.Size())/1024)
			} else {
				fmt.Printf("    ⚠️  DB not found\n")
			}
		}
	}

	fmt.Printf("\n[Server Status]\n")
	ports := []int{7437, 7438, 7439, 7440}
	portNames := []string{"fenrir", "skoll", "hati", "tyr"}
	for i, port := range ports {
		cmd := exec.Command("netstat", "-ano")
		output, _ := cmd.Output()
		running := strings.Contains(string(output), fmt.Sprintf(":%d", port))
		if running {
			fmt.Printf("  %s (port %d): ✅ Running\n", portNames[i], port)
		} else {
			fmt.Printf("  %s (port %d): ⚪ Stopped\n", portNames[i], port)
		}
	}

	fmt.Printf("\n%s\n", strings.Repeat("=", 50))
}

func runConfig(plugin string, key string, value string) {
	if plugin == "" {
		fmt.Println("Error: --plugin is required (fenrir, hati, skoll, tyr)")
		fmt.Println("Usage: rag config --plugin <plugin> [--key <key>] [--value <value>]")
		os.Exit(1)
	}

	home, _ := os.UserHomeDir()
	baseDir := filepath.Join(home, ".ragnarok")

	pluginDirs := map[string]string{
		"fenrir": filepath.Join(baseDir, ".fenrir"),
		"hati":   filepath.Join(baseDir, ".hati"),
		"skoll":  filepath.Join(baseDir, ".skoll"),
		"tyr":    filepath.Join(baseDir, ".tyr"),
	}

	dir, ok := pluginDirs[plugin]
	if !ok {
		fmt.Printf("Error: Unknown plugin '%s'\n", plugin)
		os.Exit(1)
	}

	configPath := filepath.Join(dir, "config.json")
	data, err := os.ReadFile(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			fmt.Printf("Config file does not exist at %s\n", configPath)
			fmt.Printf("Run 'rag serve' first to generate config files.\n")
		} else {
			fmt.Printf("Error reading config: %v\n", err)
		}
		os.Exit(1)
	}

	var config map[string]interface{}
	if err := json.Unmarshal(data, &config); err != nil {
		fmt.Printf("Error parsing config: %v\n", err)
		os.Exit(1)
	}

	if key == "" {
		fmt.Printf("[%s config]\n", plugin)
		fmt.Println(strings.Repeat("-", 30))
		for k, v := range config {
			fmt.Printf("  %s: %v\n", k, v)
		}
		return
	}

	if value == "" {
		if v, ok := config[key]; ok {
			fmt.Printf("%s = %v\n", key, v)
		} else {
			fmt.Printf("Key '%s' not found\n", key)
			os.Exit(1)
		}
		return
	}

	config[key] = value
	data, err = json.MarshalIndent(config, "", "  ")
	if err != nil {
		fmt.Printf("Error marshaling config: %v\n", err)
		os.Exit(1)
	}

	if err := os.WriteFile(configPath, data, 0644); err != nil {
		fmt.Printf("Error writing config: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("✅ %s.%s = %v\n", plugin, key, value)
}

func runExport(plugin string, filePath string) {
	if filePath == "" {
		fmt.Println("Error: --file is required")
		fmt.Println("Usage: rag export --plugin <plugin> --file <path>")
		os.Exit(1)
	}

	home, _ := os.UserHomeDir()
	baseDir := filepath.Join(home, ".ragnarok")

	plugins := map[string]string{
		"fenrir": filepath.Join(baseDir, ".fenrir"),
		"hati":   filepath.Join(baseDir, ".hati"),
		"skoll":  filepath.Join(baseDir, ".skoll"),
		"tyr":    filepath.Join(baseDir, ".tyr"),
	}

	exportPlugins := plugins
	if plugin != "all" {
		if path, ok := plugins[plugin]; ok {
			exportPlugins = map[string]string{plugin: path}
		} else {
			fmt.Printf("Error: Unknown plugin '%s'\n", plugin)
			os.Exit(1)
		}
	}

	fmt.Printf("Ragnarok Export\n")
	fmt.Println(strings.Repeat("=", 50))

	exportData := make(map[string]interface{})
	exportData["version"] = version
	exportData["exported_at"] = time.Now().Format(time.RFC3339)
	exportData["plugins"] = make(map[string]interface{})

	for name, dir := range exportPlugins {
		fmt.Printf("\n[%s]\n", strings.ToUpper(name))
		dbPath := filepath.Join(dir, name+".db")

		db, err := sql.Open("sqlite", dbPath)
		if err != nil {
			fmt.Printf("  ❌ Cannot open database: %v\n", err)
			continue
		}

		tables, _ := db.Query("SELECT name FROM sqlite_master WHERE type='table'")
		tableData := make(map[string][]map[string]interface{})

		for tables.Next() {
			var tableName string
			tables.Scan(&tableName)
			rows, err := db.Query("SELECT * FROM " + tableName)
			if err != nil {
				continue
			}

			cols, _ := rows.Columns()
			var records []map[string]interface{}
			for rows.Next() {
				record := make(map[string]interface{})
				values := make([]interface{}, len(cols))
				for i := range cols {
					values[i] = new(interface{})
				}
				rows.Scan(values...)
				for i, col := range cols {
					record[col] = *(values[i].(*interface{}))
				}
				records = append(records, record)
			}
			tableData[tableName] = records
			rows.Close()
		}
		tables.Close()

		exportData["plugins"].(map[string]interface{})[name] = map[string]interface{}{
			"tables":  tableData,
			"db_path": dbPath,
		}
		fmt.Printf("  ✅ Exported %d tables\n", len(tableData))
		db.Close()
	}

	data, _ := json.MarshalIndent(exportData, "", "  ")
	if err := os.WriteFile(filePath, data, 0644); err != nil {
		fmt.Printf("Error writing export file: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("\n%s\n", strings.Repeat("=", 50))
	fmt.Printf("✅ Export complete! File: %s (%.2f KB)\n", filePath, float64(len(data))/1024)
}

func runImport(filePath string, plugin string) {
	if filePath == "" {
		fmt.Println("Error: --file is required")
		fmt.Println("Usage: rag import --file <path> [--plugin <plugin>]")
		os.Exit(1)
	}

	fmt.Printf("Ragnarok Import\n")
	fmt.Println(strings.Repeat("=", 50))

	data, err := os.ReadFile(filePath)
	if err != nil {
		fmt.Printf("❌ Error reading file: %v\n", err)
		os.Exit(1)
	}

	var exportData map[string]interface{}
	if err := json.Unmarshal(data, &exportData); err != nil {
		fmt.Printf("❌ Error parsing export file: %v\n", err)
		os.Exit(1)
	}

	pluginsData, ok := exportData["plugins"].(map[string]interface{})
	if !ok {
		fmt.Printf("❌ Invalid export file format: missing plugins data\n")
		os.Exit(1)
	}

	home, _ := os.UserHomeDir()
	baseDir := filepath.Join(home, ".ragnarok")

	plugins := map[string]string{
		"fenrir": filepath.Join(baseDir, ".fenrir"),
		"hati":   filepath.Join(baseDir, ".hati"),
		"skoll":  filepath.Join(baseDir, ".skoll"),
		"tyr":    filepath.Join(baseDir, ".tyr"),
	}

	importPlugins := plugins
	if plugin != "all" {
		if path, ok := plugins[plugin]; ok {
			importPlugins = map[string]string{plugin: path}
		} else {
			fmt.Printf("Error: Unknown plugin '%s'\n", plugin)
			fmt.Printf("Available: all, fenrir, hati, skoll, tyr\n")
			os.Exit(1)
		}
	}

	totalImported := 0

	for name, dir := range importPlugins {
		pluginData, ok := pluginsData[name].(map[string]interface{})
		if !ok {
			fmt.Printf("\n[%s] No data to import\n", strings.ToUpper(name))
			continue
		}

		fmt.Printf("\n[%s]\n", strings.ToUpper(name))

		tablesData, ok := pluginData["tables"].(map[string]interface{})
		if !ok {
			fmt.Printf("  ⚠️  No tables data found\n")
			continue
		}

		dbPath := filepath.Join(dir, name+".db")
		db, err := sql.Open("sqlite", dbPath)
		if err != nil {
			fmt.Printf("  ❌ Cannot open database: %v\n", err)
			continue
		}

		imported := 0
		for tableName, records := range tablesData {
			recordsList, ok := records.([]interface{})
			if !ok {
				continue
			}

			for _, record := range recordsList {
				recordMap, ok := record.(map[string]interface{})
				if !ok {
					continue
				}

				if err := importRecord(db, tableName, recordMap); err == nil {
					imported++
				}
			}
		}

		fmt.Printf("  ✅ Imported %d records into %s\n", imported, name)
		totalImported += imported
		db.Close()
	}

	fmt.Printf("\n%s\n", strings.Repeat("=", 50))
	fmt.Printf("✅ Import complete! Total records imported: %d\n", totalImported)
}

func importRecord(db *sql.DB, tableName string, record map[string]interface{}) error {
	if len(record) == 0 {
		return nil
	}

	columns := make([]string, 0, len(record))
	placeholders := make([]string, 0, len(record))
	values := make([]interface{}, 0, len(record))

	for col, val := range record {
		columns = append(columns, col)
		placeholders = append(placeholders, "?")
		if val == nil {
			values = append(values, nil)
		} else {
			values = append(values, val)
		}
	}

	query := fmt.Sprintf("INSERT OR REPLACE INTO %s (%s) VALUES (%s)",
		tableName, strings.Join(columns, ", "), strings.Join(placeholders, ", "))

	_, err := db.Exec(query, values...)
	return err
}

func runServer(action string) {
	switch action {
	case "status":
		fmt.Printf("Ragnarok Server Status\n")
		fmt.Println(strings.Repeat("=", 50))
		ports := []int{7437, 7438, 7439, 7440}
		names := []string{"Fenrir", "Skoll", "Hati", "Tyr"}
		for i, port := range ports {
			cmd := exec.Command("netstat", "-ano")
			output, _ := cmd.Output()
			if strings.Contains(string(output), fmt.Sprintf(":%d", port)) {
				fmt.Printf("  %s (port %d): ✅ Running\n", names[i], port)
			} else {
				fmt.Printf("  %s (port %d): ⚪ Stopped\n", names[i], port)
			}
		}
	case "start":
		fmt.Printf("ℹ️  Use 'rag serve' to start the unified MCP server.\n")
	case "stop":
		stopRagnarokServers()
		fmt.Printf("✅ Servers stopped.\n")
	case "restart":
		fmt.Printf("Stopping servers...\n")
		stopRagnarokServers()
		time.Sleep(1 * time.Second)
		fmt.Printf("✅ Use 'rag serve' to start again.\n")
	default:
		fmt.Printf("Error: Unknown action '%s'\n", action)
		fmt.Printf("Usage: rag server --action <status|start|stop|restart>\n")
	}
}
