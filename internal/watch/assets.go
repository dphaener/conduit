package watch

import (
	"log"
	"path/filepath"
	"strings"
)

// AssetWatcher manages watching and reloading of static assets
type AssetWatcher struct {
	reloadServer *ReloadServer
}

// NewAssetWatcher creates a new asset watcher
func NewAssetWatcher(reloadServer *ReloadServer) *AssetWatcher {
	return &AssetWatcher{
		reloadServer: reloadServer,
	}
}

// HandleAssetChange processes asset file changes
func (aw *AssetWatcher) HandleAssetChange(files []string) error {
	// Categorize changes
	cssFiles := make([]string, 0)
	jsFiles := make([]string, 0)
	imageFiles := make([]string, 0)
	otherFiles := make([]string, 0)

	for _, file := range files {
		ext := strings.ToLower(filepath.Ext(file))
		switch ext {
		case ".css", ".scss", ".sass", ".less":
			cssFiles = append(cssFiles, file)
		case ".js", ".jsx", ".ts", ".tsx":
			jsFiles = append(jsFiles, file)
		case ".png", ".jpg", ".jpeg", ".gif", ".svg", ".webp", ".ico":
			imageFiles = append(imageFiles, file)
		default:
			otherFiles = append(otherFiles, file)
		}
	}

	// Handle CSS changes (hot reload without full page refresh)
	if len(cssFiles) > 0 {
		log.Printf("[Assets] CSS changed: %v", cssFiles)
		aw.reloadServer.NotifyReload("ui")
		return nil
	}

	// Handle JavaScript changes (requires full reload)
	if len(jsFiles) > 0 {
		log.Printf("[Assets] JS changed: %v", jsFiles)
		aw.reloadServer.NotifyReload("ui")
		return nil
	}

	// Handle image changes
	if len(imageFiles) > 0 {
		log.Printf("[Assets] Images changed: %v", imageFiles)
		aw.reloadServer.NotifyReload("ui")
		return nil
	}

	// Handle other files
	if len(otherFiles) > 0 {
		log.Printf("[Assets] Other files changed: %v", otherFiles)
		aw.reloadServer.NotifyReload("ui")
	}

	return nil
}

// IsAssetFile checks if a file is a static asset
func IsAssetFile(path string) bool {
	// Check if file is in public/assets directories
	if strings.Contains(path, "public/") || strings.Contains(path, "assets/") {
		return true
	}

	// Check file extension
	ext := strings.ToLower(filepath.Ext(path))
	assetExtensions := []string{
		".css", ".scss", ".sass", ".less",
		".js", ".jsx", ".ts", ".tsx",
		".png", ".jpg", ".jpeg", ".gif", ".svg", ".webp", ".ico",
		".woff", ".woff2", ".ttf", ".eot",
		".mp4", ".webm", ".ogg",
		".mp3", ".wav",
	}

	for _, assetExt := range assetExtensions {
		if ext == assetExt {
			return true
		}
	}

	return false
}

// ChangeImpact represents the impact of file changes
type ChangeImpact struct {
	Scope             ImpactScope
	RequiresRestart   bool
	RequiresRebuild   bool
	AffectedResources []string
	AffectedRoutes    []string
}

// ImpactScope defines the scope of changes
type ImpactScope int

const (
	ScopeUI      ImpactScope = iota // Browser refresh only
	ScopeBackend                     // Server restart needed
	ScopeConfig                      // Full restart needed
)

// AnalyzeImpact analyzes the impact of file changes
func AnalyzeImpact(files []string) *ChangeImpact {
	impact := &ChangeImpact{
		AffectedResources: make([]string, 0),
		AffectedRoutes:    make([]string, 0),
	}

	for _, file := range files {
		// Check file location and type
		switch {
		case IsAssetFile(file):
			// Asset change - browser refresh only
			if impact.Scope < ScopeUI {
				impact.Scope = ScopeUI
			}

		case strings.HasPrefix(file, "ui/") || strings.HasSuffix(file, ".html"):
			// UI template change - browser refresh
			if impact.Scope < ScopeUI {
				impact.Scope = ScopeUI
			}

		case strings.HasSuffix(file, ".cdt"):
			// Conduit source file - requires rebuild and restart
			if impact.Scope < ScopeBackend {
				impact.Scope = ScopeBackend
			}
			impact.RequiresRebuild = true
			impact.RequiresRestart = true
			impact.AffectedResources = append(impact.AffectedResources, file)

		case strings.Contains(file, "config/") || strings.HasSuffix(file, ".toml") || strings.HasSuffix(file, ".yaml"):
			// Config change - full restart
			impact.Scope = ScopeConfig
			impact.RequiresRestart = true

		case strings.HasSuffix(file, ".go"):
			// Generated Go file - requires restart
			if impact.Scope < ScopeBackend {
				impact.Scope = ScopeBackend
			}
			impact.RequiresRestart = true
		}
	}

	return impact
}
