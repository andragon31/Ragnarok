package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/andragon31/Ragnarok/internal/fenrir/config"
	"github.com/andragon31/Ragnarok/internal/fenrir/database"
	"github.com/andragon31/Ragnarok/internal/fenrir/mcp"
	"github.com/andragon31/Ragnarok/internal/fenrir/memory"
	"github.com/andragon31/Ragnarok/internal/fenrir/scanner"
)

var version = "1.4.3"

func main() {
	if len(os.Args) > 1 && os.Args[1] == "version" {
		fmt.Printf("Fenrir v%s\n", version)
		fmt.Println("Memory, Knowledge & Institutional Intelligence Layer")
		return
	}

	serveCmd := flag.NewFlagSet("serve", flag.ExitOnError)
	port := serveCmd.Int("port", 7437, "MCP server port")
	configPath := serveCmd.String("config", "", "Config file path")
	dataDir := serveCmd.String("dir", "", "Data directory")

	initCmd := flag.NewFlagSet("init", flag.ExitOnError)
	initProject := initCmd.String("project", "", "Project name")

	scanCmd := flag.NewFlagSet("scan", flag.ExitOnError)
	scanPath := scanCmd.String("path", ".", "Project path to scan")
	scanLayer := scanCmd.String("layer", "", "Layer to scan (stack, arch, config, modules, patterns)")

	bootstrapCmd := flag.NewFlagSet("bootstrap", flag.ExitOnError)
	bootstrapPath := bootstrapCmd.String("path", ".", "Project path to bootstrap")

	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	switch os.Args[1] {
	case "serve":
		serveCmd.Parse(os.Args[2:])
		if err := runServe(*port, *configPath, *dataDir); err != nil {
			log.Fatal(err)
		}
	case "init":
		initCmd.Parse(os.Args[2:])
		if *initProject == "" {
			fmt.Println("Error: --project is required")
			initCmd.PrintDefaults()
			os.Exit(1)
		}
		if err := runInit(*initProject, *dataDir); err != nil {
			log.Fatal(err)
		}
	case "scan":
		scanCmd.Parse(os.Args[2:])
		runScan(*scanPath, *scanLayer)
	case "bootstrap":
		bootstrapCmd.Parse(os.Args[2:])
		runBootstrap(*bootstrapPath)
	case "version":
		fmt.Printf("Fenrir v%s\n", version)
	case "stats":
		if err := runStats(*configPath); err != nil {
			log.Fatal(err)
		}
	case "mcp":
		if err := runMCP(*configPath, *dataDir); err != nil {
			log.Fatal(err)
		}
	default:
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println(`Fenrir v1.1.0 - Memory, Knowledge & Institutional Intelligence Layer

Usage:
  fenrir serve [--port PORT] [--config FILE] [--dir DIR]
  fenrir init --project NAME [--dir DIR]
  fenrir scan [--path PATH] [--layer LAYER]
  fenrir bootstrap [--path PATH]
  fenrir stats [--config FILE]
  fenrir version

Commands:
  serve     Start the MCP server
  init      Initialize a new project
  scan      Analyze project structure and detect stack/architecture
  bootstrap Bootstrap agentic structure from project analysis
  stats     Show statistics
  mcp       Run in MCP mode (stdio)
  version   Show version

Layers for scan:
  stack        Detect programming language, framework, package manager
  arch         Detect architecture type (monolith, modular, microservices)
  config       Detect configuration files
  modules      Detect project modules and dependencies
  patterns     Detect patterns (testing, CI/CD, docker)

Examples:
  fenrir scan --path ./myproject --layer stack
  fenrir bootstrap --path ./myproject
  fenrir scan
  fenrir serve --port 7437`)
}

func runServe(port int, configPath, dataDir string) error {
	if dataDir == "" {
		home, _ := os.UserHomeDir()
		dataDir = filepath.Join(home, ".fenrir")
	}

	cfg := &config.Config{
		Port:    port,
		DataDir: dataDir,
	}

	if configPath != "" {
		var err error
		cfg, err = config.LoadConfig(configPath)
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}
	}

	dbPath := filepath.Join(cfg.DataDir, "fenrir.db")
	db, err := database.NewDB(dbPath)
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}
	defer db.Close()

	if err := database.InitSchema(db); err != nil {
		return fmt.Errorf("failed to init schema: %w", err)
	}

	ctx := context.Background()
	server := mcp.NewServer(cfg, db)

	log.Printf("Starting Fenrir server on port %d...", port)
	return server.Run(ctx)
}

