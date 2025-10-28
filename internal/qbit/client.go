package qbit

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"strings"
	"time"

	"qb-sync/internal/config"
)

// Torrent represents a torrent from qBittorrent
type Torrent struct {
	Hash         string  `json:"hash"`
	Name         string  `json:"name"`
	State        string  `json:"state"`
	Progress     float64 `json:"progress"`
	SavePath     string  `json:"save_path"`
	ContentPath  string  `json:"content_path"`
	Size         int64   `json:"size"`
	Completed    int64   `json:"completed"`
	CompletionOn int64  `json:"completion_on"`
}

// TorrentFile represents a file within a torrent
type TorrentFile struct {
	Name     string `json:"name"`
	Size     int64  `json:"size"`
	Progress float64 `json:"progress"`
	Priority int    `json:"priority"`
	IsSeed   bool   `json:"is_seed"`
}

// Client represents a qBittorrent WebUI client
type Client struct {
	httpClient *http.Client
	baseURL    *url.URL
	config     *config.QBConfig
}

// NewClient creates a new qBittorrent client
func NewClient(cfg *config.QBConfig) (*Client, error) {
	baseURL, err := url.Parse(cfg.BaseURL)
	if err != nil {
		return nil, fmt.Errorf("invalid base URL: %w", err)
	}

	// Create HTTP client with cookie jar for session management
	jar, err := cookiejar.New(nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create cookie jar: %w", err)
	}

	// Configure TLS settings
	tlsConfig := &tls.Config{
		InsecureSkipVerify: cfg.TLSInsecureSkipVerify,
	}

	transport := &http.Transport{
		TLSClientConfig: tlsConfig,
	}

	httpClient := &http.Client{
		Jar:       jar,
		Transport: transport,
		Timeout:   30 * time.Second,
	}

	return &Client{
		httpClient: httpClient,
		baseURL:    baseURL,
		config:     cfg,
	}, nil
}

// Login authenticates with qBittorrent WebUI
func (c *Client) Login(ctx context.Context) error {
	loginURL := c.baseURL.ResolveReference(&url.URL{Path: "/api/v2/auth/login"})

	// Prepare form data
	data := fmt.Sprintf("username=%s&password=%s",
		url.QueryEscape(c.config.Username),
		url.QueryEscape(c.config.Password))

	req, err := http.NewRequestWithContext(ctx, "POST", loginURL.String(), strings.NewReader(data))
	if err != nil {
		return fmt.Errorf("failed to create login request: %w", err)
	}

	// Set required headers
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Referer", c.baseURL.String())
	req.Header.Set("Origin", c.baseURL.String())

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to perform login request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("login failed with status: %s", resp.Status)
	}

	// Check response body for success
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read login response: %w", err)
	}

	if string(body) != "Ok." {
		return fmt.Errorf("login failed: %s", string(body))
	}

	return nil
}

// ListCompletedByCategory retrieves completed torrents for a specific category
func (c *Client) ListCompletedByCategory(ctx context.Context, category string) ([]Torrent, error) {
	listURL := c.baseURL.ResolveReference(&url.URL{
		Path: "/api/v2/torrents/info",
		RawQuery: fmt.Sprintf("filter=completed&category=%s", url.QueryEscape(category)),
	})

	req, err := http.NewRequestWithContext(ctx, "GET", listURL.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create list request: %w", err)
	}

	// Set required headers
	req.Header.Set("Referer", c.baseURL.String())
	req.Header.Set("Origin", c.baseURL.String())

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to perform list request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("list torrents failed with status: %s", resp.Status)
	}

	var torrents []Torrent
	if err := decodeJSON(resp.Body, &torrents); err != nil {
		return nil, fmt.Errorf("failed to decode torrent list: %w", err)
	}

	// Filter for truly completed torrents (progress == 1.0 and not in transitional states)
	var completed []Torrent
	for _, t := range torrents {
		if t.Progress == 1.0 && !isTransitionalState(t.State) {
			completed = append(completed, t)
		}
	}

	return completed, nil
}

// FilesByHash retrieves file list for a specific torrent
func (c *Client) FilesByHash(ctx context.Context, hash string) ([]TorrentFile, error) {
	filesURL := c.baseURL.ResolveReference(&url.URL{
		Path:     "/api/v2/torrents/files",
		RawQuery: fmt.Sprintf("hash=%s", hash),
	})

	req, err := http.NewRequestWithContext(ctx, "GET", filesURL.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create files request: %w", err)
	}

	// Set required headers
	req.Header.Set("Referer", c.baseURL.String())
	req.Header.Set("Origin", c.baseURL.String())

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to perform files request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("get files failed with status: %s", resp.Status)
	}

	var files []TorrentFile
	if err := decodeJSON(resp.Body, &files); err != nil {
		return nil, fmt.Errorf("failed to decode file list: %w", err)
	}

	return files, nil
}

// DeleteTorrent removes a torrent from qBittorrent
func (c *Client) DeleteTorrent(ctx context.Context, hash string, deleteFiles bool) error {
	deleteURL := c.baseURL.ResolveReference(&url.URL{
		Path: "/api/v2/torrents/delete",
	})

	// Prepare form data
	data := fmt.Sprintf("hashes=%s&deleteFiles=%t", hash, deleteFiles)

	req, err := http.NewRequestWithContext(ctx, "POST", deleteURL.String(), strings.NewReader(data))
	if err != nil {
		return fmt.Errorf("failed to create delete request: %w", err)
	}

	// Set required headers
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Referer", c.baseURL.String())
	req.Header.Set("Origin", c.baseURL.String())

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to perform delete request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("delete torrent failed with status: %s", resp.Status)
	}

	return nil
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

// decodeJSON is a helper function to decode JSON response
func decodeJSON(r io.Reader, v interface{}) error {
	decoder := json.NewDecoder(r)
	return decoder.Decode(v)
}