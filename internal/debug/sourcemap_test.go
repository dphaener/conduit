package debug

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewSourceMapRegistry(t *testing.T) {
	registry := NewSourceMapRegistry()
	assert.NotNil(t, registry)
	assert.NotNil(t, registry.maps)
	assert.Empty(t, registry.maps)
}

func TestNewSourceMap(t *testing.T) {
	sm := NewSourceMap("/path/to/source.cdt", "/path/to/generated.go")
	assert.Equal(t, "/path/to/source.cdt", sm.SourceFile)
	assert.Equal(t, "/path/to/generated.go", sm.GeneratedFile)
	assert.NotNil(t, sm.Mappings)
	assert.Empty(t, sm.Mappings)
}

func TestSourceMap_AddMapping(t *testing.T) {
	sm := NewSourceMap("/path/to/source.cdt", "/path/to/generated.go")

	sm.AddMapping(10, 0, 15, 0)
	sm.AddMapping(20, 5, 30, 10)

	assert.Len(t, sm.Mappings, 2)
	assert.Equal(t, 10, sm.Mappings[0].SourceLine)
	assert.Equal(t, 0, sm.Mappings[0].SourceColumn)
	assert.Equal(t, 15, sm.Mappings[0].GeneratedLine)
	assert.Equal(t, 0, sm.Mappings[0].GeneratedColumn)

	assert.Equal(t, 20, sm.Mappings[1].SourceLine)
	assert.Equal(t, 5, sm.Mappings[1].SourceColumn)
	assert.Equal(t, 30, sm.Mappings[1].GeneratedLine)
	assert.Equal(t, 10, sm.Mappings[1].GeneratedColumn)
}

func TestSourceMapRegistry_Register(t *testing.T) {
	registry := NewSourceMapRegistry()
	sm := NewSourceMap("/path/to/source.cdt", "/path/to/generated.go")
	sm.AddMapping(10, 0, 15, 0)

	registry.Register(sm)

	retrieved, ok := registry.GetSourceMap("/path/to/source.cdt")
	assert.True(t, ok)
	assert.Equal(t, sm, retrieved)
}

func TestSourceMapRegistry_GetSourceMap(t *testing.T) {
	registry := NewSourceMapRegistry()
	sm := NewSourceMap("/path/to/source.cdt", "/path/to/generated.go")
	registry.Register(sm)

	// Test existing source map
	retrieved, ok := registry.GetSourceMap("/path/to/source.cdt")
	assert.True(t, ok)
	assert.NotNil(t, retrieved)

	// Test non-existing source map
	_, ok = registry.GetSourceMap("/path/to/nonexistent.cdt")
	assert.False(t, ok)
}

func TestSourceMapRegistry_TranslateBreakpoint(t *testing.T) {
	registry := NewSourceMapRegistry()
	sm := NewSourceMap("/path/to/source.cdt", "/path/to/generated.go")
	sm.AddMapping(10, 0, 15, 0)
	sm.AddMapping(20, 0, 30, 0)
	sm.AddMapping(30, 0, 45, 0)
	registry.Register(sm)

	// Test exact match
	bp, err := registry.TranslateBreakpoint("/path/to/source.cdt", 10)
	require.NoError(t, err)
	assert.Equal(t, "/path/to/source.cdt", bp.SourceFile)
	assert.Equal(t, 10, bp.SourceLine)
	assert.Equal(t, "/path/to/generated.go", bp.GeneratedFile)
	assert.Equal(t, 15, bp.GeneratedLine)
	assert.False(t, bp.Verified) // Not yet verified by debugger

	// Test closest match (line 11 should map to line 10, distance 1)
	bp, err = registry.TranslateBreakpoint("/path/to/source.cdt", 11)
	require.NoError(t, err)
	// Closest is line 10 (distance 1 vs distance 9 to line 20)
	assert.Equal(t, 10, bp.SourceLine)
	assert.Equal(t, 15, bp.GeneratedLine)

	// Test another line
	bp, err = registry.TranslateBreakpoint("/path/to/source.cdt", 20)
	require.NoError(t, err)
	assert.Equal(t, 20, bp.SourceLine)
	assert.Equal(t, 30, bp.GeneratedLine)
}

func TestSourceMapRegistry_TranslateBreakpoint_NoSourceMap(t *testing.T) {
	registry := NewSourceMapRegistry()

	_, err := registry.TranslateBreakpoint("/path/to/nonexistent.cdt", 10)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no source map")
}

