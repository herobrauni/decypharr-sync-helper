package worker

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"sync"
	"syscall"
	"time"

	"qb-sync/internal/config"
	"qb-sync/internal/files"
	"qb-sync/internal/plex"
	"qb-sync/internal/qbit"
)

// Monitor handles the polling and processing of torrents
type Monitor struct {
	client    *qbit.Client
	plexClient *plex.Client
	config    *config.Config
	logger    *log.Logger
	ctx       context.Context
	cancel    context.CancelFunc
	wg        sync.WaitGroup
	backoff   time.Duration
}

// NewMonitor creates a new monitor instance
func NewMonitor(cfg *config.Config) (*Monitor, error) {
	// Create qBittorrent client
	client, err := qbit.NewClient(&cfg.QB)
	if err != nil {
		return nil, fmt.Errorf("failed to create qBittorrent client: %w", err)
	}

	// Create Plex client if enabled
	var plexClient *plex.Client
	if cfg.Plex.Enabled {
		plexClient, err = plex.NewClient(&cfg.Plex)
		if err != nil {
			return nil, fmt.Errorf("failed to create Plex client: %w", err)
		}
	}

	// Create context for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())

	// Set up logger based on log level
	logger := log.New(os.Stdout, "[qb-sync] ", log.LstdFlags)

	return &Monitor{
		client:     client,
		plexClient: plexClient,
		config:     cfg,
		logger:     logger,
		ctx:        ctx,
		cancel:     cancel,
		backoff:    time.Second, // Initial backoff
	}, nil
}

// Run starts the monitoring loop
func (m *Monitor) Run() error {
	// Set up signal handling for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Initial login
	if err := m.loginWithRetry(); err != nil {
		return fmt.Errorf("failed to login: %w", err)
	}

	// Start monitoring goroutine
	m.wg.Add(1)
	go m.monitorLoop()

	// Wait for shutdown signal
	<-sigChan
	m.logger.Println("Shutdown signal received, stopping...")

	// Cancel context and wait for goroutines to finish
	m.cancel()
	m.wg.Wait()

	m.logger.Println("Shutdown complete")
	return nil
}

// monitorLoop runs the main polling loop
func (m *Monitor) monitorLoop() {
	defer m.wg.Done()

	ticker := time.NewTicker(m.config.Monitor.PollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-m.ctx.Done():
			return
		case <-ticker.C:
			m.logger.Printf("Polling for completed torrents (interval: %v)", m.config.Monitor.PollInterval)
			if err := m.processCompletedTorrents(); err != nil {
				m.logger.Printf("Error processing torrents: %v", err)
				// Increase backoff on error
				m.backoff = min(m.backoff*2, 2*time.Minute)
			} else {
				// Reset backoff on success
				m.backoff = time.Second
			}
		}
	}
}

// processCompletedTorrents fetches and processes completed torrents
func (m *Monitor) processCompletedTorrents() error {
	m.logger.Printf("Refreshing torrent list for category '%s'", m.config.Monitor.Category)
	
	// Get all torrents and filter in Go code
	allTorrents, err := m.client.ListAllTorrents(m.ctx)
	if err != nil {
		return fmt.Errorf("failed to list all torrents: %w", err)
	}

	m.logger.Printf("Found %d total torrents", len(allTorrents))

	// Filter for completed torrents in the configured category
	torrents := qbit.FilterCompletedTorrents(allTorrents, m.config.Monitor.Category)
	m.logger.Printf("Found %d completed torrents in category '%s'", len(torrents), m.config.Monitor.Category)
	
	if len(torrents) == 0 {
		m.logger.Printf("No completed torrents to process")
		return nil
	}

	// Log all found torrents
	for _, torrent := range torrents {
		m.logger.Printf("  - %s (hash: %s, progress: %.1f%%, state: %s, category: %s)",
			torrent.Name, torrent.Hash[:8], torrent.Progress*100, torrent.State, torrent.Category)
	}

	// Process each torrent
	for _, torrent := range torrents {
		m.logger.Printf("Processing torrent: %s", torrent.Name)
		if err := m.ProcessTorrent(&torrent); err != nil {
			m.logger.Printf("Error processing torrent '%s': %v", torrent.Name, err)
		} else {
			m.logger.Printf("Successfully processed torrent: %s", torrent.Name)
		}
	}

	return nil
}

