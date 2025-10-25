package watch

import (
	"testing"
)

func TestIsAssetFile(t *testing.T) {
	tests := []struct {
		path     string
		expected bool
	}{
		// Asset extensions
		{"style.css", true},
		{"script.js", true},
		{"image.png", true},
		{"icon.svg", true},
		{"font.woff2", true},
		{"video.mp4", true},
		{"audio.mp3", true},

		// Non-asset files
		{"main.cdt", false},
		{"server.go", false},
		{"readme.txt", false},

		// Directories
		{"public/style.css", true},
		{"assets/image.png", true},
		{"src/main.cdt", false},
	}

	for _, tt := range tests {
		result := IsAssetFile(tt.path)
		if result != tt.expected {
			t.Errorf("IsAssetFile(%q) = %v, expected %v", tt.path, result, tt.expected)
		}
	}
}

func TestAnalyzeImpact_UI(t *testing.T) {
	files := []string{"style.css", "script.js"}
	impact := AnalyzeImpact(files)

	if impact.Scope != ScopeUI {
		t.Errorf("Expected ScopeUI, got %v", impact.Scope)
	}

	if impact.RequiresRestart {
		t.Error("Expected RequiresRestart to be false for UI changes")
	}

	if impact.RequiresRebuild {
		t.Error("Expected RequiresRebuild to be false for UI changes")
	}
}

func TestAnalyzeImpact_Backend(t *testing.T) {
	files := []string{"post.cdt", "user.cdt"}
	impact := AnalyzeImpact(files)

	if impact.Scope != ScopeBackend {
		t.Errorf("Expected ScopeBackend, got %v", impact.Scope)
	}

	if !impact.RequiresRestart {
		t.Error("Expected RequiresRestart to be true for backend changes")
	}

	if !impact.RequiresRebuild {
		t.Error("Expected RequiresRebuild to be true for backend changes")
	}

	if len(impact.AffectedResources) != 2 {
		t.Errorf("Expected 2 affected resources, got %d", len(impact.AffectedResources))
	}
}

func TestAnalyzeImpact_Config(t *testing.T) {
	files := []string{"config/database.toml"}
	impact := AnalyzeImpact(files)

	if impact.Scope != ScopeConfig {
		t.Errorf("Expected ScopeConfig, got %v", impact.Scope)
	}

	if !impact.RequiresRestart {
		t.Error("Expected RequiresRestart to be true for config changes")
	}
}

func TestAnalyzeImpact_Mixed(t *testing.T) {
	// When mixing scopes, highest scope should win
	files := []string{"style.css", "post.cdt"}
	impact := AnalyzeImpact(files)

	if impact.Scope != ScopeBackend {
		t.Errorf("Expected ScopeBackend (highest scope), got %v", impact.Scope)
	}

	if !impact.RequiresRestart {
		t.Error("Expected RequiresRestart to be true")
	}
}

func TestAnalyzeImpact_ConfigOverridesBackend(t *testing.T) {
	// Config changes should override backend
	files := []string{"post.cdt", "config/app.yaml"}
	impact := AnalyzeImpact(files)

	if impact.Scope != ScopeConfig {
		t.Errorf("Expected ScopeConfig (highest), got %v", impact.Scope)
	}
}

func TestAnalyzeImpact_GoFiles(t *testing.T) {
	files := []string{"build/generated/main.go"}
	impact := AnalyzeImpact(files)

	if impact.Scope != ScopeBackend {
		t.Errorf("Expected ScopeBackend for Go files, got %v", impact.Scope)
	}

	if !impact.RequiresRestart {
		t.Error("Expected RequiresRestart to be true for Go files")
	}
}

func TestAnalyzeImpact_HTMLFiles(t *testing.T) {
	files := []string{"ui/index.html", "ui/about.html"}
	impact := AnalyzeImpact(files)

	if impact.Scope != ScopeUI {
		t.Errorf("Expected ScopeUI for HTML files, got %v", impact.Scope)
	}
}

func TestAssetWatcher_HandleAssetChange_CSS(t *testing.T) {
	rs := NewReloadServer()
	defer rs.Close()

	aw := NewAssetWatcher(rs)

	files := []string{"style.css", "theme.scss"}
	err := aw.HandleAssetChange(files)

	if err != nil {
		t.Errorf("HandleAssetChange returned error: %v", err)
	}
}

func TestAssetWatcher_HandleAssetChange_JS(t *testing.T) {
	rs := NewReloadServer()
	defer rs.Close()

	aw := NewAssetWatcher(rs)

	files := []string{"app.js", "utils.ts"}
	err := aw.HandleAssetChange(files)

	if err != nil {
		t.Errorf("HandleAssetChange returned error: %v", err)
	}
}

func TestAssetWatcher_HandleAssetChange_Images(t *testing.T) {
	rs := NewReloadServer()
	defer rs.Close()

	aw := NewAssetWatcher(rs)

	files := []string{"logo.png", "icon.svg"}
	err := aw.HandleAssetChange(files)

	if err != nil {
		t.Errorf("HandleAssetChange returned error: %v", err)
	}
}

func TestAssetWatcher_HandleAssetChange_Mixed(t *testing.T) {
	rs := NewReloadServer()
	defer rs.Close()

	aw := NewAssetWatcher(rs)

	// Mixed file types - CSS should be handled first
	files := []string{"style.css", "script.js", "image.png"}
	err := aw.HandleAssetChange(files)

	if err != nil {
		t.Errorf("HandleAssetChange returned error: %v", err)
	}
}

func BenchmarkAnalyzeImpact(b *testing.B) {
	files := []string{
		"style.css",
		"post.cdt",
		"user.cdt",
		"script.js",
		"config.toml",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		AnalyzeImpact(files)
	}
}

func BenchmarkIsAssetFile(b *testing.B) {
	paths := []string{
		"style.css",
		"script.js",
		"main.cdt",
		"image.png",
		"server.go",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, path := range paths {
			IsAssetFile(path)
		}
	}
}
