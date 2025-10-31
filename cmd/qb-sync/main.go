package main


import (
	"context"
	"flag"
	"fmt"
	"log"
	"net"
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
	// Force IPv4 preference for all network operations
	forceIPv4()

	// Define command line flags
	var (
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
	cfg, err := config.LoadConfig()
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
	if cfg.Plex.Enabled {
		log.Printf("  Plex URL: %s", cfg.Plex.URL)
		log.Printf("  Plex enabled: true")
	} else {
		log.Printf("  Plex enabled: false")
	}

	// Create and run monitor
	monitor, err := worker.NewMonitor(cfg)
	if err != nil {
		log.Fatalf("Failed to create monitor: %v", err)
	}

	// Set up signal handling for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Run monitor in a goroutine
	go monitor.Run()

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

// min returns the minimum of two integers
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// forceIPv4 configures the application to prefer IPv4 network connections
func forceIPv4() {
	// Configure the default dialer to prefer IPv4
	net.DefaultResolver = &net.Resolver{
		PreferGo: true,
		Dial: func(ctx context.Context, network, address string) (net.Conn, error) {
			// Force IPv4 by using "tcp4" instead of "tcp"
			var dialer net.Dialer
			if network == "tcp" {
				network = "tcp4"
			} else if network == "udp" {
				network = "udp4"
			}
			return dialer.DialContext(ctx, network, address)
		},
	}

	log.Printf("Network configured to prefer IPv4 connections")
}