// ProcessTorrent handles a single torrent
func (m *Monitor) ProcessTorrent(torrent *qbit.Torrent) error {
	m.logger.Printf("Processing torrent: %s (hash: %s)", torrent.Name, torrent.Hash)

	// Get file list for the torrent
	torrentFiles, err := m.client.FilesByHash(m.ctx, torrent.Hash)
	if err != nil {
		return fmt.Errorf("failed to get files for torrent: %w", err)
	}

	if len(torrentFiles) == 0 {
		m.logger.Printf("No files found for torrent: %s", torrent.Name)
		return nil
	}

	// Log all files found
	for _, file := range torrentFiles {
		m.logger.Printf("  File: %s (size: %d bytes, progress: %.1f%%)",
			file.Name, file.Size, file.Progress*100)
	}

	// Process each file
	var allSuccess bool = true
	var processedCount int
	for _, file := range torrentFiles {
		op, err := files.LinkOrCopy(&m.config.Monitor, torrent, &file)
		if err != nil {
			if !m.config.Monitor.DryRun {
				m.logger.Printf("Error preparing file operation for '%s': %v", file.Name, err)
			}
			allSuccess = false
			continue
		}

		if m.config.Monitor.DryRun {
			if op.Success {
				m.logger.Printf("[DRY RUN] Would %s %s to %s", m.config.Monitor.Operation, op.Source, op.Destination)
			} else {
				m.logger.Printf("[DRY RUN] Would fail to %s file '%s': %v", m.config.Monitor.Operation, file.Name, op.Error)
			}
			processedCount++
			continue
		}

		if !op.Success {
			m.logger.Printf("Failed to %s file '%s': %v", m.config.Monitor.Operation, file.Name, op.Error)
			allSuccess = false
		} else {
			m.logger.Printf("Successfully %s %s to %s", m.config.Monitor.Operation, op.Source, op.Destination)
			processedCount++
		}
	}

	m.logger.Printf("Processed %d/%d files for torrent '%s'", processedCount, len(torrentFiles), torrent.Name)

	// If all operations were successful and not in dry run mode, trigger Plex refresh and delete the torrent
	if !m.config.Monitor.DryRun && (allSuccess || len(torrentFiles) == 0) {
		// Trigger Plex refresh if enabled and we have processed files
		if m.config.Plex.Enabled && processedCount > 0 {
			if err := m.refreshPlexLibraries(torrent, torrentFiles); err != nil {
				m.logger.Printf("Failed to refresh Plex libraries for torrent '%s': %v", torrent.Name, err)
			}
		}

		if m.config.Monitor.DeleteTorrent {
			m.logger.Printf("Deleting torrent '%s' from qBittorrent (delete files: %t)", torrent.Name, m.config.Monitor.DeleteFiles)
			if err := m.client.DeleteTorrent(m.ctx, torrent.Hash, m.config.Monitor.DeleteFiles); err != nil {
				return fmt.Errorf("failed to delete torrent: %w", err)
			}
			m.logger.Printf("Successfully deleted torrent '%s' from qBittorrent", torrent.Name)
		} else {
			m.logger.Printf("Torrent deletion disabled, keeping '%s' in qBittorrent", torrent.Name)
		}
	} else if m.config.Monitor.DryRun {
		if m.config.Plex.Enabled && processedCount > 0 {
			m.logger.Printf("[DRY RUN] Would refresh Plex libraries for torrent '%s'", torrent.Name)
		}
		m.logger.Printf("[DRY RUN] Would delete torrent '%s' (delete files: %t)", torrent.Name, m.config.Monitor.DeleteFiles)
	}

	return nil
}

// loginWithRetry attempts to login with exponential backoff
func (m *Monitor) loginWithRetry() error {
	for {
		select {
		case <-m.ctx.Done():
			return m.ctx.Err()
		default:
			if err := m.client.Login(m.ctx); err != nil {
				m.logger.Printf("Login failed: %v, retrying in %v", err, m.backoff)
				time.Sleep(m.backoff)
				continue
			}
			m.logger.Println("Successfully logged in to qBittorrent")
			return nil
		}
	}
}

// min returns the minimum of two durations
func min(a, b time.Duration) time.Duration {
	if a < b {
		return a
	}
	return b
}

// refreshPlexLibraries triggers Plex library refreshes for the processed files
func (m *Monitor) refreshPlexLibraries(torrent *qbit.Torrent, torrentFiles []qbit.TorrentFile) error {
	if m.plexClient == nil {
		return fmt.Errorf("Plex client not initialized")
	}

	m.logger.Printf("Refreshing Plex libraries for torrent '%s'", torrent.Name)

	// Keep track of unique paths we've already refreshed to avoid duplicate refreshes
	refreshedPaths := make(map[string]bool)

	for _, file := range torrentFiles {
		// Build the destination path for this file
		destPath, err := files.BuildDestPath(&m.config.Monitor, torrent, &file)
		if err != nil {
			m.logger.Printf("Failed to build destination path for file '%s': %v", file.Name, err)
			continue
		}

		// Get the directory containing the file
		dirPath := filepath.Dir(destPath)
		
		// Skip if we've already refreshed this directory
		if refreshedPaths[dirPath] {
			continue
		}

		// Refresh the specific path in Plex
		if err := m.plexClient.RefreshPathForFile(m.ctx, destPath); err != nil {
			m.logger.Printf("Failed to refresh Plex path '%s': %v", dirPath, err)
			continue
		}

		m.logger.Printf("Successfully refreshed Plex path: %s", dirPath)
		refreshedPaths[dirPath] = true
	}

	m.logger.Printf("Completed Plex library refresh for torrent '%s'", torrent.Name)
	return nil
}
