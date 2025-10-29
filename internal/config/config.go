package config

import (
	"fmt"
	"os"
	"strings"
	"time"
)

// Config represents the application configuration
type Config struct {
	QB           QBConfig
	Monitor      MonitorConfig
	Plex         PlexConfig
	Notification NotificationConfig
}

// QBConfig contains qBittorrent connection settings
type QBConfig struct {
	BaseURL               string
	Username              string
	Password              string
	TLSInsecureSkipVerify bool
}

// MonitorConfig contains monitoring and operation settings
type MonitorConfig struct {
	Category            string
	DestPath            string
	PollInterval        time.Duration
	Operation           string // hardlink|copy
	CrossDeviceFallback string // copy|error
	DeleteTorrent       bool
	DeleteFiles         bool
	PreserveSubfolder   bool
	DryRun             bool
	LogLevel            string
}

// PlexConfig contains Plex Media Server connection settings
type PlexConfig struct {
	URL     string
	Token   string
	Enabled bool
}

// NotificationConfig contains notification settings
type NotificationConfig struct {
	Enabled         bool
	ShoutrrrURLs    []string
	OnSuccess       bool
	OnError         bool
	OnPlexError     bool
	OnTorrentDelete bool
}

// LoadConfig loads configuration from environment variables only
func LoadConfig() (*Config, error) {
	// Initialize empty config
	var cfg Config

	// Apply environment variable overrides for QBConfig
	if baseURL := os.Getenv("QB_SYNC_BASE_URL"); baseURL != "" {
		cfg.QB.BaseURL = baseURL
	}
	if username := os.Getenv("QB_SYNC_USERNAME"); username != "" {
		cfg.QB.Username = username
	}
	if password := os.Getenv("QB_SYNC_PASSWORD"); password != "" {
		cfg.QB.Password = password
	}
	if tlsInsecure := os.Getenv("QB_SYNC_TLS_INSECURE_SKIP_VERIFY"); tlsInsecure != "" {
		cfg.QB.TLSInsecureSkipVerify = tlsInsecure == "true" || tlsInsecure == "1"
	}

	// Apply environment variable overrides for MonitorConfig
	if category := os.Getenv("QB_SYNC_CATEGORY"); category != "" {
		cfg.Monitor.Category = category
	}
	if destPath := os.Getenv("QB_SYNC_DEST_PATH"); destPath != "" {
		cfg.Monitor.DestPath = destPath
	}
	if pollInterval := os.Getenv("QB_SYNC_POLL_INTERVAL"); pollInterval != "" {
		if duration, err := time.ParseDuration(pollInterval); err == nil {
			cfg.Monitor.PollInterval = duration
		}
	}
	if operation := os.Getenv("QB_SYNC_OPERATION"); operation != "" {
		cfg.Monitor.Operation = operation
	}
	if crossDeviceFallback := os.Getenv("QB_SYNC_CROSS_DEVICE_FALLBACK"); crossDeviceFallback != "" {
		cfg.Monitor.CrossDeviceFallback = crossDeviceFallback
	}
	if deleteTorrent := os.Getenv("QB_SYNC_DELETE_TORRENT"); deleteTorrent != "" {
		cfg.Monitor.DeleteTorrent = deleteTorrent == "true" || deleteTorrent == "1"
	}
	if deleteFiles := os.Getenv("QB_SYNC_DELETE_FILES"); deleteFiles != "" {
		cfg.Monitor.DeleteFiles = deleteFiles == "true" || deleteFiles == "1"
	}
	if preserveSubfolder := os.Getenv("QB_SYNC_PRESERVE_SUBFOLDER"); preserveSubfolder != "" {
		cfg.Monitor.PreserveSubfolder = preserveSubfolder == "true" || preserveSubfolder == "1"
	}
	if dryRun := os.Getenv("QB_SYNC_DRY_RUN"); dryRun != "" {
		cfg.Monitor.DryRun = dryRun == "true" || dryRun == "1"
	}
	if logLevel := os.Getenv("QB_SYNC_LOG_LEVEL"); logLevel != "" {
		cfg.Monitor.LogLevel = logLevel
	}

	// Apply environment variable overrides for PlexConfig
	if plexURL := os.Getenv("QB_SYNC_PLEX_URL"); plexURL != "" {
		cfg.Plex.URL = plexURL
	}
	if plexToken := os.Getenv("QB_SYNC_PLEX_TOKEN"); plexToken != "" {
		cfg.Plex.Token = plexToken
	}
	if plexEnabled := os.Getenv("QB_SYNC_PLEX_ENABLED"); plexEnabled != "" {
		cfg.Plex.Enabled = plexEnabled == "true" || plexEnabled == "1"
	}

	// Apply environment variable overrides for NotificationConfig
	if notifEnabled := os.Getenv("QB_SYNC_NOTIFICATION_ENABLED"); notifEnabled != "" {
		cfg.Notification.Enabled = notifEnabled == "true" || notifEnabled == "1"
	}
	if shoutrrrURLs := os.Getenv("QB_SYNC_SHOUTRRR_URLS"); shoutrrrURLs != "" {
		cfg.Notification.ShoutrrrURLs = strings.Split(shoutrrrURLs, ",")
		// Trim whitespace from each URL
		for i, url := range cfg.Notification.ShoutrrrURLs {
			cfg.Notification.ShoutrrrURLs[i] = strings.TrimSpace(url)
		}
	}
	if onSuccess := os.Getenv("QB_SYNC_NOTIFICATION_ON_SUCCESS"); onSuccess != "" {
		cfg.Notification.OnSuccess = onSuccess == "true" || onSuccess == "1"
	}
	if onError := os.Getenv("QB_SYNC_NOTIFICATION_ON_ERROR"); onError != "" {
		cfg.Notification.OnError = onError == "true" || onError == "1"
	}
	if onPlexError := os.Getenv("QB_SYNC_NOTIFICATION_ON_PLEX_ERROR"); onPlexError != "" {
		cfg.Notification.OnPlexError = onPlexError == "true" || onPlexError == "1"
	}
	if onTorrentDelete := os.Getenv("QB_SYNC_NOTIFICATION_ON_TORRENT_DELETE"); onTorrentDelete != "" {
		cfg.Notification.OnTorrentDelete = onTorrentDelete == "true" || onTorrentDelete == "1"
	}

	// Set defaults (only for non-required fields)
	if cfg.Monitor.PollInterval == 0 {
		cfg.Monitor.PollInterval = 30 * time.Second
	}
	if cfg.Monitor.Operation == "" {
		cfg.Monitor.Operation = "hardlink"
	}
	if cfg.Monitor.CrossDeviceFallback == "" {
		cfg.Monitor.CrossDeviceFallback = "copy"
	}
	if cfg.Monitor.LogLevel == "" {
		cfg.Monitor.LogLevel = "info"
	}
	
	// Set optional QB defaults
	if cfg.QB.Username == "" {
		cfg.QB.Username = cfg.Monitor.Category
	}
	if cfg.QB.Password == "" {
		cfg.QB.Password = cfg.Monitor.Category
	}

	// Set optional Plex defaults
	if cfg.Plex.URL == "" {
		cfg.Plex.URL = "http://localhost:32400"
	}

	// Validate configuration
	if err := validateConfig(&cfg); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	return &cfg, nil
}

