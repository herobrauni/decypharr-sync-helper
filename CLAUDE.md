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

### Telegram Bot Integration (Optional)
- `QB_SYNC_TELEGRAM_ENABLED`: Enable Telegram bot (default: false)
- `QB_SYNC_TELEGRAM_TOKEN`: Telegram bot token (required if enabled)
- `QB_SYNC_TELEGRAM_ALLOWED_USERS`: Comma-separated list of allowed Telegram user IDs (optional, defaults to all users)
- `QB_SYNC_TELEGRAM_ADMIN_CHAT_ID`: Telegram chat ID to receive admin notifications (optional)

## Key Features

1. **Resilient Polling**: Exponential backoff retry logic for API failures
2. **File Operations**: Hardlinks with automatic cross-device fallback to copies
3. **Idempotency**: Skips files that already exist with correct size
4. **Plex Integration**: Automatic library refreshes for processed files
5. **Graceful Shutdown**: Proper signal handling and cleanup
6. **Dry Run Mode**: Safe testing without actual file operations
7. **Telegram Bot**: Add torrents via Telegram messages using magnet links or .torrent files with admin notifications

## Telegram Bot Setup

The application includes a Telegram bot that can receive magnet links and .torrent files and add them to qBittorrent.

### Setting up a Telegram Bot

1. **Create a Bot:**
   - Talk to @BotFather on Telegram
   - Use `/newbot` command to create a new bot
   - Get the bot token (looks like `1234567890:ABCdefGHIjklMNOpqrsTUVwxyz`)

2. **Get Your User ID (Optional):**
   - Talk to @userinfobot on Telegram to get your user ID
   - This is used to restrict access to your bot

3. **Configure Environment Variables:**
   ```bash
   export QB_SYNC_TELEGRAM_ENABLED=true
   export QB_SYNC_TELEGRAM_TOKEN="your_bot_token_here"
   export QB_SYNC_TELEGRAM_ALLOWED_USERS="123456789,987654321"  # Optional
   ```

### Telegram Bot Commands

- `/start` - Show welcome message
- `/help` - Show help information
- `/status` - Show qBittorrent status
- Send magnet links directly to add torrents
- Upload .torrent files to add torrents

All torrents added via Telegram will be automatically assigned to the configured category.

## Common Issues

- qBittorrent WebUI must be enabled and accessible
- Ensure the qBittorrent user has permission to access torrent categories
- Cross-device hardlinks require filesystem support or fallback configuration
- Plex tokens are required for library refreshes when enabled
- Notification URLs must be properly formatted for the respective service

## Dependencies

- Go 1.21+
- github.com/containrrr/shoutrrr for notifications
- github.com/go-telegram/bot for Telegram bot integration
- gopkg.in/yaml.v3 for configuration (though currently only env vars are used)

## Dependency Management

This project uses [Renovate Bot](https://github.com/renovatebot/renovate) for automated dependency management.

### Renovate Configuration

- **Configuration Files**:
  - `renovate.json` - Main configuration (JSON format)
  - `.github/renovate.json5` - Alternative configuration (JSON5 format)
- **Schedule**: Runs every weekend (Saturday/Sunday)
- **Timezone**: Europe/Vienna
- **Automerge**: Enabled for most dependency updates
- **Grouping**: Dependencies are grouped by type (Go deps, GitHub Actions, etc.)
- **Security**: Vulnerability alerts are processed immediately

### Renovate Features

- **Dependency Dashboard**: Creates an issue with dependency overview
- **Lock File Maintenance**: Monthly go.mod tidy and cleanup
- **Security Updates**: Immediate PRs for security vulnerabilities
- **Auto-merge**: Safe updates are merged automatically
- **Grouped PRs**: Related dependencies are updated together
- **Go Modules**: Full support with `go mod tidy` after updates

### Manual Renovate Runs

If you need to run Renovate manually:

```bash
# Using Docker (recommended)
docker run -it --rm \
  -v "$(pwd):/workspace" \
  -e LOG_LEVEL=debug \
  renovate/renovate

# Using npm
npx renovate

# Local development runs
renovate --dry-run
```

### Configuration Customization

To customize Renovate behavior:

1. Edit `renovate.json` or `.github/renovate.json5`
2. Test configuration: `renovate --dry-run`
3. Push changes to trigger a Renovate run

Common customizations:
- Change `schedule` for different update frequency
- Modify `automerge` settings for your workflow
- Adjust `assignees` and `reviewers` for your team
- Add custom `packageRules` for specific dependencies