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

	// Read config file
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file %s: %w", configPath, err)
	}

	// Parse YAML
	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	// Apply environment variable overrides
	if username := os.Getenv("QB_SYNC_USERNAME"); username != "" {
		cfg.QB.Username = username
	}
	if password := os.Getenv("QB_SYNC_PASSWORD"); password != "" {
		cfg.QB.Password = password
	}

	// Set defaults
	if cfg.QB.BaseURL == "" {
		cfg.QB.BaseURL = "http://localhost:8080"
	}
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

	// Validate configuration
	if err := validateConfig(&cfg); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	return &cfg, nil
}

// validateConfig validates the configuration values
func validateConfig(cfg *Config) error {
	if cfg.QB.BaseURL == "" {
		return fmt.Errorf("qb.base_url is required")
	}
	if cfg.QB.Username == "" {
		return fmt.Errorf("qb.username is required")
	}
	if cfg.QB.Password == "" {
		return fmt.Errorf("qb.password is required")
	}
	if cfg.Monitor.Category == "" {
		return fmt.Errorf("monitor.category is required")
	}
	if cfg.Monitor.DestPath == "" {
		return fmt.Errorf("monitor.dest_path is required")
	}
	if cfg.Monitor.PollInterval <= 0 {
		return fmt.Errorf("monitor.poll_interval must be positive")
	}
	if cfg.Monitor.Operation != "hardlink" && cfg.Monitor.Operation != "copy" {
		return fmt.Errorf("monitor.operation must be 'hardlink' or 'copy'")
	}
	if cfg.Monitor.CrossDeviceFallback != "copy" && cfg.Monitor.CrossDeviceFallback != "error" {
		return fmt.Errorf("monitor.cross_device_fallback must be 'copy' or 'error'")
	}
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