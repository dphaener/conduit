package debug

import (
	"fmt"
	"sync"
)

// Breakpoint represents a breakpoint in the debugger
type Breakpoint struct {
	ID            int
	SourceFile    string
	SourceLine    int
	GeneratedFile string
	GeneratedLine int
	Condition     string
	HitCondition  string
	LogMessage    string
	Verified      bool
	mutex         sync.RWMutex
}

// BreakpointManager manages breakpoints for debugging sessions
type BreakpointManager struct {
	breakpoints map[int]*Breakpoint
	nextID      int
	mutex       sync.RWMutex
}

// NewBreakpointManager creates a new breakpoint manager
func NewBreakpointManager() *BreakpointManager {
	return &BreakpointManager{
		breakpoints: make(map[int]*Breakpoint),
		nextID:      1,
	}
}

// Add adds a breakpoint to the manager
func (bm *BreakpointManager) Add(bp *Breakpoint) int {
	bm.mutex.Lock()
	defer bm.mutex.Unlock()

	id := bm.nextID
	bm.nextID++

	bp.ID = id
	bm.breakpoints[id] = bp

	return id
}

// Remove removes a breakpoint by ID
func (bm *BreakpointManager) Remove(id int) error {
	bm.mutex.Lock()
	defer bm.mutex.Unlock()

	if _, ok := bm.breakpoints[id]; !ok {
		return fmt.Errorf("breakpoint %d not found", id)
	}

	delete(bm.breakpoints, id)
	return nil
}

// Get retrieves a breakpoint by ID
func (bm *BreakpointManager) Get(id int) (*Breakpoint, error) {
	bm.mutex.RLock()
	defer bm.mutex.RUnlock()

	bp, ok := bm.breakpoints[id]
	if !ok {
		return nil, fmt.Errorf("breakpoint %d not found", id)
	}

	return bp, nil
}

// GetBySourceLocation retrieves breakpoints by source file and line
func (bm *BreakpointManager) GetBySourceLocation(sourceFile string, sourceLine int) []*Breakpoint {
	bm.mutex.RLock()
	defer bm.mutex.RUnlock()

	result := make([]*Breakpoint, 0)
	for _, bp := range bm.breakpoints {
		if bp.SourceFile == sourceFile && bp.SourceLine == sourceLine {
			result = append(result, bp)
		}
	}

	return result
}

// GetByGeneratedLocation retrieves breakpoints by generated file and line
func (bm *BreakpointManager) GetByGeneratedLocation(generatedFile string, generatedLine int) []*Breakpoint {
	bm.mutex.RLock()
	defer bm.mutex.RUnlock()

	result := make([]*Breakpoint, 0)
	for _, bp := range bm.breakpoints {
		if bp.GeneratedFile == generatedFile && bp.GeneratedLine == generatedLine {
			result = append(result, bp)
		}
	}

	return result
}

// List returns all breakpoints
func (bm *BreakpointManager) List() []*Breakpoint {
	bm.mutex.RLock()
	defer bm.mutex.RUnlock()

	result := make([]*Breakpoint, 0, len(bm.breakpoints))
	for _, bp := range bm.breakpoints {
		result = append(result, bp)
	}

	return result
}

// Clear removes all breakpoints
func (bm *BreakpointManager) Clear() {
	bm.mutex.Lock()
	defer bm.mutex.Unlock()

	bm.breakpoints = make(map[int]*Breakpoint)
}

// ClearBySourceFile removes all breakpoints for a source file
func (bm *BreakpointManager) ClearBySourceFile(sourceFile string) {
	bm.mutex.Lock()
	defer bm.mutex.Unlock()

	for id, bp := range bm.breakpoints {
		if bp.SourceFile == sourceFile {
			delete(bm.breakpoints, id)
		}
	}
}

// SetVerified marks a breakpoint as verified
func (bp *Breakpoint) SetVerified(verified bool) {
	bp.mutex.Lock()
	defer bp.mutex.Unlock()
	bp.Verified = verified
}

// IsVerified returns whether the breakpoint is verified
func (bp *Breakpoint) IsVerified() bool {
	bp.mutex.RLock()
	defer bp.mutex.RUnlock()
	return bp.Verified
}

// SetCondition sets the breakpoint condition
func (bp *Breakpoint) SetCondition(condition string) {
	bp.mutex.Lock()
	defer bp.mutex.Unlock()
	bp.Condition = condition
}

// GetCondition returns the breakpoint condition
func (bp *Breakpoint) GetCondition() string {
	bp.mutex.RLock()
	defer bp.mutex.RUnlock()
	return bp.Condition
}

// HasCondition returns whether the breakpoint has a condition
func (bp *Breakpoint) HasCondition() bool {
	bp.mutex.RLock()
	defer bp.mutex.RUnlock()
	return bp.Condition != ""
}

// String returns a string representation of the breakpoint
func (bp *Breakpoint) String() string {
	bp.mutex.RLock()
	defer bp.mutex.RUnlock()

	verified := "unverified"
	if bp.Verified {
		verified = "verified"
	}

	condition := ""
	if bp.Condition != "" {
		condition = fmt.Sprintf(" [condition: %s]", bp.Condition)
	}

	return fmt.Sprintf("Breakpoint %d: %s:%d -> %s:%d (%s)%s",
		bp.ID, bp.SourceFile, bp.SourceLine,
		bp.GeneratedFile, bp.GeneratedLine,
		verified, condition)
}
