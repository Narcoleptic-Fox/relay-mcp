package utils

import (
	"encoding/base64"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestProcessImages_DataURI(t *testing.T) {
	// Create a small valid base64 image (1x1 red PNG)
	pngData := []byte{
		0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A, // PNG signature
		0x00, 0x00, 0x00, 0x0D, 0x49, 0x48, 0x44, 0x52, // IHDR chunk
		0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x01,
		0x08, 0x02, 0x00, 0x00, 0x00, 0x90, 0x77, 0x53,
		0xDE, 0x00, 0x00, 0x00, 0x0C, 0x49, 0x44, 0x41, // IDAT chunk
		0x54, 0x08, 0xD7, 0x63, 0xF8, 0xFF, 0xFF, 0x3F,
		0x00, 0x05, 0xFE, 0x02, 0xFE, 0xDC, 0xCC, 0x59,
		0xE7, 0x00, 0x00, 0x00, 0x00, 0x49, 0x45, 0x4E, // IEND chunk
		0x44, 0xAE, 0x42, 0x60, 0x82,
	}
	b64 := base64.StdEncoding.EncodeToString(pngData)

	dataURI := "data:image/png;base64," + b64

	results := ProcessImages([]string{dataURI})

	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}

	if results[0].MimeType != "image/png" {
		t.Errorf("expected mime type 'image/png', got '%s'", results[0].MimeType)
	}

	if results[0].Base64 != b64 {
		t.Error("base64 data doesn't match")
	}
}

func TestProcessImages_RawBase64(t *testing.T) {
	// Create a long enough base64 string that looks valid
	data := make([]byte, 200)
	for i := range data {
		data[i] = byte(i % 256)
	}
	b64 := base64.StdEncoding.EncodeToString(data)

	results := ProcessImages([]string{b64})

	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}

	// Raw base64 defaults to JPEG
	if results[0].MimeType != "image/jpeg" {
		t.Errorf("expected default mime type 'image/jpeg', got '%s'", results[0].MimeType)
	}
}

func TestProcessImages_FilePath(t *testing.T) {
	// Create a temp file with fake image data
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.png")

	// Write some fake PNG data
	err := os.WriteFile(testFile, []byte("fake png data for testing"), 0644)
	if err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	results := ProcessImages([]string{testFile})

	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}

	if results[0].MimeType != "image/png" {
		t.Errorf("expected mime type 'image/png', got '%s'", results[0].MimeType)
	}

	// Verify it's valid base64
	_, err = base64.StdEncoding.DecodeString(results[0].Base64)
	if err != nil {
		t.Errorf("result is not valid base64: %v", err)
	}
}

func TestProcessImages_RelativePathRejected(t *testing.T) {
	results := ProcessImages([]string{"relative/path/image.png"})

	if len(results) != 0 {
		t.Error("expected relative path to be rejected")
	}
}

func TestProcessImages_NonexistentFile(t *testing.T) {
	results := ProcessImages([]string{"/nonexistent/path/image.png"})

	if len(results) != 0 {
		t.Error("expected nonexistent file to be rejected")
	}
}

func TestProcessImages_InvalidInput(t *testing.T) {
	// Short string that's not base64 and not a file path
	results := ProcessImages([]string{"invalid"})

	if len(results) != 0 {
		t.Error("expected invalid input to be rejected")
	}
}

func TestProcessImages_MultipleInputs(t *testing.T) {
	// Create valid data URI
	data := make([]byte, 200)
	for i := range data {
		data[i] = byte(i % 256)
	}
	b64 := base64.StdEncoding.EncodeToString(data)
	dataURI := "data:image/jpeg;base64," + b64

	// Create temp file
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.jpg")
	os.WriteFile(testFile, data, 0644)

	results := ProcessImages([]string{
		dataURI,
		testFile,
		"invalid",   // This should be skipped
		b64,         // Raw base64
	})

	if len(results) != 3 {
		t.Errorf("expected 3 valid results, got %d", len(results))
	}
}

func TestGetMimeType(t *testing.T) {
	tests := []struct {
		path     string
		expected string
	}{
		{"/path/to/image.jpg", "image/jpeg"},
		{"/path/to/image.jpeg", "image/jpeg"},
		{"/path/to/image.png", "image/png"},
		{"/path/to/image.gif", "image/gif"},
		{"/path/to/image.webp", "image/webp"},
		{"/path/to/image.bmp", "image/bmp"},
		{"/path/to/image.tiff", "image/tiff"},
		{"/path/to/image.unknown", "image/jpeg"}, // Default
		{"/path/to/image.PNG", "image/png"},      // Case insensitive
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			result := getMimeType(tt.path)
			if result != tt.expected {
				t.Errorf("getMimeType(%s) = %s, want %s", tt.path, result, tt.expected)
			}
		})
	}
}

func TestLooksLikeFilePath(t *testing.T) {
	tests := []struct {
		input    string
		expected bool
	}{
		{"/absolute/path/image.png", true},
		{"C:\\Windows\\path\\image.jpg", true},
		{"relative/path/image.gif", true},
		{"image.jpg", true},  // Has extension
		{"image.png", true},
		{"dGVzdCBkYXRh", false},   // Base64-like
		{"short", false},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := looksLikeFilePath(tt.input)
			if result != tt.expected {
				t.Errorf("looksLikeFilePath(%s) = %v, want %v", tt.input, result, tt.expected)
			}
		})
	}
}

func TestParseDataURI(t *testing.T) {
	// Valid PNG data URI
	data := make([]byte, 100)
	b64 := base64.StdEncoding.EncodeToString(data)

	result, err := parseDataURI("data:image/png;base64," + b64)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.MimeType != "image/png" {
		t.Errorf("expected mime type 'image/png', got '%s'", result.MimeType)
	}

	// Invalid: missing comma
	_, err = parseDataURI("data:image/png;base64" + b64)
	if err == nil {
		t.Error("expected error for missing comma")
	}

	// Invalid: not a data URI
	_, err = parseDataURI("not a data uri")
	if err == nil {
		t.Error("expected error for non-data URI")
	}

	// Invalid: not base64 encoded
	_, err = parseDataURI("data:image/png,not-base64")
	if err == nil || !strings.Contains(err.Error(), "not base64") {
		t.Errorf("expected error about not being base64, got: %v", err)
	}
}
