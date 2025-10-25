package build

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// AssetType represents the type of asset
type AssetType int

const (
	AssetTypeCSS AssetType = iota
	AssetTypeJS
	AssetTypeImage
	AssetTypeFont
	AssetTypeOther
)

// AssetCompiler handles compilation of frontend assets
type AssetCompiler struct {
	minify bool
}

// NewAssetCompiler creates a new asset compiler
func NewAssetCompiler() *AssetCompiler {
	return &AssetCompiler{
		minify: false,
	}
}

// SetMinify enables or disables minification
func (ac *AssetCompiler) SetMinify(minify bool) {
	ac.minify = minify
}

// CompileAssets compiles all assets from source to build directory
func (ac *AssetCompiler) CompileAssets(sourceDir, buildDir string) error {
	assetsSourceDir := filepath.Join(sourceDir, "assets")
	assetsBuildDir := filepath.Join(buildDir, "assets")

	// Check if assets directory exists
	if _, err := os.Stat(assetsSourceDir); os.IsNotExist(err) {
		// No assets to compile
		return nil
	}

	// Create build assets directory
	if err := os.MkdirAll(assetsBuildDir, 0755); err != nil {
		return fmt.Errorf("failed to create assets build directory: %w", err)
	}

	// Walk through assets directory
	err := filepath.Walk(assetsSourceDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip directories
		if info.IsDir() {
			return nil
		}

		// Determine asset type
		assetType := ac.getAssetType(path)

		// Calculate relative path
		relPath, err := filepath.Rel(assetsSourceDir, path)
		if err != nil {
			return fmt.Errorf("failed to get relative path: %w", err)
		}

		destPath := filepath.Join(assetsBuildDir, relPath)

		// Compile based on type
		switch assetType {
		case AssetTypeCSS:
			return ac.compileCSS(path, destPath)
		case AssetTypeJS:
			return ac.compileJS(path, destPath)
		default:
			return ac.copyAsset(path, destPath)
		}
	})

	if err != nil {
		return fmt.Errorf("failed to compile assets: %w", err)
	}

	return nil
}

// getAssetType determines the type of an asset based on its extension
func (ac *AssetCompiler) getAssetType(path string) AssetType {
	ext := strings.ToLower(filepath.Ext(path))

	switch ext {
	case ".css":
		return AssetTypeCSS
	case ".js":
		return AssetTypeJS
	case ".png", ".jpg", ".jpeg", ".gif", ".svg", ".webp":
		return AssetTypeImage
	case ".woff", ".woff2", ".ttf", ".otf", ".eot":
		return AssetTypeFont
	default:
		return AssetTypeOther
	}
}

// compileCSS compiles a CSS file
func (ac *AssetCompiler) compileCSS(sourcePath, destPath string) error {
	// Read source CSS
	content, err := os.ReadFile(sourcePath)
	if err != nil {
		return fmt.Errorf("failed to read CSS file: %w", err)
	}

	// Apply minification if enabled
	if ac.minify {
		content = ac.minifyCSS(content)
	}

	// Create destination directory
	if err := os.MkdirAll(filepath.Dir(destPath), 0755); err != nil {
		return fmt.Errorf("failed to create destination directory: %w", err)
	}

	// Write compiled CSS
	if err := os.WriteFile(destPath, content, 0644); err != nil {
		return fmt.Errorf("failed to write CSS file: %w", err)
	}

	return nil
}

// compileJS compiles a JavaScript file
func (ac *AssetCompiler) compileJS(sourcePath, destPath string) error {
	// Read source JS
	content, err := os.ReadFile(sourcePath)
	if err != nil {
		return fmt.Errorf("failed to read JS file: %w", err)
	}

	// Apply minification if enabled
	if ac.minify {
		content = ac.minifyJS(content)
	}

	// Create destination directory
	if err := os.MkdirAll(filepath.Dir(destPath), 0755); err != nil {
		return fmt.Errorf("failed to create destination directory: %w", err)
	}

	// Write compiled JS
	if err := os.WriteFile(destPath, content, 0644); err != nil {
		return fmt.Errorf("failed to write JS file: %w", err)
	}

	return nil
}

// copyAsset copies a static asset
func (ac *AssetCompiler) copyAsset(sourcePath, destPath string) error {
	// Open source file
	source, err := os.Open(sourcePath)
	if err != nil {
		return fmt.Errorf("failed to open source file: %w", err)
	}
	defer source.Close()

	// Create destination directory
	if err := os.MkdirAll(filepath.Dir(destPath), 0755); err != nil {
		return fmt.Errorf("failed to create destination directory: %w", err)
	}

	// Create destination file
	dest, err := os.Create(destPath)
	if err != nil {
		return fmt.Errorf("failed to create destination file: %w", err)
	}
	defer dest.Close()

	// Copy
	if _, err := io.Copy(dest, source); err != nil {
		return fmt.Errorf("failed to copy file: %w", err)
	}

	return nil
}

// minifyCSS performs basic CSS minification
func (ac *AssetCompiler) minifyCSS(content []byte) []byte {
	// Basic minification:
	// - Remove comments
	// - Remove unnecessary whitespace
	// - Remove newlines

	s := string(content)

	// Remove CSS comments
	s = removeComments(s, "/*", "*/")

	// Remove extra whitespace
	s = strings.Join(strings.Fields(s), " ")

	// Remove spaces around specific characters
	replacements := []struct {
		old string
		new string
	}{
		{" {", "{"},
		{"{ ", "{"},
		{" }", "}"},
		{"} ", "}"},
		{" :", ":"},
		{": ", ":"},
		{" ;", ";"},
		{"; ", ";"},
		{" ,", ","},
		{", ", ","},
	}

	for _, r := range replacements {
		s = strings.ReplaceAll(s, r.old, r.new)
	}

	return []byte(s)
}

// minifyJS performs basic JavaScript minification
func (ac *AssetCompiler) minifyJS(content []byte) []byte {
	// Basic minification:
	// - Remove single-line comments
	// - Remove multi-line comments
	// - Remove unnecessary whitespace

	s := string(content)

	// Remove single-line comments
	lines := strings.Split(s, "\n")
	filtered := make([]string, 0, len(lines))
	for _, line := range lines {
		// Check if line contains //
		if idx := strings.Index(line, "//"); idx != -1 {
			// Keep everything before //
			line = line[:idx]
		}
		if strings.TrimSpace(line) != "" {
			filtered = append(filtered, line)
		}
	}
	s = strings.Join(filtered, "\n")

	// Remove multi-line comments
	s = removeComments(s, "/*", "*/")

	// Remove extra whitespace (but preserve newlines for basic readability)
	lines = strings.Split(s, "\n")
	for i, line := range lines {
		lines[i] = strings.TrimSpace(line)
	}
	s = strings.Join(lines, "\n")

	return []byte(s)
}

// removeComments removes comments from text
func removeComments(text, startDelim, endDelim string) string {
	for {
		start := strings.Index(text, startDelim)
		if start == -1 {
			break
		}

		end := strings.Index(text[start:], endDelim)
		if end == -1 {
			break
		}

		end += start + len(endDelim)
		text = text[:start] + text[end:]
	}

	return text
}
