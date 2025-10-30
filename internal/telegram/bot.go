package telegram

import (
	"context"
	"fmt"
	"log"
	"regexp"
	"strings"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
	"qb-sync/internal/config"
	"qb-sync/internal/qbit"
)

// Bot represents the Telegram bot
type Bot struct {
	bot     *bot.Bot
	qbClient *qbit.Client
	config  *config.TelegramConfig
	category string // Will use the monitor category
}

// NewBot creates a new Telegram bot instance
func NewBot(ctx context.Context, token string, qbClient *qbit.Client, cfg *config.TelegramConfig, category string) (*Bot, error) {
	telegramBot := &Bot{
		qbClient: qbClient,
		config:   cfg,
		category: category,
	}

	b, err := bot.New(token,
		bot.WithMessageTextHandler("/start", bot.MatchTypeExact, telegramBot.handleStart),
		bot.WithMessageTextHandler("/help", bot.MatchTypeExact, telegramBot.handleHelp),
		bot.WithMessageTextHandler("/status", bot.MatchTypeExact, telegramBot.handleStatus),
		bot.WithMessageTextHandler("", bot.MatchTypePrefix, telegramBot.handleTorrentMessage),
		bot.WithDefaultHandler(telegramBot.handleDefault),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create telegram bot: %w", err)
	}

	telegramBot.bot = b

	return telegramBot, nil
}

// Start starts the Telegram bot
func (b *Bot) Start(ctx context.Context) {
	log.Printf("Starting Telegram bot...")
	b.bot.Start(ctx)
}

// handleStart handles the /start command
func (b *Bot) handleStart(ctx context.Context, api *bot.Bot, update *models.Update) {
	b.sendStartMessage(ctx, update.Message.From.ID)
}

// handleHelp handles the /help command
func (b *Bot) handleHelp(ctx context.Context, api *bot.Bot, update *models.Update) {
	b.sendHelpMessage(ctx, update.Message.From.ID)
}

// handleStatus handles the /status command
func (b *Bot) handleStatus(ctx context.Context, api *bot.Bot, update *models.Update) {
	b.sendStatusMessage(ctx, update.Message.From.ID)
}

// handleDefault handles all other message types (including documents)
func (b *Bot) handleDefault(ctx context.Context, api *bot.Bot, update *models.Update) {
	// Check if this is a document message
	if update.Message.Document != nil {
		b.handleTorrentFile(ctx, api, update)
	}
}

// sendStartMessage sends the welcome message
func (b *Bot) sendStartMessage(ctx context.Context, chatID int64) {
	msg := "🎬 *Welcome to qb-sync Bot!*\n\n" +
		"I can help you add torrents to qBittorrent via Telegram.\n\n" +
		"*Commands:*\n" +
		"/start - Show this welcome message\n" +
		"/help - Show help information\n" +
		"/status - Show qBittorrent status\n\n" +
		"*Usage:*\n" +
		"• Send me a magnet link\n" +
		"• Send me a .torrent file\n\n" +
		"All torrents will be added to the configured category: `" + b.category + "`"

	b.bot.SendMessage(ctx, &bot.SendMessageParams{
		ChatID:    chatID,
		Text:      msg,
		ParseMode: models.ParseModeMarkdown,
	})
}

// sendHelpMessage sends the help message
func (b *Bot) sendHelpMessage(ctx context.Context, chatID int64) {
	msg := "📚 *qb-sync Bot Help*\n\n" +
		"*Commands:*\n" +
		"/start - Welcome message\n" +
		"/help - Show this help\n" +
		"/status - Show qBittorrent status\n\n" +
		"*Adding Torrents:*\n" +
		"1. **Magnet Links:** Simply paste and send any magnet link\n" +
		"2. **Torrent Files:** Upload .torrent files directly\n\n" +
		"*Example Magnet Link:*\n" +
		"`magnet:?xt=urn:btih:example...`\n\n" +
		"All torrents are automatically added to category: `" + b.category + "`"

	b.bot.SendMessage(ctx, &bot.SendMessageParams{
		ChatID:    chatID,
		Text:      msg,
		ParseMode: models.ParseModeMarkdown,
	})
}

// sendStatusMessage sends the qBittorrent status
func (b *Bot) sendStatusMessage(ctx context.Context, chatID int64) {
	// Get torrents from qBittorrent
	torrents, err := b.qbClient.ListAllTorrents(ctx)
	if err != nil {
		b.bot.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: chatID,
			Text:   "❌ Failed to connect to qBittorrent: " + err.Error(),
		})
		return
	}

	// Count torrents by status
	totalTorrents := len(torrents)
	completedTorrents := 0
	downloadingTorrents := 0
	categoryTorrents := 0

	for _, torrent := range torrents {
		if torrent.Progress == 1.0 && !isTransitionalState(torrent.State) {
			completedTorrents++
		} else if torrent.Progress < 1.0 {
			downloadingTorrents++
		}
		if torrent.Category == b.category {
			categoryTorrents++
		}
	}

	msg := fmt.Sprintf("📊 *qBittorrent Status*\n\n"+
		"🔢 *Total Torrents:* %d\n"+
		"✅ *Completed:* %d\n"+
		"⬇️ *Downloading:* %d\n"+
		"📁 *In Category '%s':* %d",
		totalTorrents, completedTorrents, downloadingTorrents, b.category, categoryTorrents)

	b.bot.SendMessage(ctx, &bot.SendMessageParams{
		ChatID:    chatID,
		Text:      msg,
		ParseMode: models.ParseModeMarkdown,
	})
}

