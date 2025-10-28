# qb-sync

A simple Go tool that monitors qBittorrent categories and processes completed torrents by hardlinking or copying files, then removing torrent from qBittorrent.

## Features

- Monitors a specific qBittorrent category for completed torrents
- Configurable file operations: hardlink or copy
- Cross-device fallback options for hardlinks
- Preserves file permissions and modification times
- Idempotent operations (skips already processed files)
- Dry-run mode for testing
- Graceful shutdown with signal handling
- YAML configuration with environment variable overrides
- TLS support with insecure skip option
- Fetches ALL torrents and filters in Go code for better control

## Configuration

The tool looks for configuration files in this order:
1. `./qb-sync.yaml`
2. `~/.config/qb-sync/config.yaml`
3. `/etc/qb-sync/config.yaml`

Or specify a custom path with `--config` flag.

### Environment Variables

- `QB_SYNC_CONFIG`: Path to configuration file
- `QB_SYNC_USERNAME`: Override qBittorrent username
- `QB_SYNC_PASSWORD`: Override qBittorrent password

### Sample Configuration

See `qb-sync.yaml` for a complete example:

```yaml
# qBittorrent connection settings
qb:
  base_url: http://localhost:8080
  username: admin
  password: adminadmin
  tls_insecure_skip_verify: false

# Monitoring and operation settings
monitor:
  category: Movies
  dest_path: /data/completed
  poll_interval: 30s
  operation: hardlink
  cross_device_fallback: copy
  delete_torrent: true
  delete_files: false
  preserve_subfolder: true
  dry_run: false
  log_level: info
```

## Usage

```bash
# Build
go build -o qb-sync ./cmd/qb-sync

# Run with default config
./qb-sync

# Run with dry-run mode
./qb-sync --dry-run

# Run with custom config
./qb-sync --config /path/to/config.yaml

# Show version
./qb-sync --version
```

## Docker Deployment

The tool includes Docker support for easy deployment:

```bash
# Build Docker image
docker build -t ghcr.io/herobrauni/decypharr-sync-helper:latest .

# Run with Docker
docker run --rm \
  -v /path/to/qb-sync.yaml:/config/qb-sync.yaml \
  -v /data/completed:/data/completed \
  ghcr.io/herobrauni/decypharr-sync-helper:latest
```

### Docker Compose

```yaml
version: '3.8'
services:
  qb-sync:
    image: ghcr.io/herobrauni/decypharr-sync-helper:latest
    volumes:
      - ./qb-sync.yaml:/config/qb-sync.yaml
      - /data/completed:/data/completed
    environment:
      - QB_SYNC_USERNAME=admin
      - QB_SYNC_PASSWORD=adminadmin
```

## Operation Modes

- **hardlink**: Creates hard links (default). Faster and space-efficient.
- **copy**: Copies files. Useful when hardlinks aren't desired.

When using hardlinks across different devices/filesystems, configure `cross_device_fallback`:
- **copy**: Automatically falls back to copying (default)
- **error**: Fails with an error

## Behavior

1. Logs into qBittorrent WebUI using cookie-based authentication
2. Fetches ALL torrents from qBittorrent API
3. Filters torrents in Go code for completed ones in specified category
4. For each completed torrent:
   - Skips files ending with `.!qB` (incomplete files)
   - Skips files already present at destination with same size
   - Performs hardlink or copy operation based on configuration
   - Preserves permissions and modification times
   - After successful processing, removes torrent from qBittorrent
5. Continues polling at configured interval
6. Supports graceful shutdown via SIGINT/SIGTERM

## Requirements

- Go 1.21+
- qBittorrent with WebUI enabled
- Access to qBittorrent WebUI API

## Building

```bash
go mod tidy
go build -o qb-sync ./cmd/qb-sync
```

## GitHub Actions

The project includes automated GitHub Actions for building and publishing multi-architecture Docker images:

- **Build Trigger**: Push to main/master or tags
- **Multi-arch Support**: Builds for linux/amd64 and linux/arm64
- **Registry**: Publishes to GitHub Container Registry (ghcr.io)
- **Image Tags**: Semantic versioning (vX.Y.Z)
- **Caching**: Uses GitHub Actions cache for faster builds

Images are available at: `ghcr.io/herobrauni/decypharr-sync-helper:latest`