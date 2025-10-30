package files

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"qb-sync/internal/config"
	"qb-sync/internal/qbit"
)

// FileOperation represents the result of a file operation
type FileOperation struct {
	Source      string
	Destination string
	Size        int64
	Success     bool
	Error       error
}

// LinkOrCopy performs hardlink or copy operation based on configuration
func LinkOrCopy(cfg *config.MonitorConfig, torrent *qbit.Torrent, file *qbit.TorrentFile) (*FileOperation, error) {
	// Skip incomplete files
	if strings.HasSuffix(file.Name, ".!qB") {
		return nil, fmt.Errorf("skipping incomplete file: %s", file.Name)
	}

	// Build source and destination paths
	// Use content_path as the base directory for files, not save_path
	// content_path already includes the full path to where the files are located
	sourcePath := filepath.Join(torrent.ContentPath, file.Name)
	destPath, err := BuildDestPath(cfg, torrent, file)
	if err != nil {
		return nil, fmt.Errorf("failed to build destination path: %w", err)
	}

	// Check if destination already exists with same size (idempotency)
	if info, err := os.Stat(destPath); err == nil && info.Size() == file.Size {
		return &FileOperation{
			Source:      sourcePath,
			Destination: destPath,
			Size:        file.Size,
			Success:     true,
		}, nil
	}

	// Create destination directory
	if err := os.MkdirAll(filepath.Dir(destPath), 0755); err != nil {
		return nil, fmt.Errorf("failed to create destination directory: %w", err)
	}

	// Perform operation based on configuration
	var opErr error
	switch cfg.Operation {
	case "hardlink":
		opErr = createHardlink(sourcePath, destPath, cfg.CrossDeviceFallback, file.Size)
	case "copy":
		opErr = copyFile(sourcePath, destPath, file.Size)
	default:
		return nil, fmt.Errorf("unsupported operation: %s", cfg.Operation)
	}

	success := opErr == nil
	return &FileOperation{
		Source:      sourcePath,
		Destination: destPath,
		Size:        file.Size,
		Success:     success,
		Error:       opErr,
	}, opErr
}

// BuildDestPath constructs the destination path based on configuration
func BuildDestPath(cfg *config.MonitorConfig, torrent *qbit.Torrent, file *qbit.TorrentFile) (string, error) {
	if cfg.PreserveSubfolder {
		// Preserve subfolder structure: dest_path/torrent_name/file_path
		return filepath.Join(cfg.DestPath, torrent.Name, file.Name), nil
	}
	// Flatten structure: dest_path/file_path
	return filepath.Join(cfg.DestPath, file.Name), nil
}

// createHardlink attempts to create a hardlink, with fallback for cross-device errors
func createHardlink(src, dst, fallback string, expectedSize int64) error {
	err := os.Link(src, dst)
	if err == nil {
		return nil
	}

	// Check if it's a cross-device error
	if isCrossDeviceError(err) {
		switch fallback {
		case "copy":
			return copyFile(src, dst, expectedSize)
		case "error":
			return fmt.Errorf("cross-device hardlink not allowed: %w", err)
		default:
			return fmt.Errorf("invalid cross-device fallback option: %s", fallback)
		}
	}

	return fmt.Errorf("failed to create hardlink: %w", err)
}

// copyFile copies a file with preservation of metadata
func copyFile(src, dst string, expectedSize int64) error {
	// Open source file
	srcFile, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("failed to open source file: %w", err)
	}
	defer srcFile.Close()

	// Get source file info for metadata
	srcInfo, err := srcFile.Stat()
	if err != nil {
		return fmt.Errorf("failed to stat source file: %w", err)
	}

	// Create destination file
	dstFile, err := os.OpenFile(dst, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, srcInfo.Mode())
	if err != nil {
		return fmt.Errorf("failed to create destination file: %w", err)
	}
	defer dstFile.Close()

	// Copy file content
	copied, err := io.Copy(dstFile, srcFile)
	if err != nil {
		return fmt.Errorf("failed to copy file content: %w", err)
	}

	// Verify size
	if copied != expectedSize {
		return fmt.Errorf("size mismatch: expected %d, got %d", expectedSize, copied)
	}

	// Sync to ensure data is written to disk
	if err := dstFile.Sync(); err != nil {
		return fmt.Errorf("failed to sync destination file: %w", err)
	}

	// Preserve modification time
	if err := os.Chtimes(dst, time.Now(), srcInfo.ModTime()); err != nil {
		return fmt.Errorf("failed to set modification time: %w", err)
	}

	return nil
}

// isCrossDeviceError checks if the error is a cross-device link error
func isCrossDeviceError(err error) bool {
	// On Unix systems, cross-device link errors have errno EXDEV (18)
	// On Windows, they might have different error codes
	if err == nil {
		return false
	}

	errStr := err.Error()
	return strings.Contains(errStr, "cross-device") ||
		strings.Contains(errStr, "invalid cross-device link") ||
		strings.Contains(errStr, "EXDEV")
}

// VerifyFileIntegrity checks if a file exists and has the expected size
func VerifyFileIntegrity(path string, expectedSize int64) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return info.Size() == expectedSize
}

// CleanupDestination removes a file from the destination
func CleanupDestination(path string) error {
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to cleanup destination file %s: %w", path, err)
	}
	return nil
}