// validateConfig validates the configuration values
func validateConfig(cfg *Config) error {
	// Check required qBittorrent settings
	if cfg.QB.BaseURL == "" {
		return fmt.Errorf("qb.base_url is required (set via QB_SYNC_BASE_URL environment variable)")
	}

	// Check required monitor settings
	if cfg.Monitor.Category == "" {
		return fmt.Errorf("monitor.category is required (set via QB_SYNC_CATEGORY environment variable)")
	}
	if cfg.Monitor.DestPath == "" {
		return fmt.Errorf("monitor.dest_path is required (set via QB_SYNC_DEST_PATH environment variable)")
	}
	
	// Validate poll interval
	if cfg.Monitor.PollInterval <= 0 {
		return fmt.Errorf("monitor.poll_interval must be positive")
	}
	
	// Validate operation
	if cfg.Monitor.Operation != "hardlink" && cfg.Monitor.Operation != "copy" {
		return fmt.Errorf("monitor.operation must be 'hardlink' or 'copy'")
	}
	
	// Validate cross device fallback
	if cfg.Monitor.CrossDeviceFallback != "copy" && cfg.Monitor.CrossDeviceFallback != "error" {
		return fmt.Errorf("monitor.cross_device_fallback must be 'copy' or 'error'")
	}
	
	// Validate log level
	validLogLevels := map[string]bool{
		"debug": true,
		"info":  true,
		"warn":  true,
		"error": true,
	}
	if !validLogLevels[cfg.Monitor.LogLevel] {
		return fmt.Errorf("monitor.log_level must be one of: debug, info, warn, error")
	}
	
	// Validate Plex configuration if enabled
	if cfg.Plex.Enabled {
		if cfg.Plex.URL == "" {
			return fmt.Errorf("plex.url is required when plex.enabled is true (set via QB_SYNC_PLEX_URL environment variable)")
		}
		if cfg.Plex.Token == "" {
			return fmt.Errorf("plex.token is required when plex.enabled is true (set via QB_SYNC_PLEX_TOKEN environment variable)")
		}
	}

	// Validate Notification configuration if enabled
	if cfg.Notification.Enabled {
		if len(cfg.Notification.ShoutrrrURLs) == 0 {
			return fmt.Errorf("notification.shoutrrr_urls is required when notification.enabled is true (set via QB_SYNC_SHOUTRRR_URLS environment variable)")
		}
	}

	return nil
}