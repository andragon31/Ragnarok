package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/ragnarok-ecosystem/tyr/internal/config"
	"github.com/ragnarok-ecosystem/tyr/internal/database"
	"github.com/ragnarok-ecosystem/tyr/internal/mcp"
)

var version = "1.0.0"

func main() {
	if len(os.Args) > 1 && os.Args[1] == "version" {
		fmt.Printf("Tyr v%s\n", version)
		fmt.Println("Security, Validation & Standards Layer")
		return
	}

	serveCmd := flag.NewFlagSet("serve", flag.ExitOnError)
	port := serveCmd.Int("port", 7440, "MCP server port")
	configDir := serveCmd.String("dir", "", "Data directory")

	initCmd := flag.NewFlagSet("init", flag.ExitOnError)
	initProject := initCmd.String("project", "", "Project name")

	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	switch os.Args[1] {
	case "serve":
		serveCmd.Parse(os.Args[2:])
		runServe(*port, *configDir)
	case "init":
		initCmd.Parse(os.Args[2:])
		if *initProject == "" {
			fmt.Println("Error: --project is required")
			initCmd.PrintDefaults()
			os.Exit(1)
		}
		runInit(*initProject, *configDir)
	case "version":
		fmt.Printf("Tyr v%s\n", version)
	default:
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println(`Tyr v1.0.0 - Security, Validation & Standards Layer

Usage:
  tyr serve [--port PORT] [--dir DIR]
  tyr init --project NAME [--dir DIR]
  tyr version

Commands:
  serve    Start the MCP server
  init     Initialize a new project
  mcp      Run in MCP mode (stdio)
  version  Show version

Examples:
  tyr serve --port 7440
  tyr init --project "my-project"`)
}

func runServe(port int, dataDir string) {
	cfg, err := config.LoadConfig(dataDir)
	if err != nil {
		fmt.Printf("Error loading config: %v\n", err)
		os.Exit(1)
	}

	if port != 7440 {
		cfg.Port = port
	}

	db, err := database.NewDB(cfg.DBPath())
	if err != nil {
		fmt.Printf("Error opening database: %v\n", err)
		os.Exit(1)
	}
	defer db.Close()

	if err := database.InitSchema(db); err != nil {
		fmt.Printf("Error initializing schema: %v\n", err)
		os.Exit(1)
	}

	server := mcp.NewServer(cfg, db)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigChan
		fmt.Println("\nShutting down...")
		cancel()
	}()

	fmt.Printf("Tyr MCP server starting on port %d...\n", cfg.Port)
	fmt.Printf("Data directory: %s\n", cfg.DataDir)
	fmt.Printf("Database: %s\n", cfg.DBPath())

	if err := server.Run(ctx); err != nil && err != context.Canceled {
		fmt.Printf("Server error: %v\n", err)
		os.Exit(1)
	}
}

func runInit(projectName, dataDir string) {
	if dataDir == "" {
		home, _ := os.UserHomeDir()
		dataDir = filepath.Join(home, ".tyr")
	}

	cfg, err := config.LoadConfig(dataDir)
	if err != nil {
		fmt.Printf("Error creating directory: %v\n", err)
		os.Exit(1)
	}

	db, err := database.NewDB(cfg.DBPath())
	if err != nil {
		fmt.Printf("Error opening database: %v\n", err)
		os.Exit(1)
	}
	defer db.Close()

	if err := database.InitSchema(db); err != nil {
		fmt.Printf("Error initializing schema: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("✓ Initialized Tyr for project: %s\n", projectName)
	fmt.Printf("  Data directory: %s\n", cfg.DataDir)
	fmt.Printf("  Database: %s\n", cfg.DBPath())
	fmt.Printf("\nTo start the MCP server:\n  tyr serve\n")
}
