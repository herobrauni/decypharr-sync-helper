package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"qb-sync/internal/config"
	"qb-sync/internal/worker"
)

// Version information - can be set during build
var (
	Version   = "dev"
	BuildTime = "unknown"
	GitCommit = "unknown"
)

func main() {
	// Define command line flags
	var (
		configPath  = flag.String("config", "", "Path to configuration file")
		showVersion = flag.Bool("version", false, "Show version information and exit")
		dryRun      = flag.Bool("dry-run", false, "Run in dry-run mode (no actual file operations or deletions)")
	)
	flag.Parse()

	// Show version if requested
	if *showVersion {
		fmt.Printf("qb-sync %s\n", Version)
		fmt.Printf("Built: %s\n", BuildTime)
		fmt.Printf("Commit: %s\n", GitCommit)
		os.Exit(0)
	}

	// Load configuration
	cfg, err := config.LoadConfig(*configPath)
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Override dry-run if specified via command line
	if *dryRun {
		cfg.Monitor.DryRun = true
	}

	// Set up logging based on log level
	setLogLevel(cfg.Monitor.LogLevel)

	// Log startup information
	log.Printf("Starting qb-sync %s", Version)
	log.Printf("Configuration loaded:")
	log.Printf("  qBittorrent URL: %s", cfg.QB.BaseURL)
	log.Printf("  Category: %s", cfg.Monitor.Category)
	log.Printf("  Destination: %s", cfg.Monitor.DestPath)
	log.Printf("  Operation: %s", cfg.Monitor.Operation)
	log.Printf("  Poll interval: %v", cfg.Monitor.PollInterval)
	log.Printf("  Dry run: %t", cfg.Monitor.DryRun)

	// Create and run monitor
	monitor, err := worker.NewMonitor(cfg)
	if err != nil {
		log.Fatalf("Failed to create monitor: %v", err)
	}

	// Set up signal handling for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Run monitor in a goroutine
	go func() {
		if err := monitor.Run(); err != nil {
			log.Printf("Monitor error: %v", err)
		}
	}()

	// Wait for shutdown signal
	<-sigChan
	log.Println("Received shutdown signal, exiting...")
}

// setLogLevel configures the global logger based on the specified level
func setLogLevel(level string) {
	// For simplicity, we'll just use the standard logger
	// In a more sophisticated implementation, you might use a structured logger
	switch level {
	case "debug":
		log.SetFlags(log.LstdFlags | log.Lshortfile)
	case "info", "warn", "error":
		log.SetFlags(log.LstdFlags)
	default:
		log.SetFlags(log.LstdFlags)
	}
}