func TestSourceMapRegistry_TranslateBreakpoint_NoMappings(t *testing.T) {
	registry := NewSourceMapRegistry()
	sm := NewSourceMap("/path/to/source.cdt", "/path/to/generated.go")
	// Don't add any mappings
	registry.Register(sm)

	_, err := registry.TranslateBreakpoint("/path/to/source.cdt", 10)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no mapping found")
}

func TestSourceMapRegistry_TranslateLocation(t *testing.T) {
	registry := NewSourceMapRegistry()
	sm := NewSourceMap("/path/to/source.cdt", "/path/to/generated.go")
	sm.AddMapping(10, 0, 15, 0)
	sm.AddMapping(20, 5, 30, 10)
	sm.AddMapping(30, 0, 45, 0)
	registry.Register(sm)

	// Test exact match
	source, line, col, err := registry.TranslateLocation("/path/to/generated.go", 15)
	require.NoError(t, err)
	assert.Equal(t, "/path/to/source.cdt", source)
	assert.Equal(t, 10, line)
	assert.Equal(t, 0, col)

	// Test closest match (line 16 should map to closest line 15)
	source, line, col, err = registry.TranslateLocation("/path/to/generated.go", 16)
	require.NoError(t, err)
	assert.Equal(t, "/path/to/source.cdt", source)
	assert.Equal(t, 10, line)
	assert.Equal(t, 0, col)

	// Test another line with column
	source, line, col, err = registry.TranslateLocation("/path/to/generated.go", 30)
	require.NoError(t, err)
	assert.Equal(t, "/path/to/source.cdt", source)
	assert.Equal(t, 20, line)
	assert.Equal(t, 5, col)
}

func TestSourceMapRegistry_TranslateLocation_NoSourceMap(t *testing.T) {
	registry := NewSourceMapRegistry()

	_, _, _, err := registry.TranslateLocation("/path/to/nonexistent.go", 10)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no source map")
}

func TestSourceMapRegistry_TranslateLocation_NoMappings(t *testing.T) {
	registry := NewSourceMapRegistry()
	sm := NewSourceMap("/path/to/source.cdt", "/path/to/generated.go")
	// Don't add any mappings
	registry.Register(sm)

	_, _, _, err := registry.TranslateLocation("/path/to/generated.go", 10)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no mapping found")
}

func TestSourceMap_SaveToFile(t *testing.T) {
	tempDir := t.TempDir()
	filePath := filepath.Join(tempDir, "test.map.json")

	sm := NewSourceMap("/path/to/source.cdt", "/path/to/generated.go")
	sm.AddMapping(10, 0, 15, 0)
	sm.AddMapping(20, 5, 30, 10)

	err := sm.SaveToFile(filePath)
	require.NoError(t, err)

	// Verify file exists
	_, err = os.Stat(filePath)
	require.NoError(t, err)

	// Read and verify content
	data, err := os.ReadFile(filePath)
	require.NoError(t, err)
	assert.Contains(t, string(data), "sourceFile")
	assert.Contains(t, string(data), "/path/to/source.cdt")
	assert.Contains(t, string(data), "/path/to/generated.go")
}

func TestSourceMapRegistry_LoadFromDirectory(t *testing.T) {
	tempDir := t.TempDir()

	// Create test source maps
	sm1 := NewSourceMap("/path/to/source1.cdt", "/path/to/generated1.go")
	sm1.AddMapping(10, 0, 15, 0)
	err := sm1.SaveToFile(filepath.Join(tempDir, "source1.map.json"))
	require.NoError(t, err)

	sm2 := NewSourceMap("/path/to/source2.cdt", "/path/to/generated2.go")
	sm2.AddMapping(20, 0, 30, 0)
	err = sm2.SaveToFile(filepath.Join(tempDir, "source2.map.json"))
	require.NoError(t, err)

	// Load from directory
	registry := NewSourceMapRegistry()
	err = registry.LoadFromDirectory(tempDir)
	require.NoError(t, err)

	// Verify both source maps were loaded
	retrieved1, ok := registry.GetSourceMap("/path/to/source1.cdt")
	assert.True(t, ok)
	assert.NotNil(t, retrieved1)

	retrieved2, ok := registry.GetSourceMap("/path/to/source2.cdt")
	assert.True(t, ok)
	assert.NotNil(t, retrieved2)
}

func TestSourceMapRegistry_LoadFromDirectory_EmptyDir(t *testing.T) {
	tempDir := t.TempDir()

	registry := NewSourceMapRegistry()
	err := registry.LoadFromDirectory(tempDir)
	require.NoError(t, err)

	// Verify no source maps loaded
	assert.Empty(t, registry.maps)
}

