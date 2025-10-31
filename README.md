# qb-sync

A Go-based application that monitors qBittorrent for completed torrents and performs file operations (hardlinks or copies) to destination directories, with optional Plex Media Server integration.

## Quick Start

### Required Environment Variables

You must set these environment variables for the application to run:

```bash
# qBittorrent connection
QB_SYNC_BASE_URL="http://localhost:8080"          # qBittorrent WebUI URL
QB_SYNC_CATEGORY="movies"                          # Torrent category to monitor
QB_SYNC_DEST_PATH="/data/movies"                   # Destination path for processed files
```

### Optional Environment Variables

```bash
# qBittorrent credentials (defaults to category name if not set)
QB_SYNC_USERNAME="qbittorrent"                     # qBittorrent username
QB_SYNC_PASSWORD="qbittorrent"                     # qBittorrent password

# Operation settings
QB_SYNC_POLL_INTERVAL="30s"                        # Polling interval (default: 30s)
QB_SYNC_OPERATION="hardlink"                       # "hardlink" (default) or "copy"
QB_SYNC_CROSS_DEVICE_FALLBACK="copy"               # "copy" (default) or "error"
QB_SYNC_PRESERVE_SUBFOLDER="true"                  # Preserve torrent subfolder structure (default: false)

# Torrent management
QB_SYNC_DELETE_TORRENT="true"                      # Delete torrent after processing (default: false)
QB_SYNC_DELETE_FILES="false"                       # Delete files with torrent (default: false)

# Application settings
QB_SYNC_DRY_RUN="false"                            # Enable dry-run mode (default: false)
QB_SYNC_LOG_LEVEL="info"                           # "debug", "info" (default), "warn", "error"

# Plex Media Server integration (optional)
QB_SYNC_PLEX_ENABLED="true"                        # Enable Plex integration (default: false)
QB_SYNC_PLEX_URL="http://localhost:32400"          # Plex server URL (default: http://localhost:32400)
QB_SYNC_PLEX_TOKEN="your_plex_token_here"          # Plex authentication token (required if enabled)
```

## Usage Examples

### Basic Usage
```bash
export QB_SYNC_BASE_URL="http://localhost:8080"
export QB_SYNC_CATEGORY="movies"
export QB_SYNC_DEST_PATH="/data/movies"

./qb-sync
```

### With Plex Integration
```bash
export QB_SYNC_BASE_URL="http://localhost:8080"
export QB_SYNC_CATEGORY="tv-shows"
export QB_SYNC_DEST_PATH="/data/tv-shows"
export QB_SYNC_PLEX_ENABLED="true"
export QB_SYNC_PLEX_TOKEN="your_plex_token"

./qb-sync
```

### Dry Run Mode
```bash
export QB_SYNC_BASE_URL="http://localhost:8080"
export QB_SYNC_CATEGORY="test"
export QB_SYNC_DEST_PATH="/tmp/test"

./qb-sync -dry-run
```

## Build

```bash
# Build the binary
go build -o qb-sync ./cmd/qb-sync/main.go

# Build for production with build info
go build -ldflags "-X main.Version=1.0.0 -X main.BuildTime=$(date -u +%Y-%m-%dT%H:%M:%SZ) -X main.GitCommit=$(git rev-parse HEAD)" -o qb-sync ./cmd/qb-sync/main.go
```

## How It Works

1. **Monitor**: Polls qBittorrent at regular intervals for completed torrents in the specified category
2. **Process**: Performs hardlinks (or copies) of torrent files to the destination directory
3. **Refresh**: Optionally triggers Plex library refreshes for the processed files
4. **Cleanup**: Optionally deletes torrents from qBittorrent after successful processing

## Features

- ✅ Resilient polling with exponential backoff
- ✅ Hardlinks with automatic cross-device fallback to copies
- ✅ Idempotent operations (skips existing files)
- ✅ Plex Media Server integration
- ✅ Graceful shutdown handling
- ✅ Dry run mode for safe testing
- ✅ Environment-based configuration
- ✅ IPv4 preference for network operations

## Requirements

- Go 1.21+
- qBittorrent with WebUI enabled
- Access to qBittorrent WebUI API
- (Optional) Plex Media Server with authentication token

## Docker

```bash
# Build Docker image
docker build -t qb-sync .

# Run with environment variables
docker run -e QB_SYNC_BASE_URL="http://qbit:8080" \
           -e QB_SYNC_CATEGORY="movies" \
           -e QB_SYNC_DEST_PATH="/data/movies" \
           -v /data/movies:/data/movies \
           qb-sync
```