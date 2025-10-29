package plex

import (
	"context"
	"encoding/xml"
	"log"
	"fmt"
	"net/http"
	"net/url"
	"path/filepath"
	"strings"
	"time"

	"qb-sync/internal/config"
)

// Library represents a Plex library section
type Library struct {
	Key       string    `xml:"key,attr"`
	Type      string    `xml:"type,attr"`
	Title     string    `xml:"title,attr"`
	Locations []Location `xml:"Location"`
}

// Location represents a library location path
type Location struct {
	ID   int    `xml:"id,attr"`
	Path string `xml:"path,attr"`
}

// MediaContainer represents the root XML element in Plex API responses
type MediaContainer struct {
	Size     int       `xml:"size,attr"`
	Libraries []Library `xml:"Directory"`
}

// Client represents a Plex Media Server client
type Client struct {
	httpClient *http.Client
	baseURL    *url.URL
	token      string
}

// NewClient creates a new Plex client
func NewClient(cfg *config.PlexConfig) (*Client, error) {
	baseURL, err := url.Parse(cfg.URL)
	if err != nil {
		return nil, fmt.Errorf("invalid Plex URL: %w", err)
	}

	// Ensure URL has the correct port if not specified
	if baseURL.Port() == "" {
		baseURL.Host = baseURL.Host + ":32400"
	}

	httpClient := &http.Client{
		Timeout: 30 * time.Second,
	}

	return &Client{
		httpClient: httpClient,
		baseURL:    baseURL,
		token:      cfg.Token,
	}, nil
}

// GetLibraries retrieves all libraries from the Plex server
func (c *Client) GetLibraries(ctx context.Context) ([]Library, error) {
	librariesURL := c.baseURL.ResolveReference(&url.URL{
		Path: "/library/sections",
	})

	req, err := http.NewRequestWithContext(ctx, "GET", librariesURL.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create libraries request: %w", err)
	}

	// Add Plex token to request
	req.Header.Set("X-Plex-Token", c.token)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to perform libraries request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("get libraries failed with status: %s", resp.Status)
	}

	var container MediaContainer
	if err := xml.NewDecoder(resp.Body).Decode(&container); err != nil {
		return nil, fmt.Errorf("failed to decode libraries response: %w", err)
	}

	return container.Libraries, nil
}

// FindLibraryByPath finds the library that contains the given path
func (c *Client) FindLibraryByPath(ctx context.Context, filePath string) (*Library, string, error) {
	libraries, err := c.GetLibraries(ctx)
	if err != nil {
		return nil, "", fmt.Errorf("failed to get libraries: %w", err)
	}

	// Clean the file path for consistent comparison
	cleanFilePath := filepath.Clean(filePath)

	// Find the library whose location is a prefix of the file path
	for _, library := range libraries {
		for _, location := range library.Locations {
			cleanLocation := filepath.Clean(location.Path)
			
			// Check if the file path starts with the library location
			if strings.HasPrefix(cleanFilePath, cleanLocation+string(filepath.Separator)) ||
			   strings.HasPrefix(cleanFilePath, cleanLocation) {
				// Extract the relative path from the library location
				relativePath := strings.TrimPrefix(cleanFilePath, cleanLocation)
				relativePath = strings.TrimPrefix(relativePath, string(filepath.Separator))
				
				return &library, relativePath, nil
			}
		}
	}

	return nil, "", fmt.Errorf("no library found for path: %s", filePath)
}

// RefreshLibrary triggers a refresh of a specific library
func (c *Client) RefreshLibrary(ctx context.Context, libraryKey string) error {
	refreshURL := c.baseURL.ResolveReference(&url.URL{
		Path: fmt.Sprintf("/library/sections/%s/refresh", libraryKey),
	})
	log.Printf("Refreshing Plex library (ID: %s) with URL: %s", libraryKey, refreshURL.String())

	req, err := http.NewRequestWithContext(ctx, "GET", refreshURL.String(), nil)
	if err != nil {
		return fmt.Errorf("failed to create refresh request: %w", err)
	}

	// Add Plex token to request
	req.Header.Set("X-Plex-Token", c.token)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to perform refresh request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("library refresh failed with status: %s", resp.Status)
	}

	return nil
}

// RefreshLibraryPath triggers a refresh of a specific path within a library
func (c *Client) RefreshLibraryPath(ctx context.Context, libraryKey, path string) error {
	refreshURL := c.baseURL.ResolveReference(&url.URL{
		Path:     fmt.Sprintf("/library/sections/%s/refresh", libraryKey),
		RawQuery: fmt.Sprintf("path=%s", url.QueryEscape(path)),
	})
	log.Printf("Refreshing Plex library path (ID: %s, Path: %s) with URL: %s", libraryKey, path, refreshURL.String())

	req, err := http.NewRequestWithContext(ctx, "GET", refreshURL.String(), nil)
	if err != nil {
		return fmt.Errorf("failed to create path refresh request: %w", err)
	}

	// Add Plex token to request
	req.Header.Set("X-Plex-Token", c.token)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to perform path refresh request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("library path refresh failed with status: %s", resp.Status)
	}

	return nil
}

// RefreshPathForFile finds the appropriate library and refreshes the specific path containing the file
func (c *Client) RefreshPathForFile(ctx context.Context, filePath string) error {
	library, _, err := c.FindLibraryByPath(ctx, filePath)
	if err != nil {
		return fmt.Errorf("failed to find library for file: %w", err)
	}
	log.Printf("Found library '%s' (ID: %s) for file: %s", library.Title, library.Key, filePath)

	// Extract the directory containing the file
	dirPath := filepath.Dir(filePath)
	
	// Refresh the specific directory path containing the file
	return c.RefreshLibraryPath(ctx, library.Key, dirPath)
}