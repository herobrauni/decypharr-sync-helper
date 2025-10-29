# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

This is a Go-based application called `qb-sync` that monitors qBittorrent for completed torrents and performs file operations (hardlinks or copies) to destination directories, with optional Plex Media Server integration. The application is designed to run as a service that polls qBittorrent at regular intervals.

## Architecture

The application follows a modular structure:

- **cmd/qb-sync/main.go**: Entry point with CLI flags and signal handling
- **internal/config/**: Environment-based configuration management
- **internal/qbit/**: qBittorrent WebUI API client
- **internal/files/**: File operations (hardlink/copy with cross-device fallback)
- **internal/worker/**: Main monitoring logic and orchestration
- **internal/plex/**: Plex Media Server API client for library refreshes
- **internal/notification/**: Shoutrrr-based notification system

## Build and Development Commands

### Build
```bash
# Build the binary
go build -o qb-sync ./cmd/qb-sync/main.go

# Build for production with build info
go build -ldflags "-X main.Version=1.0.0 -X main.BuildTime=$(date -u +%Y-%m-%dT%H:%M:%SZ) -X main.GitCommit=$(git rev-parse HEAD)" -o qb-sync ./cmd/qb-sync/main.go
```

### Docker
```bash
# Build Docker image
docker build -t qb-sync .

# The multi-stage Dockerfile builds a distroless runtime image
```

### Testing and Development
```bash
# Run tests
go test ./...

# Run with verbose logging
QB_SYNC_LOG_LEVEL=debug ./qb-sync

# Dry run to test configuration
./qb-sync -dry-run
```

## Configuration

The application uses **environment variables only** for configuration (no config files):

### Required Variables
- `QB_SYNC_BASE_URL`: qBittorrent WebUI URL (e.g., http://localhost:8080)
- `QB_SYNC_CATEGORY`: Torrent category to monitor
- `QB_SYNC_DEST_PATH`: Destination path for processed files

### Optional Variables
- `QB_SYNC_USERNAME`/`QB_SYNC_PASSWORD`: qBittorrent credentials (defaults to category name)
- `QB_SYNC_POLL_INTERVAL`: Polling interval (default: 30s)
- `QB_SYNC_OPERATION`: "hardlink" (default) or "copy"
- `QB_SYNC_CROSS_DEVICE_FALLBACK`: "copy" (default) or "error"
- `QB_SYNC_DELETE_TORRENT`: Delete torrent after processing (default: false)
- `QB_SYNC_DELETE_FILES`: Delete files with torrent (default: false)
- `QB_SYNC_PRESERVE_SUBFOLDER`: Preserve torrent subfolder structure (default: false)
- `QB_SYNC_DRY_RUN`: Enable dry-run mode (default: false)
- `QB_SYNC_LOG_LEVEL`: "debug", "info", "warn", "error" (default: "info")

### Plex Integration (Optional)
- `QB_SYNC_PLEX_ENABLED`: Enable Plex integration (default: false)
- `QB_SYNC_PLEX_URL`: Plex server URL (default: http://localhost:32400)
- `QB_SYNC_PLEX_TOKEN`: Plex authentication token

### Notification Integration (Optional)
- `QB_SYNC_NOTIFICATION_ENABLED`: Enable notifications (default: false)
- `QB_SYNC_SHOUTRRR_URLS`: Comma-separated Shoutrrr URLs (e.g., `slack://token1/token2/token3,discord://webhook_id/webhook_token`)
- `QB_SYNC_NOTIFICATION_ON_SUCCESS`: Send notifications on successful torrent processing (default: false)
- `QB_SYNC_NOTIFICATION_ON_ERROR`: Send notifications on errors (default: true)
- `QB_SYNC_NOTIFICATION_ON_PLEX_ERROR`: Send notifications on Plex refresh failures (default: false)
- `QB_SYNC_NOTIFICATION_ON_TORRENT_DELETE`: Send notifications when torrents are deleted (default: false)

## Key Features

1. **Resilient Polling**: Exponential backoff retry logic for API failures
2. **File Operations**: Hardlinks with automatic cross-device fallback to copies
3. **Idempotency**: Skips files that already exist with correct size
4. **Plex Integration**: Automatic library refreshes for processed files
5. **Notification System**: Configurable notifications via Shoutrrr for various events
6. **Graceful Shutdown**: Proper signal handling and cleanup
7. **Dry Run Mode**: Safe testing without actual file operations

## Notification Setup

The application uses Shoutrrr for notifications, supporting many platforms:

### Example Notification URLs

**Slack:**
```
slack://token1/token2/token3
```

**Discord:**
```
discord://webhook_id/webhook_token
```

**Telegram:**
```
telegram://bot_token/chat_id
```

**Email:**
```
smtp://username:password@host:port/?from=from@example.com&to=to@example.com
```

Multiple URLs can be combined with commas:
```
QB_SYNC_SHOUTRRR_URLS="slack://token1/token2/token3,discord://webhook_id/webhook_token"
```

## Common Issues

- qBittorrent WebUI must be enabled and accessible
- Ensure the qBittorrent user has permission to access torrent categories
- Cross-device hardlinks require filesystem support or fallback configuration
- Plex tokens are required for library refreshes when enabled
- Notification URLs must be properly formatted for the respective service

## Dependencies

- Go 1.21+
- github.com/containrrr/shoutrrr for notifications
- gopkg.in/yaml.v3 for configuration (though currently only env vars are used)