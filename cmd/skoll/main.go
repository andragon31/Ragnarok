package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/andragon31/Ragnarok/internal/skoll/config"
	"github.com/andragon31/Ragnarok/internal/skoll/database"
	"github.com/andragon31/Ragnarok/internal/skoll/mcp"
)

var version = "1.2.0"

func main() {
	if len(os.Args) > 1 && os.Args[1] == "version" {
		fmt.Printf("Skoll v%s\n", version)
		fmt.Println("RSAW Orchestration Layer")
		return
	}

	serveCmd := flag.NewFlagSet("serve", flag.ExitOnError)
	port := serveCmd.Int("port", 7438, "MCP server port")
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
		fmt.Printf("Skoll v%s\n", version)
	default:
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println(`Skoll v1.0.0 - RSAW Orchestration Layer

Usage:
  skoll serve [--port PORT] [--dir DIR]
  skoll init --project NAME [--dir DIR]
  skoll version

Commands:
  serve    Start the MCP server
  init     Initialize a new project
  mcp      Run in MCP mode (stdio)
  version  Show version

Examples:
  skoll serve --port 7438
  skoll init --project "my-project"`)
}

func runServe(port int, dataDir string) {
	cfg, err := config.LoadConfig(dataDir)
	if err != nil {
		fmt.Printf("Error loading config: %v\n", err)
		os.Exit(1)
	}

	if port != 7438 {
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

	fmt.Printf("Skoll MCP server starting on port %d...\n", cfg.Port)
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
		dataDir = filepath.Join(home, ".skoll")
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

	fmt.Printf("✓ Initialized Skoll for project: %s\n", projectName)
	fmt.Printf("  Data directory: %s\n", cfg.DataDir)
	fmt.Printf("  Database: %s\n", cfg.DBPath())
	fmt.Printf("\nTo start the MCP server:\n  skoll serve\n")
}
