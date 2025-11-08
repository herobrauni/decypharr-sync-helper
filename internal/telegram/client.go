package telegram

import (
	"context"
	"fmt"
	"log"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// Bot represents the Telegram bot client
type Bot struct {
	api           *tgbotapi.BotAPI
	allowedUsers  map[int64]bool
	qbClient      QBClient
	isEnabled     bool
}

// QBClient interface for qBittorrent operations
type QBClient interface {
	GetAllTorrents(ctx context.Context) ([]TorrentInfo, error)
	AddTorrent(ctx context.Context, magnetLink, category string) error
}

// TorrentInfo represents basic torrent information for /status command
type TorrentInfo struct {
	Name     string
	Category string
	State    string
	Hash     string
}

// NewBot creates a new Telegram bot instance
func NewBot(token string, allowedUsers []int64, qbClient QBClient, enabled bool) (*Bot, error) {
	if !enabled {
		return &Bot{isEnabled: false}, nil
	}

	api, err := tgbotapi.NewBotAPI(token)
	if err != nil {
		return nil, fmt.Errorf("failed to create telegram bot: %w", err)
	}

	// Create allowed users map for efficient lookup
	allowedMap := make(map[int64]bool)
	for _, userID := range allowedUsers {
		allowedMap[userID] = true
	}

	bot := &Bot{
		api:          api,
		allowedUsers: allowedMap,
		qbClient:     qbClient,
		isEnabled:    true,
	}

	log.Printf("telegram: bot initialized (authorized as @%s)", api.Self.UserName)

	return bot, nil
}

// Start begins the bot's update handling loop
func (b *Bot) Start(ctx context.Context) error {
	if !b.isEnabled {
		log.Printf("telegram: bot disabled, skipping start")
		return nil
	}

	updateConfig := tgbotapi.NewUpdate(0)
	updateConfig.Timeout = 60

	updates := b.api.GetUpdatesChan(updateConfig)

	log.Printf("telegram: bot started, listening for updates")

	for {
		select {
		case <-ctx.Done():
			log.Printf("telegram: bot stopping due to context cancellation")
			return ctx.Err()
		case update, ok := <-updates:
			if !ok {
				log.Printf("telegram: updates channel closed")
				return nil
			}

			if update.Message != nil {
				b.handleMessage(ctx, update.Message)
			}
		}
	}
}

// handleMessage processes incoming messages
func (b *Bot) handleMessage(ctx context.Context, message *tgbotapi.Message) {
	// Check if user is authorized
	if !b.allowedUsers[message.From.ID] {
		log.Printf("telegram: unauthorized access attempt from user %d (%s)",
			message.From.ID, message.From.UserName)
		b.sendUnauthorizedMessage(message.Chat.ID)
		return
	}

	log.Printf("telegram: received command '%s' from user %d (%s)",
		message.Text, message.From.ID, message.From.UserName)

	// Handle commands and magnet links
	switch {
	case strings.HasPrefix(message.Text, "/status"):
		b.handleStatusCommand(ctx, message)
	case strings.HasPrefix(message.Text, "/add"):
		b.handleAddCommand(ctx, message)
	case strings.HasPrefix(message.Text, "magnet:"):
		// Auto-detect magnet links and add them
		b.handleMagnetLink(ctx, message)
	default:
		b.sendHelpMessage(message.Chat.ID)
	}
}

// handleMagnetLink handles raw magnet links sent without /add command
func (b *Bot) handleMagnetLink(ctx context.Context, message *tgbotapi.Message) {
	magnetLink := strings.TrimSpace(message.Text)

	// Validate magnet link
	if !b.isValidMagnetLink(magnetLink) {
		b.sendMessage(message.Chat.ID, "❌ *Error*\n\nInvalid magnet link format.\n\nMagnet links should start with `magnet:?` and contain `xt=urn:btih:`")
		return
	}

	// Extract hash from magnet link for logging
	hash := b.extractHashFromMagnet(magnetLink)
	logPrefix := hash
	if logPrefix == "" {
		logPrefix = "unknown"
	}

	log.Printf("telegram: adding torrent %s for user %d (auto-detected magnet link)", logPrefix, message.From.ID)

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

// isUserAllowed checks if a user is authorized to use the bot
func (b *Bot) isUserAllowed(userID int64) bool {
	return b.allowedUsers[userID]
}

// sendMessage sends a message to the specified chat
func (b *Bot) sendMessage(chatID int64, text string) {
	if !b.isEnabled {
		return
	}

	msg := tgbotapi.NewMessage(chatID, text)
	msg.ParseMode = "Markdown"

	if _, err := b.api.Send(msg); err != nil {
		log.Printf("telegram: failed to send message to chat %d: %v", chatID, err)
		log.Printf("telegram: message content that failed: %q", text)
	}
}

// sendUnauthorizedMessage sends an unauthorized access message
func (b *Bot) sendUnauthorizedMessage(chatID int64) {
	b.sendMessage(chatID, "⚠️ *Unauthorized Access*\n\nYou are not authorized to use this bot.")
}

// sendHelpMessage sends the help message with available commands
func (b *Bot) sendHelpMessage(chatID int64) {
	helpText := `🤖 *QB Sync Bot Commands*

/status - List all torrents with their names, categories, and states
/add \<magnet_link\> - Add a torrent using a magnet link

You can also send magnet links directly without the /add command.

Example:
` + "`" + `magnet:?xt=urn:btih:...` + "`"

	b.sendMessage(chatID, helpText)
}

// IsEnabled returns whether the bot is enabled
func (b *Bot) IsEnabled() bool {
	return b.isEnabled
}