package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/andragon31/Ragnarok/internal/hati/config"
	"github.com/andragon31/Ragnarok/internal/hati/database"
	"github.com/andragon31/Ragnarok/internal/hati/mcp"
)

var version = "1.2.0"

func main() {
	if len(os.Args) > 1 && os.Args[1] == "version" {
		fmt.Printf("Hati v%s\n", version)
		fmt.Println("Task Planning & Human-in-the-Loop Layer")
		return
	}

	serveCmd := flag.NewFlagSet("serve", flag.ExitOnError)
	port := serveCmd.Int("port", 7439, "MCP server port")
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
		fmt.Printf("Hati v%s\n", version)
	default:
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println(`Hati v1.0.0 - Task Planning & Human-in-the-Loop Layer

Usage:
  hati serve [--port PORT] [--dir DIR]
  hati init --project NAME [--dir DIR]
  hati version

Commands:
  serve    Start the MCP server
  init     Initialize a new project
  mcp      Run in MCP mode (stdio)
  version  Show version

Examples:
  hati serve --port 7439
  hati init --project "my-project"`)
}

func runServe(port int, dataDir string) {
	cfg, err := config.LoadConfig(dataDir)
	if err != nil {
		fmt.Printf("Error loading config: %v\n", err)
		os.Exit(1)
	}

	if port != 7439 {
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

	fmt.Printf("Hati MCP server starting on port %d...\n", cfg.Port)
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
		dataDir = filepath.Join(home, ".hati")
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

	fmt.Printf("✓ Initialized Hati for project: %s\n", projectName)
	fmt.Printf("  Data directory: %s\n", cfg.DataDir)
	fmt.Printf("  Database: %s\n", cfg.DBPath())
	fmt.Printf("\nTo start the MCP server:\n  hati serve\n")
}
