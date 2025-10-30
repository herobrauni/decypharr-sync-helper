package notification

import (
	"context"
	"fmt"
	"log"

	"github.com/containrrr/shoutrrr"

	"qb-sync/internal/config"
	"qb-sync/internal/qbit"
)

// EventType represents different notification event types
type EventType string

const (
	EventSuccess       EventType = "success"
	EventError         EventType = "error"
	EventPlexError     EventType = "plex_error"
	EventTorrentDelete EventType = "torrent_delete"
)

// Client represents a notification client
type Client struct {
	config *config.NotificationConfig
	logger *log.Logger
}

// NewClient creates a new notification client
func NewClient(cfg *config.NotificationConfig) (*Client, error) {
	if !cfg.Enabled || len(cfg.ShoutrrrURLs) == 0 {
		return nil, nil // Notification disabled
	}

	return &Client{
		config: cfg,
		logger: log.New(log.Writer(), "[notification] ", log.LstdFlags),
	}, nil
}

// SendNotification sends a notification if the event type is enabled
func (c *Client) SendNotification(ctx context.Context, eventType EventType, title, message string) error {
	if c == nil || !c.config.Enabled {
		return nil // Notifications disabled
	}

	// Check if this event type is enabled
	if !c.isEventEnabled(eventType) {
		return nil
	}

	// Build full message
	fullMessage := fmt.Sprintf("**%s**\n\n%s", title, message)

	c.logger.Printf("Sending %s notification: %s", eventType, title)

	// Send notification to all configured URLs
	for _, url := range c.config.ShoutrrrURLs {
		err := shoutrrr.Send(url, fullMessage)
		if err != nil {
			return fmt.Errorf("failed to send %s notification to %s: %w", eventType, url, err)
		}
	}

	c.logger.Printf("Successfully sent %s notification", eventType)
	return nil
}

// SendTorrentSuccess sends a notification for successfully processed torrent
func (c *Client) SendTorrentSuccess(ctx context.Context, torrent *qbit.Torrent, fileCount int, operation string) error {
	title := fmt.Sprintf("✅ Torrent Processed Successfully")
	message := fmt.Sprintf(
		"**Torrent:** %s\n**Operation:** %s\n**Files processed:** %d\n**Size:** %s\n**Category:** %s",
		torrent.Name,
		operation,
		fileCount,
		formatBytes(torrent.Size),
		torrent.Category,
	)

	return c.SendNotification(ctx, EventSuccess, title, message)
}

// SendTorrentError sends a notification for torrent processing errors
func (c *Client) SendTorrentError(ctx context.Context, torrent *qbit.Torrent, err error) error {
	title := fmt.Sprintf("❌ Torrent Processing Failed")
	message := fmt.Sprintf(
		"**Torrent:** %s\n**Error:** %s\n**Category:** %s",
		torrent.Name,
		err.Error(),
		torrent.Category,
	)

	return c.SendNotification(ctx, EventError, title, message)
}

// SendFileOperationError sends a notification for file operation errors
func (c *Client) SendFileOperationError(ctx context.Context, torrent *qbit.Torrent, fileName string, err error) error {
	title := fmt.Sprintf("❌ File Operation Failed")
	message := fmt.Sprintf(
		"**Torrent:** %s\n**File:** %s\n**Error:** %s\n**Category:** %s",
		torrent.Name,
		fileName,
		err.Error(),
		torrent.Category,
	)

	return c.SendNotification(ctx, EventError, title, message)
}

// SendPlexError sends a notification for Plex refresh errors
func (c *Client) SendPlexError(ctx context.Context, torrent *qbit.Torrent, err error) error {
	title := fmt.Sprintf("⚠️ Plex Refresh Failed")
	message := fmt.Sprintf(
		"**Torrent:** %s\n**Plex Error:** %s\n**Category:** %s",
		torrent.Name,
		err.Error(),
		torrent.Category,
	)

	return c.SendNotification(ctx, EventPlexError, title, message)
}

// SendTorrentDeleted sends a notification when a torrent is deleted
func (c *Client) SendTorrentDeleted(ctx context.Context, torrent *qbit.Torrent, deleteFiles bool) error {
	title := fmt.Sprintf("🗑️ Torrent Deleted")
	message := fmt.Sprintf(
		"**Torrent:** %s\n**Files deleted:** %t\n**Category:** %s",
		torrent.Name,
		deleteFiles,
		torrent.Category,
	)

	return c.SendNotification(ctx, EventTorrentDelete, title, message)
}

// SendGeneralError sends a notification for general errors
func (c *Client) SendGeneralError(ctx context.Context, err error) error {
	title := fmt.Sprintf("🚨 General Error")
	message := fmt.Sprintf(
		"**Error:** %s",
		err.Error(),
	)

	return c.SendNotification(ctx, EventError, title, message)
}

// isEventEnabled checks if the specified event type is enabled for notifications
func (c *Client) isEventEnabled(eventType EventType) bool {
	switch eventType {
	case EventSuccess:
		return c.config.OnSuccess
	case EventError:
		return c.config.OnError
	case EventPlexError:
		return c.config.OnPlexError
	case EventTorrentDelete:
		return c.config.OnTorrentDelete
	default:
		return false
	}
}

// formatBytes formats bytes into a human-readable string
func formatBytes(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

// IsEnabled returns true if notifications are properly configured and enabled
func (c *Client) IsEnabled() bool {
	return c != nil && c.config.Enabled
}