func runInit(projectName, dataDir string) error {
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}

	agentsPath := filepath.Join(cwd, "AGENTS.md")
	if _, err := os.Stat(agentsPath); os.IsNotExist(err) {
		if err := generateAgentsMD(projectName); err != nil {
			fmt.Printf("  Warning: Could not generate AGENTS.md: %v\n", err)
		} else {
			fmt.Printf("  Generated: %s\n", agentsPath)
		}
	} else {
		fmt.Printf("  AGENTS.md already exists: %s\n", agentsPath)
	}

	if dataDir == "" {
		home, _ := os.UserHomeDir()
		dataDir = filepath.Join(home, ".fenrir")
	}

	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return fmt.Errorf("failed to create data dir: %w", err)
	}

	dbPath := filepath.Join(dataDir, "fenrir.db")
	db, err := database.NewDB(dbPath)
	if err != nil {
		fmt.Printf("  Warning: Database initialization failed (CGO required for sqlite3): %v\n", err)
		fmt.Printf("\nTo start the MCP server (requires CGO-enabled build):\n  fenrir serve\n")
		return nil
	}
	defer db.Close()

	if err := database.InitSchema(db); err != nil {
		return fmt.Errorf("failed to init schema: %w", err)
	}

	cfg := &config.Config{
		Project: projectName,
		Version: version,
		Port:    7437,
		DataDir: dataDir,
	}

	configPath := filepath.Join(dataDir, "config.json")
	if err := cfg.Save(configPath); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	fmt.Printf("✓ Initialized Fenrir for project: %s\n", projectName)
	fmt.Printf("  Data directory: %s\n", dataDir)
	fmt.Printf("  Config: %s\n", configPath)

	fmt.Printf("\nTo start the MCP server:\n  fenrir serve\n")

	return nil
}

func generateAgentsMD(projectName string) error {
	cwd, err := os.Getwd()
	if err != nil {
		return err
	}

	analyzer := scanner.NewProjectAnalyzer(cwd)
	analysis, err := analyzer.Analyze()
	if err != nil {
		return err
	}

	var buf strings.Builder

	buf.WriteString("# " + projectName + " - Agent Guidelines\n\n")
	buf.WriteString("> Auto-generated by Fenrir v" + version + " on project scan\n\n")

	buf.WriteString("## Project Stack\n\n")
	if analysis.Stack.Language != "" {
		buf.WriteString("- **Language**: " + analysis.Stack.Language + "\n")
	}
	if analysis.Stack.Framework != "" {
		buf.WriteString("- **Framework**: " + analysis.Stack.Framework + "\n")
	}
	if analysis.Stack.PackageMgr != "" {
		buf.WriteString("- **Package Manager**: " + analysis.Stack.PackageMgr + "\n")
	}
	if analysis.Stack.TestFramework != "" {
		buf.WriteString("- **Test Framework**: " + analysis.Stack.TestFramework + "\n")
	}
	buf.WriteString("- **Architecture**: " + analysis.Architecture.Type + "\n\n")

	if len(analysis.Modules) > 0 {
		buf.WriteString("## Modules\n\n")
		for _, m := range analysis.Modules {
			buf.WriteString("- `" + m.Path + "` (" + m.Type + ")\n")
		}
		buf.WriteString("\n")
	}

	rules := scanner.GenerateRulesConfig(analysis)
	if len(rules) > 0 {
		buf.WriteString("## Project Rules\n\n")
		for _, r := range rules {
			buf.WriteString("- **" + r["name"] + "**: " + r["description"] + " [" + r["severity"] + "]\n")
		}
		buf.WriteString("\n")
	}

	standards := scanner.GenerateStandardsConfig(analysis)
	if len(standards) > 0 {
		buf.WriteString("## Quality Standards\n\n")
		for _, s := range standards {
			buf.WriteString("- `" + s["name"] + "`")
			if s["block"] == "true" {
				buf.WriteString(" (blocks merge)")
			}
			buf.WriteString(": " + s["description"] + "\n")
		}
		buf.WriteString("\n")
	}

	skills := scanner.GenerateSkillsConfig(analysis)
	if sg, ok := skills["suggested_skills"].([]map[string]string); ok && len(sg) > 0 {
		buf.WriteString("## Suggested Skills\n\n")
		for _, s := range sg {
			buf.WriteString("- " + s["skill"] + " (" + s["type"] + ")\n")
		}
		buf.WriteString("\n")
	}

	buf.WriteString("## Commands\n\n")
	if analysis.Stack.TestFramework == "jest" || analysis.Stack.TestFramework == "vitest" {
		buf.WriteString("- Run tests: `npm test`\n")
	} else if analysis.Stack.TestFramework == "pytest" {
		buf.WriteString("- Run tests: `pytest`\n")
	} else if analysis.Stack.Language == "go" {
		buf.WriteString("- Run tests: `go test ./...`\n")
	}
	if analysis.Stack.Language == "typescript" || analysis.Stack.Language == "javascript/typescript" {
		buf.WriteString("- Run linter: `npm run lint`\n")
	}
	if analysis.Stack.Language == "go" {
		buf.WriteString("- Build: `go build ./...`\n")
	}
	buf.WriteString("\n")

	agentsPath := filepath.Join(cwd, "AGENTS.md")
	return os.WriteFile(agentsPath, []byte(buf.String()), 0644)
}