func TestSourceMapRegistry_LoadFromDirectory_NonexistentDir(t *testing.T) {
	registry := NewSourceMapRegistry()
	err := registry.LoadFromDirectory("/nonexistent/directory")
	require.NoError(t, err) // Glob returns no error for nonexistent dirs

	assert.Empty(t, registry.maps)
}

func TestLineMapping_Structure(t *testing.T) {
	mapping := &LineMapping{
		SourceLine:      10,
		SourceColumn:    5,
		GeneratedLine:   15,
		GeneratedColumn: 8,
	}

	assert.Equal(t, 10, mapping.SourceLine)
	assert.Equal(t, 5, mapping.SourceColumn)
	assert.Equal(t, 15, mapping.GeneratedLine)
	assert.Equal(t, 8, mapping.GeneratedColumn)
}

func TestAbs(t *testing.T) {
	assert.Equal(t, 5, abs(5))
	assert.Equal(t, 5, abs(-5))
	assert.Equal(t, 0, abs(0))
	assert.Equal(t, 100, abs(100))
	assert.Equal(t, 100, abs(-100))
}

func TestSourceMapRegistry_ConcurrentAccess(t *testing.T) {
	registry := NewSourceMapRegistry()
	sm := NewSourceMap("/path/to/source.cdt", "/path/to/generated.go")
	sm.AddMapping(10, 0, 15, 0)

	// Test concurrent register and read
	done := make(chan bool, 2)

	go func() {
		for i := 0; i < 100; i++ {
			registry.Register(sm)
		}
		done <- true
	}()

	go func() {
		for i := 0; i < 100; i++ {
			registry.GetSourceMap("/path/to/source.cdt")
		}
		done <- true
	}()

	<-done
	<-done

	// Verify final state
	retrieved, ok := registry.GetSourceMap("/path/to/source.cdt")
	assert.True(t, ok)
	assert.NotNil(t, retrieved)
}

func TestSourceMapRegistry_MultipleSourceFiles(t *testing.T) {
	registry := NewSourceMapRegistry()

	// Register multiple source maps
	files := []string{"a.cdt", "b.cdt", "c.cdt"}
	for _, file := range files {
		sm := NewSourceMap(file, file+".go")
		sm.AddMapping(10, 0, 15, 0)
		registry.Register(sm)
	}

	// Verify all can be retrieved
	for _, file := range files {
		retrieved, ok := registry.GetSourceMap(file)
		assert.True(t, ok)
		assert.NotNil(t, retrieved)
		assert.Equal(t, file, retrieved.SourceFile)
	}
}

func TestSourceMap_ComplexMappings(t *testing.T) {
	sm := NewSourceMap("/path/to/source.cdt", "/path/to/generated.go")

	// Add many mappings to simulate real compilation
	for i := 0; i < 100; i++ {
		sourceLine := i * 2
		generatedLine := i*3 + 5
		sm.AddMapping(sourceLine, 0, generatedLine, 0)
	}

	assert.Len(t, sm.Mappings, 100)

	// Verify first and last mappings
	assert.Equal(t, 0, sm.Mappings[0].SourceLine)
	assert.Equal(t, 5, sm.Mappings[0].GeneratedLine)

	assert.Equal(t, 198, sm.Mappings[99].SourceLine)
	assert.Equal(t, 302, sm.Mappings[99].GeneratedLine)
}

func TestSourceMapRegistry_TranslateBreakpoint_ClosestMatch(t *testing.T) {
	registry := NewSourceMapRegistry()
	sm := NewSourceMap("/path/to/source.cdt", "/path/to/generated.go")

	// Add sparse mappings
	sm.AddMapping(10, 0, 15, 0)
	sm.AddMapping(30, 0, 45, 0)
	sm.AddMapping(50, 0, 75, 0)
	registry.Register(sm)

	// Test line 20 (between 10 and 30) should map to closest
	bp, err := registry.TranslateBreakpoint("/path/to/source.cdt", 20)
	require.NoError(t, err)
	// Distance calc: |20-10|=10, |20-30|=10, |20-50|=30
	// When distances are equal, it picks the first one found
	assert.Equal(t, 10, bp.SourceLine)
	assert.Equal(t, 15, bp.GeneratedLine)

	// Test line 25
	bp, err = registry.TranslateBreakpoint("/path/to/source.cdt", 25)
	require.NoError(t, err)
	// Distance calc: |25-10|=15, |25-30|=5, |25-50|=25
	// Closest is 30 (distance 5)
	assert.Equal(t, 30, bp.SourceLine)
	assert.Equal(t, 45, bp.GeneratedLine)
}
