package main


import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"qb-sync/internal/config"
	"qb-sync/internal/qbit"
	"qb-sync/internal/telegram"
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
	if cfg.Notification.Enabled {
		log.Printf("  Notifications enabled: true")
		log.Printf("  Notification URLs: %d configured", len(cfg.Notification.ShoutrrrURLs))
		log.Printf("  Notify on success: %t", cfg.Notification.OnSuccess)
		log.Printf("  Notify on error: %t", cfg.Notification.OnError)
		log.Printf("  Notify on Plex error: %t", cfg.Notification.OnPlexError)
		log.Printf("  Notify on torrent delete: %t", cfg.Notification.OnTorrentDelete)
	} else {
		log.Printf("  Notifications enabled: false")
	}
	if cfg.Telegram.Enabled {
		log.Printf("  Telegram enabled: true")
		log.Printf("  Telegram bot will use category: %s", cfg.Monitor.Category)
		if len(cfg.Telegram.AllowedUserIDs) > 0 {
			log.Printf("  Telegram allowed users: %v", cfg.Telegram.AllowedUserIDs)
		} else {
			log.Printf("  Telegram allowed users: all users")
		}
	} else {
		log.Printf("  Telegram enabled: false")
	}

	// Create context for bot
	ctx := context.Background()

	// Create Telegram bot if enabled
	var telegramBot *telegram.Bot
	if cfg.Telegram.Enabled {
		qbClient, err := qbit.NewClient(&cfg.QB)
		if err != nil {
			log.Printf("Failed to create qBittorrent client for Telegram bot: %v", err)
		} else {
			telegramBot, err = telegram.NewBot(ctx, cfg.Telegram.Token, qbClient, &cfg.Telegram, cfg.Monitor.Category)
			if err != nil {
				log.Printf("Failed to create Telegram bot: %v", err)
			} else {
				log.Printf("Telegram bot created successfully")
				// Start bot in a goroutine
				go telegramBot.Start(ctx)
			}
		}
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