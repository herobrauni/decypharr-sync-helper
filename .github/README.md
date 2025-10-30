# GitHub Configuration

This directory contains GitHub-specific configuration files for the qb-sync project.

## Files

- `renovate.json5` - Renovate Bot configuration (JSON5 format with comments)
- `README.md` - This file

## Renovate Bot

This repository uses [Renovate Bot](https://github.com/renovatebot/renovate) for automated dependency management.

### Quick Setup

1. **Install Renovate App**: Install the [Renovate GitHub App](https://github.com/apps/renovate) on this repository
2. **Configure**: Edit the `renovate.json5` file as needed
3. **Wait**: Renovate will automatically create PRs for dependency updates

### Configuration Details

The `renovate.json5` file contains:

- **Schedule**: Weekend updates (Saturday/Sunday)
- **Timezone**: Europe/Vienna
- **Automerge**: Enabled for safe updates
- **Labels**: Dependencies are labeled with `dependencies` and `renovate`
- **Assignees**: @brauni is assigned to PRs
- **Reviewers**: @brauni is requested for review
- **Groups**: Related dependencies are grouped together
- **Security**: Immediate updates for security vulnerabilities

### Dependency Groups

- **Go dependencies**: All Go modules grouped together
- **Telegram Bot**: `github.com/go-telegram/bot` updates
- **Notifications**: `github.com/containrrr/shoutrrr` updates
- **Go x/* packages**: `golang.org/x/*` standard library extensions
- **GitHub Actions**: Actions in workflows (if any)

### Dashboard

Renovate creates a dependency dashboard issue showing:
- All pending dependency updates
- PR status overview
- Security vulnerabilities
- Update schedule

### Manual Operations

```bash
# Test configuration
npx renovate --dry-run

# Force run all updates
npx renovate --force

# Run specific dependency
npx renovate --package-name=github.com/go-telegram/bot
```

### Customization

Common modifications in `renovate.json5`:

```json5
{
  // Change update frequency
  schedule: ["every monday"],

  // Disable automerge
  automerge: false,

  // Add more reviewers
  reviewers: ["@user1", "@user2"],

  // Custom schedule for specific packages
  packageRules: [
    {
      matchPackageNames: ["critical-package"],
      schedule: ["daily"],
      automerge: false
    }
  ]
}
```

For full configuration options, see the [Renovate Documentation](https://docs.renovatebot.com/).