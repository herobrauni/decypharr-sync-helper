package telegram

import (
	"context"
	"fmt"
	"log"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// handleStatusCommand handles the /status command
func (b *Bot) handleStatusCommand(ctx context.Context, message *tgbotapi.Message) {
	torrents, err := b.qbClient.GetAllTorrents(ctx)
	if err != nil {
		log.Printf("telegram: failed to get torrents for /status: %v", err)
		b.sendMessage(message.Chat.ID, "❌ *Error*\n\nFailed to retrieve torrent status. Please try again later.")
		return
	}

	if len(torrents) == 0 {
		b.sendMessage(message.Chat.ID, "📊 *Torrent Status*\n\nNo torrents found.")
		return
	}

	// Build status message
	var statusText strings.Builder
	statusText.WriteString("📊 *Torrent Status*\n\n")

	// Group torrents by state for better organization
	torrentsByState := make(map[string][]TorrentInfo)
	for _, torrent := range torrents {
		torrentsByState[torrent.State] = append(torrentsByState[torrent.State], torrent)
	}

	// Define state order and icons
	stateOrder := []string{"downloading", "stalledDL", "stalledUP", "uploading", "completed", "pausedUP", "pausedDL", "error", "missingFiles"}
	stateIcons := map[string]string{
		"downloading": "⬇️",
		"stalledDL":   "⏸️",
		"stalledUP":   "⏸️",
		"uploading":   "⬆️",
		"completed":   "✅",
		"pausedUP":    "⏸️",
		"pausedDL":    "⏸️",
		"error":       "❌",
		"missingFiles": "❌",
	}

	// Display torrents by state
	for _, state := range stateOrder {
		if torrents, exists := torrentsByState[state]; exists && len(torrents) > 0 {
			icon := stateIcons[state]
			stateName := b.formatStateName(state)
			statusText.WriteString(fmt.Sprintf("%s *%s* (%d)\n", icon, stateName, len(torrents)))

			for i, torrent := range torrents {
				// Limit display to avoid message length limits
				if i >= 10 {
					statusText.WriteString(fmt.Sprintf("... and %d more\n", len(torrents)-i))
					break
				}

				// Truncate long names
				name := torrent.Name
				if len(name) > 50 {
					name = name[:47] + "..."
				}

				statusText.WriteString(fmt.Sprintf("• `%s`\n", name))

				if torrent.Category != "" {
					statusText.WriteString(fmt.Sprintf("  📁 Category: `%s`\n", torrent.Category))
				}
			}
			statusText.WriteString("\n")
		}
	}

	// Add summary
	statusText.WriteString(fmt.Sprintf("📈 *Summary*: %d total torrents", len(torrents)))

	b.sendMessage(message.Chat.ID, statusText.String())
}

// handleAddCommand handles the /add command
func (b *Bot) handleAddCommand(ctx context.Context, message *tgbotapi.Message) {
	// Extract magnet link from command
	parts := strings.Fields(message.Text)
	if len(parts) < 2 {
		b.sendMessage(message.Chat.ID, "❌ *Error*\n\nPlease provide a magnet link.\n\nUsage: `/add <magnet_link>`")
		return
	}

	magnetLink := strings.Join(parts[1:], " ")

	// Validate magnet link
	if !b.isValidMagnetLink(magnetLink) {
		b.sendMessage(message.Chat.ID, "❌ *Error*\n\nInvalid magnet link format.\n\nMagnet links should start with `magnet:?`")
		return
	}

	// Extract hash from magnet link for logging
	hash := b.extractHashFromMagnet(magnetLink)
	logPrefix := hash
	if logPrefix == "" {
		logPrefix = "unknown"
	}

	log.Printf("telegram: adding torrent %s for user %d", logPrefix, message.From.ID)

	// Add torrent using qBittorrent client
	err := b.qbClient.AddTorrent(ctx, magnetLink, "")
	if err != nil {
		log.Printf("telegram: failed to add torrent %s: %v", logPrefix, err)
		b.sendMessage(message.Chat.ID, "❌ *Error*\n\nFailed to add torrent. Please check the magnet link and try again.")
		return
	}

	log.Printf("telegram: successfully added torrent %s", logPrefix)

	// Send success message
	successText := fmt.Sprintf("✅ *Success*\n\nTorrent added successfully!\n\n*Hash:* `%s`", hash)
	if hash == "" {
		successText = "✅ *Success*\n\nTorrent added successfully!"
	}

	b.sendMessage(message.Chat.ID, successText)
}

// formatStateName converts qBittorrent state names to user-friendly names
func (b *Bot) formatStateName(state string) string {
	stateNames := map[string]string{
		"downloading":  "Downloading",
		"stalledDL":    "Stalled (Downloading)",
		"stalledUP":    "Stalled (Uploading)",
		"uploading":    "Uploading",
		"completed":    "Completed",
		"pausedUP":     "Paused (Uploading)",
		"pausedDL":     "Paused (Downloading)",
		"error":        "Error",
		"missingFiles": "Missing Files",
		"checkingDL":   "Checking (Downloading)",
		"checkingUP":   "Checking (Uploading)",
		"checkingResumeData": "Checking Resume Data",
		"moving":       "Moving",
		"queuedDL":     "Queued (Downloading)",
		"queuedUP":     "Queued (Uploading)",
		"forcedDL":     "Forced (Downloading)",
		"forcedUP":     "Forced (Uploading)",
		"allocating":   "Allocating",
		"metaDL":       "Downloading Metadata",
	}

	if name, exists := stateNames[state]; exists {
		return name
	}
	return state
}

// isValidMagnetLink checks if the provided string is a valid magnet link
func (b *Bot) isValidMagnetLink(link string) bool {
	return strings.HasPrefix(link, "magnet:?") && strings.Contains(link, "xt=urn:btih:")
}

// extractHashFromMagnet extracts the hash from a magnet link
func (b *Bot) extractHashFromMagnet(link string) string {
	// Look for xt=urn:btih: parameter
	parts := strings.Split(link, "xt=urn:btih:")
	if len(parts) < 2 {
		return ""
	}

	// Extract hash (until next parameter or end of string)
	hashAndMore := parts[1]
	if ampersandIndex := strings.Index(hashAndMore, "&"); ampersandIndex != -1 {
		return hashAndMore[:ampersandIndex]
	}
	return hashAndMore
}

// SendTorrentAddedNotification sends a notification when a torrent is added to Plex
func (b *Bot) SendTorrentAddedNotification(torrentName string) {
	if !b.isEnabled {
		return
	}

	message := fmt.Sprintf("🎉 *Torrent Added to Plex*\n\n*%s*\n\nThe torrent has been successfully processed and added to your Plex library!", torrentName)

	log.Printf("telegram: sending Plex addition notification for torrent: %s", torrentName)

	// Send notification to all allowed users
	for userID := range b.allowedUsers {
		b.sendMessage(userID, message)
	}
}