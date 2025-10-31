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

	// Handle commands
	switch {
	case strings.HasPrefix(message.Text, "/status"):
		b.handleStatusCommand(ctx, message)
	case strings.HasPrefix(message.Text, "/add"):
		b.handleAddCommand(ctx, message)
	default:
		b.sendHelpMessage(message.Chat.ID)
	}
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
/add <magnet_link> - Add a torrent using a magnet link

Example:
/add magnet:?xt=urn:btih:...`

	b.sendMessage(chatID, helpText)
}

// IsEnabled returns whether the bot is enabled
func (b *Bot) IsEnabled() bool {
	return b.isEnabled
}