func runStats(configPath string) error {
	home, _ := os.UserHomeDir()
	if configPath == "" {
		configPath = filepath.Join(home, ".fenrir", "config.json")
	}

	cfg, err := config.LoadConfig(configPath)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	dbPath := filepath.Join(cfg.DataDir, "fenrir.db")
	db, err := database.NewDB(dbPath)
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}
	defer db.Close()

	store := memory.NewMemoryStore(db)
	stats, err := store.GetStats()
	if err != nil {
		return fmt.Errorf("failed to get stats: %w", err)
	}

	fmt.Println("Fenrir Statistics")
	fmt.Println("==================")
	fmt.Printf("Project: %s\n", cfg.Project)
	fmt.Printf("Data Dir: %s\n", cfg.DataDir)
	fmt.Println("─────────────────")
	fmt.Printf("Sessions: %d\n", stats.TotalSessions)
	fmt.Printf("Observations: %d\n", stats.TotalObservations)
	fmt.Printf("Specs: %d\n", stats.TotalSpecs)
	fmt.Printf("Open Incidents: %d\n", stats.OpenIncidents)

	return nil
}

func runMCP(configPath, dataDir string) error {
	home, _ := os.UserHomeDir()
	if dataDir == "" {
		dataDir = filepath.Join(home, ".fenrir")
	}

	cfg := &config.Config{
		Port:    7437,
		DataDir: dataDir,
	}

	if configPath != "" {
		var err error
		cfg, err = config.LoadConfig(configPath)
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}
	}

	dbPath := filepath.Join(cfg.DataDir, "fenrir.db")
	db, err := database.NewDB(dbPath)
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}
	defer db.Close()

	if err := database.InitSchema(db); err != nil {
		return fmt.Errorf("failed to init schema: %w", err)
	}

	ctx := context.Background()
	server := mcp.NewServer(cfg, db)

	return server.Run(ctx)
}

func runScan(projectPath, layer string) {
	absPath, err := filepath.Abs(projectPath)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Scanning project: %s\n", absPath)
	fmt.Println(strings.Repeat("─", 50))

	analyzer := scanner.NewProjectAnalyzer(absPath)
	analysis, err := analyzer.Analyze()
	if err != nil {
		fmt.Printf("Error analyzing project: %v\n", err)
		os.Exit(1)
	}

	if layer == "" || layer == "all" {
		printFullAnalysis(analysis)
	} else {
		printLayerAnalysis(analysis, layer)
	}
}