// handleTorrentMessage processes messages containing magnet links
func (b *Bot) handleTorrentMessage(ctx context.Context, api *bot.Bot, update *models.Update) {
	if update.Message.Text == "" {
		return
	}

	// Check if user is authorized
	if !b.isAuthorized(update.Message.From.ID) {
		api.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: update.Message.Chat.ID,
			Text:   "❌ You are not authorized to use this bot.",
		})
		return
	}

	// Extract magnet links from the message
	magnetLinks := extractMagnetLinks(update.Message.Text)

	if len(magnetLinks) == 0 {
		return // No magnet links found, let other handlers deal with it
	}

	// Process each magnet link
	for _, magnetLink := range magnetLinks {
		b.processMagnetLink(ctx, update.Message.Chat.ID, magnetLink)
	}
}

// handleTorrentFile processes uploaded torrent files
func (b *Bot) handleTorrentFile(ctx context.Context, api *bot.Bot, update *models.Update) {
	// Check if user is authorized
	if !b.isAuthorized(update.Message.From.ID) {
		api.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: update.Message.Chat.ID,
			Text:   "❌ You are not authorized to use this bot.",
		})
		return
	}

	document := update.Message.Document
	if document == nil {
		return
	}

	// Check if it's a torrent file
	if !strings.HasSuffix(document.FileName, ".torrent") {
		api.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: update.Message.Chat.ID,
			Text:   "❌ Please upload a valid .torrent file.",
		})
		return
	}

	// Get file info
	fileInfo := bot.GetFileParams{
		FileID: document.FileID,
	}

	file, err := api.GetFile(ctx, &fileInfo)
	if err != nil {
		api.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: update.Message.Chat.ID,
			Text:   "❌ Failed to get file information: " + err.Error(),
		})
		return
	}

	// Download the file
	fileURL := fmt.Sprintf("https://api.telegram.org/file/bot%s/%s", api.Token, file.FilePath)

	// Add torrent from file URL
	b.processTorrentFile(ctx, update.Message.Chat.ID, fileURL, document.FileName)
}

// processMagnetLink adds a magnet link to qBittorrent
func (b *Bot) processMagnetLink(ctx context.Context, chatID int64, magnetLink string) {
	// First login to qBittorrent
	if err := b.qbClient.Login(ctx); err != nil {
		b.bot.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: chatID,
			Text:   "❌ Failed to connect to qBittorrent: " + err.Error(),
		})
		return
	}

	// Add the torrent
	err := b.qbClient.AddTorrentFromMagnet(ctx, magnetLink, b.category)
	if err != nil {
		b.bot.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: chatID,
			Text:   "❌ Failed to add torrent: " + err.Error(),
		})
		return
	}

	// Send success message
	msg := fmt.Sprintf("✅ *Torrent Added Successfully!*\n\n"+
		"🧭 **Magnet Link added to qBittorrent**\n"+
		"📁 **Category:** `%s`\n"+
		"🔗 **Link:** `%s...`",
		b.category, magnetLink[:min(50, len(magnetLink))])

	b.bot.SendMessage(ctx, &bot.SendMessageParams{
		ChatID:    chatID,
		Text:      msg,
		ParseMode: models.ParseModeMarkdown,
	})
}

// processTorrentFile adds a torrent file to qBittorrent
func (b *Bot) processTorrentFile(ctx context.Context, chatID int64, fileURL, fileName string) {
	// First login to qBittorrent
	if err := b.qbClient.Login(ctx); err != nil {
		b.bot.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: chatID,
			Text:   "❌ Failed to connect to qBittorrent: " + err.Error(),
		})
		return
	}

	// Add the torrent from file
	err := b.qbClient.AddTorrentFromFile(ctx, fileURL, b.category)
	if err != nil {
		b.bot.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: chatID,
			Text:   "❌ Failed to add torrent file: " + err.Error(),
		})
		return
	}

	// Send success message
	msg := fmt.Sprintf("✅ *Torrent File Added Successfully!*\n\n"+
		"📄 **File:** `%s`\n"+
		"📁 **Category:** `%s`",
		fileName, b.category)

	b.bot.SendMessage(ctx, &bot.SendMessageParams{
		ChatID:    chatID,
		Text:      msg,
		ParseMode: models.ParseModeMarkdown,
	})
}

// isAuthorized checks if a user is authorized to use the bot
func (b *Bot) isAuthorized(userID int64) bool {
	// If no authorized users are configured, allow everyone
	if len(b.config.AllowedUserIDs) == 0 {
		return true
	}

	// Check if user ID is in the allowed list
	for _, allowedID := range b.config.AllowedUserIDs {
		if allowedID == userID {
			return true
		}
	}
	return false
}

// extractMagnetLinks extracts magnet links from text using regex
func extractMagnetLinks(text string) []string {
	// Magnet link regex pattern
	magnetRegex := regexp.MustCompile(`magnet:\?xt=urn:btih:[a-fA-F0-9]{40}[^\\s]*`)

	matches := magnetRegex.FindAllString(text, -1)
	return matches
}

// isTransitionalState checks if a torrent is in a transitional state
func isTransitionalState(state string) bool {
	transitionalStates := []string{
		"checkingDL", "checkingUP", "checkingResumeData",
		"moving", "metaDL", "allocating",
	}
	for _, s := range transitionalStates {
		if state == s {
			return true
		}
	}
	return false
}

// min returns the minimum of two integers
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}