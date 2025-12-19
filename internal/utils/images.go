package utils

import (
	"encoding/base64"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
)

// ImageData represents processed image data ready for API consumption
type ImageData struct {
	Base64   string
	MimeType string
}

// ProcessImages converts image inputs (paths or base64) to ImageData
// Accepts:
// - Absolute file paths (will be read and base64 encoded)
// - Data URIs (data:image/jpeg;base64,...)
// - Raw base64 strings
func ProcessImages(images []string) []ImageData {
	var result []ImageData

	for _, img := range images {
		data, err := processImage(img)
		if err != nil {
			slog.Warn("failed to process image", "image", truncateForLog(img, 50), "error", err)
			continue
		}
		result = append(result, data)
	}

	return result
}

func processImage(img string) (ImageData, error) {
	// Check if it's a data URI (data:image/jpeg;base64,...)
	if strings.HasPrefix(img, "data:") {
		return parseDataURI(img)
	}

	// Check if it looks like an actual file path (starts with path indicators)
	if looksLikeFilePath(img) {
		return readImageFile(img)
	}

	// Try to decode as raw base64 - if it's long enough and decodes, use it
	// Base64 characters are: A-Za-z0-9+/=
	if len(img) > 100 && looksLikeBase64(img) {
		if _, err := base64.StdEncoding.DecodeString(img); err == nil {
			return ImageData{
				Base64:   img,
				MimeType: "image/jpeg", // Default to JPEG for raw base64
			}, nil
		}
	}

	return ImageData{}, fmt.Errorf("unrecognized image format: not a file path, data URI, or valid base64")
}

func parseDataURI(uri string) (ImageData, error) {
	// Format: data:image/jpeg;base64,/9j/4AAQ...
	if !strings.HasPrefix(uri, "data:") {
		return ImageData{}, fmt.Errorf("not a data URI")
	}

	// Find the comma separating metadata from data
	commaIdx := strings.Index(uri, ",")
	if commaIdx == -1 {
		return ImageData{}, fmt.Errorf("invalid data URI: missing comma")
	}

	metadata := uri[5:commaIdx] // Skip "data:"
	data := uri[commaIdx+1:]

	// Parse MIME type from metadata
	mimeType := "image/jpeg" // Default
	if strings.Contains(metadata, ";base64") {
		mimeType = strings.TrimSuffix(metadata, ";base64")
	} else {
		// Not base64 encoded
		return ImageData{}, fmt.Errorf("data URI is not base64 encoded")
	}

	// Validate the base64 data
	if _, err := base64.StdEncoding.DecodeString(data); err != nil {
		return ImageData{}, fmt.Errorf("invalid base64 in data URI: %w", err)
	}

	return ImageData{
		Base64:   data,
		MimeType: mimeType,
	}, nil
}

func looksLikeFilePath(s string) bool {
	// Check for absolute path indicators
	// Unix absolute paths start with /
	if strings.HasPrefix(s, "/") {
		return true
	}
	// Windows absolute paths like C:\ or D:\
	if len(s) >= 3 && s[1] == ':' && (s[2] == '\\' || s[2] == '/') {
		return true
	}
	// Relative paths starting with ./ or ../
	if strings.HasPrefix(s, "./") || strings.HasPrefix(s, "../") || strings.HasPrefix(s, ".\\") || strings.HasPrefix(s, "..\\") {
		return true
	}

	// Check for common image extensions (e.g., "image.png" without path)
	ext := strings.ToLower(filepath.Ext(s))
	switch ext {
	case ".jpg", ".jpeg", ".png", ".gif", ".webp", ".bmp", ".tiff":
		return true
	}

	return false
}

func looksLikeBase64(s string) bool {
	// Base64 only contains: A-Za-z0-9+/=
	// Quick validation without full decode
	for _, c := range s {
		if !((c >= 'A' && c <= 'Z') || (c >= 'a' && c <= 'z') || (c >= '0' && c <= '9') || c == '+' || c == '/' || c == '=') {
			return false
		}
	}
	return true
}

func readImageFile(path string) (ImageData, error) {
	// Ensure absolute path
	if !filepath.IsAbs(path) {
		return ImageData{}, fmt.Errorf("image path must be absolute: %s", path)
	}

	// Check file exists and get info
	info, err := os.Stat(path)
	if err != nil {
		return ImageData{}, fmt.Errorf("cannot access image file: %w", err)
	}

	// Check file size (limit to 20MB for safety)
	const maxSize = 20 * 1024 * 1024
	if info.Size() > maxSize {
		return ImageData{}, fmt.Errorf("image file too large: %d bytes (max %d)", info.Size(), maxSize)
	}

	// Read the file
	data, err := os.ReadFile(path)
	if err != nil {
		return ImageData{}, fmt.Errorf("reading image file: %w", err)
	}

	// Determine MIME type from extension
	mimeType := getMimeType(path)

	// Encode to base64
	encoded := base64.StdEncoding.EncodeToString(data)

	return ImageData{
		Base64:   encoded,
		MimeType: mimeType,
	}, nil
}

func getMimeType(path string) string {
	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".png":
		return "image/png"
	case ".gif":
		return "image/gif"
	case ".webp":
		return "image/webp"
	case ".bmp":
		return "image/bmp"
	case ".tiff", ".tif":
		return "image/tiff"
	default:
		return "image/jpeg" // Default to JPEG
	}
}

func truncateForLog(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