func printFullAnalysis(a *scanner.ProjectAnalysis) {
	fmt.Printf("Project: %s\n", filepath.Base(a.Path))
	fmt.Printf("Path: %s\n", a.Path)
	fmt.Println()

	if a.Stack.Language != "" {
		fmt.Println("📦 STACK")
		fmt.Printf("   Language: %s\n", a.Stack.Language)
		if a.Stack.Framework != "" {
			fmt.Printf("   Framework: %s\n", a.Stack.Framework)
		}
		fmt.Printf("   Package Manager: %s\n", a.Stack.PackageMgr)
		if a.Stack.TestFramework != "" {
			fmt.Printf("   Test Framework: %s\n", a.Stack.TestFramework)
		}
		fmt.Printf("   Docker: %v | CI/CD: %v | Tests: %v\n",
			a.Stack.HasDocker, a.Stack.HasCI, a.Stack.HasTests)
		fmt.Println()
	}

	if a.Architecture.Type != "" {
		fmt.Println("🏗️  ARCHITECTURE")
		fmt.Printf("   Type: %s\n", a.Architecture.Type)
		fmt.Printf("   Monorepo: %v\n", a.Architecture.IsMonorepo)
		fmt.Printf("   API: %v | Frontend: %v\n",
			a.Architecture.HasAPI, a.Architecture.HasFrontend)
		if len(a.Architecture.Modules) > 0 {
			fmt.Printf("   Modules: %s\n", strings.Join(a.Architecture.Modules, ", "))
		}
		fmt.Println()
	}

	if len(a.Modules) > 0 {
		fmt.Printf("📁 MODULES (%d)\n", len(a.Modules))
		for _, m := range a.Modules {
			if m.HasTests {
				fmt.Printf("   ✓ %s (%s)\n", m.Name, m.Type)
			} else {
				fmt.Printf("   • %s (%s)\n", m.Name, m.Type)
			}
		}
		fmt.Println()
	}

	if len(a.Patterns) > 0 {
		fmt.Println("🔍 PATTERNS DETECTED")
		for _, p := range a.Patterns {
			if p.Detected {
				fmt.Printf("   ✓ %s: %s\n", p.Name, p.Description)
			}
		}
		fmt.Println()
	}

	if len(a.ConfigFiles) > 0 {
		fmt.Printf("⚙️  CONFIG FILES (%d)\n", len(a.ConfigFiles))
		for _, c := range a.ConfigFiles {
			fmt.Printf("   • %s (%s)\n", c.Name, c.Type)
		}
		fmt.Println()
	}

	skills := scanner.GenerateSkillsConfig(a)
	if sg, ok := skills["suggested_skills"].([]map[string]string); ok && len(sg) > 0 {
		fmt.Println("🎯 SUGGESTED SKILLS")
		for _, s := range sg {
			fmt.Printf("   • %s (%s): %s\n", s["name"], s["type"], s["skill"])
		}
		fmt.Println()
	}

	rules := scanner.GenerateRulesConfig(a)
	if len(rules) > 0 {
		fmt.Println("📋 SUGGESTED RULES")
		for _, r := range rules {
			fmt.Printf("   • %s [%s]: %s\n", r["name"], r["severity"], r["description"])
		}
		fmt.Println()
	}

	standards := scanner.GenerateStandardsConfig(a)
	if len(standards) > 0 {
		fmt.Println("✅ SUGGESTED STANDARDS")
		for _, st := range standards {
			block := ""
			if st["block"] == "true" {
				block = " (blocks)"
			}
			fmt.Printf("   • %s%s: %s\n", st["name"], block, st["description"])
		}
		fmt.Println()
	}
}

