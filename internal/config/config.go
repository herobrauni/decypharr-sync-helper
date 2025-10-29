package config

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"gopkg.in/yaml.v3"
)

// Config represents the application configuration
type Config struct {
	QB      QBConfig      `yaml:"qb"`
	Monitor MonitorConfig `yaml:"monitor"`
}

// QBConfig contains qBittorrent connection settings
type QBConfig struct {
	BaseURL               string `yaml:"base_url"`
	Username              string `yaml:"username"`
	Password              string `yaml:"password"`
	TLSInsecureSkipVerify bool   `yaml:"tls_insecure_skip_verify"`
}

// MonitorConfig contains monitoring and operation settings
type MonitorConfig struct {
	Category            string        `yaml:"category"`
	DestPath            string        `yaml:"dest_path"`
	PollInterval        time.Duration `yaml:"poll_interval"`
	Operation           string        `yaml:"operation"`           // hardlink|copy
	CrossDeviceFallback string        `yaml:"cross_device_fallback"` // copy|error
	DeleteTorrent       bool          `yaml:"delete_torrent"`
	DeleteFiles         bool          `yaml:"delete_files"`
	PreserveSubfolder   bool          `yaml:"preserve_subfolder"`
	DryRun             bool          `yaml:"dry_run"`
	LogLevel            string        `yaml:"log_level"`
}

// LoadConfig loads configuration from file with environment variable overrides
func LoadConfig(configPath string) (*Config, error) {
	// Determine config file path if not specified
	if configPath == "" {
		// Check environment variable first
		if envPath := os.Getenv("QB_SYNC_CONFIG"); envPath != "" {
			configPath = envPath
		} else {
			// Try default locations in order
			paths := []string{
				"./qb-sync.yaml",
				filepath.Join(os.Getenv("HOME"), ".config", "qb-sync", "config.yaml"),
				"/etc/qb-sync/config.yaml",
			}
			for _, p := range paths {
				if _, err := os.Stat(p); err == nil {
					configPath = p
					break
				}
			}
			// If no config found, use current directory
			if configPath == "" {
				configPath = "./qb-sync.yaml"
			}
		}
	}

	// Initialize empty config
	var cfg Config
	
	// Try to read config file
	data, err := os.ReadFile(configPath)
	if err != nil {
		// Config file doesn't exist, continue with empty config (will be populated by env vars)
	} else {
		// Parse YAML if file was successfully read
		if err := yaml.Unmarshal(data, &cfg); err != nil {
			return nil, fmt.Errorf("failed to parse config file: %w", err)
		}
	}

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
		return fmt.Errorf("qb.base_url is required (set via config file or QB_SYNC_BASE_URL environment variable)")
	}
	
	// Check required monitor settings
	if cfg.Monitor.Category == "" {
		return fmt.Errorf("monitor.category is required (set via config file or QB_SYNC_CATEGORY environment variable)")
	}
	if cfg.Monitor.DestPath == "" {
		return fmt.Errorf("monitor.dest_path is required (set via config file or QB_SYNC_DEST_PATH environment variable)")
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
	
	return nil
}