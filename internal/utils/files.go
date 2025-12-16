package utils

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// FileContent holds file path and content
type FileContent struct {
	Path    string
	Content string
}

// ReadFiles reads multiple files
func ReadFiles(paths []string, workDir string) ([]FileContent, error) {
	var results []FileContent

	for _, p := range paths {
		// Validate path
		if !filepath.IsAbs(p) {
			p = filepath.Join(workDir, p)
		}

		if err := validatePath(p, workDir); err != nil {
			continue // Skip invalid paths
		}

		content, err := os.ReadFile(p)
		if err != nil {
			continue // Skip unreadable files
		}

		results = append(results, FileContent{
			Path:    p,
			Content: string(content),
		})
	}

	return results, nil
}

// validatePath ensures the path is safe to read
func validatePath(path, workDir string) error {
	// Clean and resolve the path
	cleanPath := filepath.Clean(path)

	// Ensure it's absolute
	if !filepath.IsAbs(cleanPath) {
		return fmt.Errorf("path must be absolute")
	}

	// Check for path traversal
	if strings.Contains(cleanPath, "..") {
		return fmt.Errorf("path traversal not allowed")
	}

	// Check if path exists
	info, err := os.Stat(cleanPath)
	if err != nil {
		return err
	}

	// Don't allow directories
	if info.IsDir() {
		return fmt.Errorf("path is a directory")
	}

	// Check file size (limit to 1MB)
	if info.Size() > 1024*1024 {
		return fmt.Errorf("file too large")
	}

	return nil
}

// IsBinaryFile checks if a file appears to be binary
func IsBinaryFile(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	binaryExts := map[string]bool{
		".exe": true, ".dll": true, ".so": true, ".dylib": true,
		".zip": true, ".tar": true, ".gz": true, ".7z": true,
		".png": true, ".jpg": true, ".jpeg": true, ".gif": true,
		".pdf": true, ".doc": true, ".docx": true,
		".bin": true, ".dat": true,
	}
	return binaryExts[ext]
}

// IsCodeFile checks if a file is source code
func IsCodeFile(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	codeExts := map[string]bool{
		".go": true, ".py": true, ".js": true, ".ts": true,
		".jsx": true, ".tsx": true, ".java": true, ".c": true,
		".cpp": true, ".h": true, ".hpp": true, ".rs": true,
		".rb": true, ".php": true, ".swift": true, ".kt": true,
		".cs": true, ".fs": true, ".scala": true, ".clj": true,
		".ex": true, ".exs": true, ".erl": true, ".hs": true,
		".ml": true, ".vue": true, ".svelte": true,
	}
	return codeExts[ext]
}