func printLayerAnalysis(a *scanner.ProjectAnalysis, layer string) {
	switch layer {
	case "stack":
		fmt.Println("📦 STACK LAYER")
		fmt.Printf("   Language: %s\n", a.Stack.Language)
		fmt.Printf("   Framework: %s\n", a.Stack.Framework)
		fmt.Printf("   Package Manager: %s\n", a.Stack.PackageMgr)
		fmt.Printf("   Runtime: %s\n", a.Stack.Runtime)
		fmt.Printf("   Test Framework: %s\n", a.Stack.TestFramework)
		fmt.Printf("   Docker: %v\n", a.Stack.HasDocker)
		fmt.Printf("   CI/CD: %v (%s)\n", a.Stack.HasCI, a.Stack.CITool)
		fmt.Printf("   Has Tests: %v\n", a.Stack.HasTests)

	case "arch", "architecture":
		fmt.Println("🏗️  ARCHITECTURE LAYER")
		fmt.Printf("   Type: %s\n", a.Architecture.Type)
		fmt.Printf("   Monorepo: %v\n", a.Architecture.IsMonorepo)
		fmt.Printf("   Has API: %v\n", a.Architecture.HasAPI)
		fmt.Printf("   API Framework: %s\n", a.Architecture.APIFramework)
		fmt.Printf("   Has Frontend: %v\n", a.Architecture.HasFrontend)
		fmt.Printf("   Modules: %v\n", len(a.Architecture.Modules))
		for _, m := range a.Architecture.Modules {
			fmt.Printf("      - %s\n", m)
		}

	case "modules":
		fmt.Printf("📁 MODULES LAYER (%d modules)\n", len(a.Modules))
		for _, m := range a.Modules {
			fmt.Printf("   %s | Type: %s | Lang: %s | Tests: %v\n",
				m.Name, m.Type, m.Language, m.HasTests)
		}

	case "patterns":
		fmt.Println("🔍 PATTERNS LAYER")
		for _, p := range a.Patterns {
			status := "✗"
			if p.Detected {
				status = "✓"
			}
			fmt.Printf("   %s %s: %.0f%% - %s\n", status, p.Name, p.Confidence*100, p.Description)
		}

	case "config":
		fmt.Printf("⚙️  CONFIG FILES (%d files)\n", len(a.ConfigFiles))
		for _, c := range a.ConfigFiles {
			fmt.Printf("   %s | Type: %s | Path: %s\n", c.Name, c.Type, c.Path)
		}

	default:
		fmt.Printf("Unknown layer: %s\n", layer)
		fmt.Println("Available layers: stack, arch, config, modules, patterns")
	}
}

func runBootstrap(projectPath string) {
	absPath, err := filepath.Abs(projectPath)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Bootstrapping agentic structure for: %s\n", absPath)
	fmt.Println(strings.Repeat("─", 50))

	analyzer := scanner.NewProjectAnalyzer(absPath)
	analysis, err := analyzer.Analyze()
	if err != nil {
		fmt.Printf("Error analyzing project: %v\n", err)
		os.Exit(1)
	}

	analysis.Name = filepath.Base(absPath)

	bootstrapDir := filepath.Join(absPath, ".ragnarok")
	os.MkdirAll(bootstrapDir, 0755)

	skillsConfig := scanner.GenerateSkillsConfig(analysis)
	rulesConfig := scanner.GenerateRulesConfig(analysis)
	standardsConfig := scanner.GenerateStandardsConfig(analysis)

	skillsFile := filepath.Join(bootstrapDir, "skills.json")
	skillsJSON, _ := json.MarshalIndent(skillsConfig, "", "  ")
	os.WriteFile(skillsFile, skillsJSON, 0644)
	fmt.Printf("✓ Created: %s\n", skillsFile)

	rulesFile := filepath.Join(bootstrapDir, "rules.json")
	rulesJSON, _ := json.MarshalIndent(rulesConfig, "", "  ")
	os.WriteFile(rulesFile, rulesJSON, 0644)
	fmt.Printf("✓ Created: %s\n", rulesFile)

	standardsFile := filepath.Join(bootstrapDir, "standards.json")
	standardsJSON, _ := json.MarshalIndent(standardsConfig, "", "  ")
	os.WriteFile(standardsFile, standardsJSON, 0644)
	fmt.Printf("✓ Created: %s\n", standardsFile)

	fmt.Println()
	fmt.Printf("Bootstrap complete!\n")
	fmt.Printf("Project: %s\n", analysis.Name)
	fmt.Printf("Stack: %s", analysis.Stack.Language)
	if analysis.Stack.Framework != "" {
		fmt.Printf(" + %s", analysis.Stack.Framework)
	}
	fmt.Println()
	fmt.Printf("Generated %d skills, %d rules, %d standards\n",
		len(skillsConfig["suggested_skills"].([]map[string]string)),
		len(rulesConfig),
		len(standardsConfig))
	fmt.Printf("\nFiles saved to: %s\n", bootstrapDir)
}
