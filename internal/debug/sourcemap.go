package debug

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

// SourceMapRegistry manages source maps for Conduit to Go translation
type SourceMapRegistry struct {
	maps  map[string]*SourceMap
	mutex sync.RWMutex
}

// SourceMap represents the mapping between Conduit source and generated Go code
type SourceMap struct {
	SourceFile    string         `json:"sourceFile"`
	GeneratedFile string         `json:"generatedFile"`
	Mappings      []*LineMapping `json:"mappings"`
}

// LineMapping represents a single line mapping from source to generated code
type LineMapping struct {
	SourceLine      int `json:"sourceLine"`
	SourceColumn    int `json:"sourceColumn"`
	GeneratedLine   int `json:"generatedLine"`
	GeneratedColumn int `json:"generatedColumn"`
}

// NewSourceMapRegistry creates a new source map registry
func NewSourceMapRegistry() *SourceMapRegistry {
	return &SourceMapRegistry{
		maps: make(map[string]*SourceMap),
	}
}

// LoadFromDirectory loads all source maps from a directory
func (smr *SourceMapRegistry) LoadFromDirectory(dir string) error {
	smr.mutex.Lock()
	defer smr.mutex.Unlock()

	pattern := filepath.Join(dir, "*.map.json")
	files, err := filepath.Glob(pattern)
	if err != nil {
		return fmt.Errorf("failed to find source maps: %w", err)
	}

	for _, file := range files {
		if err := smr.loadSourceMapFile(file); err != nil {
			return fmt.Errorf("failed to load source map %s: %w", file, err)
		}
	}

	return nil
}

// loadSourceMapFile loads a single source map file
func (smr *SourceMapRegistry) loadSourceMapFile(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	var sm SourceMap
	if err := json.Unmarshal(data, &sm); err != nil {
		return err
	}

	smr.maps[sm.SourceFile] = &sm
	return nil
}

// Register registers a source map
func (smr *SourceMapRegistry) Register(sm *SourceMap) {
	smr.mutex.Lock()
	defer smr.mutex.Unlock()
	smr.maps[sm.SourceFile] = sm
}

// GetSourceMap retrieves a source map by source file path
func (smr *SourceMapRegistry) GetSourceMap(sourceFile string) (*SourceMap, bool) {
	smr.mutex.RLock()
	defer smr.mutex.RUnlock()
	sm, ok := smr.maps[sourceFile]
	return sm, ok
}

// TranslateBreakpoint translates a breakpoint from source to generated code
func (smr *SourceMapRegistry) TranslateBreakpoint(sourceFile string, sourceLine int) (*Breakpoint, error) {
	smr.mutex.RLock()
	defer smr.mutex.RUnlock()

	sourceMap, ok := smr.maps[sourceFile]
	if !ok {
		return nil, fmt.Errorf("no source map for %s", sourceFile)
	}

	// Find the closest mapping for the source line
	var bestMapping *LineMapping
	minDistance := int(^uint(0) >> 1) // max int

	for _, mapping := range sourceMap.Mappings {
		// Exact match - use it immediately
		if mapping.SourceLine == sourceLine {
			bestMapping = mapping
			break
		}

		distance := abs(mapping.SourceLine - sourceLine)
		if distance < minDistance {
			minDistance = distance
			bestMapping = mapping
		}
	}

	if bestMapping == nil {
		return nil, fmt.Errorf("no mapping found for line %d in %s", sourceLine, sourceFile)
	}

	return &Breakpoint{
		SourceFile:    sourceFile,
		SourceLine:    bestMapping.SourceLine,
		GeneratedFile: sourceMap.GeneratedFile,
		GeneratedLine: bestMapping.GeneratedLine,
		Verified:      false,
	}, nil
}

// TranslateLocation translates a location from generated code back to source
func (smr *SourceMapRegistry) TranslateLocation(generatedFile string, generatedLine int) (string, int, int, error) {
	smr.mutex.RLock()
	defer smr.mutex.RUnlock()

	// Find source map that generated this file
	var sourceMap *SourceMap
	for _, sm := range smr.maps {
		if sm.GeneratedFile == generatedFile {
			sourceMap = sm
			break
		}
	}

	if sourceMap == nil {
		return "", 0, 0, fmt.Errorf("no source map for generated file %s", generatedFile)
	}

	// Find the closest mapping for the generated line
	var bestMapping *LineMapping
	minDistance := int(^uint(0) >> 1) // max int

	for _, mapping := range sourceMap.Mappings {
		distance := abs(mapping.GeneratedLine - generatedLine)
		if distance < minDistance {
			minDistance = distance
			bestMapping = mapping
		}

		// Exact match - use it
		if mapping.GeneratedLine == generatedLine {
			bestMapping = mapping
			break
		}
	}

	if bestMapping == nil {
		return "", 0, 0, fmt.Errorf("no mapping found for generated line %d in %s", generatedLine, generatedFile)
	}

	return sourceMap.SourceFile, bestMapping.SourceLine, bestMapping.SourceColumn, nil
}

// SaveToFile saves a source map to a file
func (sm *SourceMap) SaveToFile(path string) error {
	data, err := json.MarshalIndent(sm, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal source map: %w", err)
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write source map: %w", err)
	}

	return nil
}

// AddMapping adds a line mapping to the source map
func (sm *SourceMap) AddMapping(sourceLine, sourceColumn, generatedLine, generatedColumn int) {
	sm.Mappings = append(sm.Mappings, &LineMapping{
		SourceLine:      sourceLine,
		SourceColumn:    sourceColumn,
		GeneratedLine:   generatedLine,
		GeneratedColumn: generatedColumn,
	})
}

// NewSourceMap creates a new source map
func NewSourceMap(sourceFile, generatedFile string) *SourceMap {
	return &SourceMap{
		SourceFile:    sourceFile,
		GeneratedFile: generatedFile,
		Mappings:      make([]*LineMapping, 0),
	}
}

// abs returns the absolute value of an integer
func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}
