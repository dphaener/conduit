package debug

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewBreakpointManager(t *testing.T) {
	bm := NewBreakpointManager()
	assert.NotNil(t, bm)
	assert.NotNil(t, bm.breakpoints)
	assert.Empty(t, bm.breakpoints)
	assert.Equal(t, 1, bm.nextID)
}

func TestBreakpointManager_Add(t *testing.T) {
	bm := NewBreakpointManager()

	bp1 := &Breakpoint{
		SourceFile:    "/path/to/source.cdt",
		SourceLine:    10,
		GeneratedFile: "/path/to/generated.go",
		GeneratedLine: 15,
		Verified:      true,
	}

	id := bm.Add(bp1)
	assert.Equal(t, 1, id)
	assert.Equal(t, 1, bp1.ID)

	bp2 := &Breakpoint{
		SourceFile:    "/path/to/source.cdt",
		SourceLine:    20,
		GeneratedFile: "/path/to/generated.go",
		GeneratedLine: 30,
		Verified:      true,
	}

	id = bm.Add(bp2)
	assert.Equal(t, 2, id)
	assert.Equal(t, 2, bp2.ID)
}

func TestBreakpointManager_Get(t *testing.T) {
	bm := NewBreakpointManager()

	bp := &Breakpoint{
		SourceFile:    "/path/to/source.cdt",
		SourceLine:    10,
		GeneratedFile: "/path/to/generated.go",
		GeneratedLine: 15,
		Verified:      true,
	}

	id := bm.Add(bp)

	retrieved, err := bm.Get(id)
	require.NoError(t, err)
	assert.Equal(t, bp, retrieved)
	assert.Equal(t, id, retrieved.ID)
	assert.Equal(t, "/path/to/source.cdt", retrieved.SourceFile)
	assert.Equal(t, 10, retrieved.SourceLine)
}

func TestBreakpointManager_Get_NotFound(t *testing.T) {
	bm := NewBreakpointManager()

	_, err := bm.Get(999)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestBreakpointManager_Remove(t *testing.T) {
	bm := NewBreakpointManager()

	bp := &Breakpoint{
		SourceFile:    "/path/to/source.cdt",
		SourceLine:    10,
		GeneratedFile: "/path/to/generated.go",
		GeneratedLine: 15,
	}

	id := bm.Add(bp)

	err := bm.Remove(id)
	require.NoError(t, err)

	_, err = bm.Get(id)
	assert.Error(t, err)
}

func TestBreakpointManager_Remove_NotFound(t *testing.T) {
	bm := NewBreakpointManager()

	err := bm.Remove(999)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestBreakpointManager_GetBySourceLocation(t *testing.T) {
	bm := NewBreakpointManager()

	bp1 := &Breakpoint{
		SourceFile:    "/path/to/source.cdt",
		SourceLine:    10,
		GeneratedFile: "/path/to/generated.go",
		GeneratedLine: 15,
	}

	bp2 := &Breakpoint{
		SourceFile:    "/path/to/source.cdt",
		SourceLine:    10,
		GeneratedFile: "/path/to/generated.go",
		GeneratedLine: 16,
	}

	bp3 := &Breakpoint{
		SourceFile:    "/path/to/source.cdt",
		SourceLine:    20,
		GeneratedFile: "/path/to/generated.go",
		GeneratedLine: 30,
	}

	bm.Add(bp1)
	bm.Add(bp2)
	bm.Add(bp3)

	// Get breakpoints for line 10
	bps := bm.GetBySourceLocation("/path/to/source.cdt", 10)
	assert.Len(t, bps, 2)

	// Get breakpoints for line 20
	bps = bm.GetBySourceLocation("/path/to/source.cdt", 20)
	assert.Len(t, bps, 1)
	assert.Equal(t, 20, bps[0].SourceLine)

	// Get breakpoints for non-existent line
	bps = bm.GetBySourceLocation("/path/to/source.cdt", 99)
	assert.Empty(t, bps)
}

func TestBreakpointManager_GetByGeneratedLocation(t *testing.T) {
	bm := NewBreakpointManager()

	bp1 := &Breakpoint{
		SourceFile:    "/path/to/source.cdt",
		SourceLine:    10,
		GeneratedFile: "/path/to/generated.go",
		GeneratedLine: 15,
	}

	bp2 := &Breakpoint{
		SourceFile:    "/path/to/source.cdt",
		SourceLine:    20,
		GeneratedFile: "/path/to/generated.go",
		GeneratedLine: 30,
	}

	bm.Add(bp1)
	bm.Add(bp2)

	// Get breakpoints for generated line 15
	bps := bm.GetByGeneratedLocation("/path/to/generated.go", 15)
	assert.Len(t, bps, 1)
	assert.Equal(t, 10, bps[0].SourceLine)

	// Get breakpoints for generated line 30
	bps = bm.GetByGeneratedLocation("/path/to/generated.go", 30)
	assert.Len(t, bps, 1)
	assert.Equal(t, 20, bps[0].SourceLine)

	// Get breakpoints for non-existent line
	bps = bm.GetByGeneratedLocation("/path/to/generated.go", 99)
	assert.Empty(t, bps)
}

func TestBreakpointManager_List(t *testing.T) {
	bm := NewBreakpointManager()

	// Empty list
	bps := bm.List()
	assert.Empty(t, bps)

	// Add breakpoints
	bp1 := &Breakpoint{SourceFile: "a.cdt", SourceLine: 10}
	bp2 := &Breakpoint{SourceFile: "b.cdt", SourceLine: 20}
	bp3 := &Breakpoint{SourceFile: "c.cdt", SourceLine: 30}

	bm.Add(bp1)
	bm.Add(bp2)
	bm.Add(bp3)

	bps = bm.List()
	assert.Len(t, bps, 3)
}

func TestBreakpointManager_Clear(t *testing.T) {
	bm := NewBreakpointManager()

	// Add breakpoints
	bp1 := &Breakpoint{SourceFile: "a.cdt", SourceLine: 10}
	bp2 := &Breakpoint{SourceFile: "b.cdt", SourceLine: 20}

	bm.Add(bp1)
	bm.Add(bp2)

	assert.Len(t, bm.List(), 2)

	// Clear all
	bm.Clear()

	assert.Empty(t, bm.List())
}

func TestBreakpointManager_ClearBySourceFile(t *testing.T) {
	bm := NewBreakpointManager()

	// Add breakpoints from different files
	bp1 := &Breakpoint{SourceFile: "a.cdt", SourceLine: 10}
	bp2 := &Breakpoint{SourceFile: "a.cdt", SourceLine: 20}
	bp3 := &Breakpoint{SourceFile: "b.cdt", SourceLine: 30}

	bm.Add(bp1)
	bm.Add(bp2)
	bm.Add(bp3)

	assert.Len(t, bm.List(), 3)

	// Clear breakpoints for a.cdt
	bm.ClearBySourceFile("a.cdt")

	bps := bm.List()
	assert.Len(t, bps, 1)
	assert.Equal(t, "b.cdt", bps[0].SourceFile)
}

func TestBreakpoint_SetVerified(t *testing.T) {
	bp := &Breakpoint{
		SourceFile: "/path/to/source.cdt",
		SourceLine: 10,
		Verified:   false,
	}

	assert.False(t, bp.IsVerified())

	bp.SetVerified(true)
	assert.True(t, bp.IsVerified())

	bp.SetVerified(false)
	assert.False(t, bp.IsVerified())
}

func TestBreakpoint_Condition(t *testing.T) {
	bp := &Breakpoint{
		SourceFile: "/path/to/source.cdt",
		SourceLine: 10,
	}

	assert.False(t, bp.HasCondition())
	assert.Equal(t, "", bp.GetCondition())

	bp.SetCondition("x > 5")
	assert.True(t, bp.HasCondition())
	assert.Equal(t, "x > 5", bp.GetCondition())

	bp.SetCondition("")
	assert.False(t, bp.HasCondition())
}

func TestBreakpoint_String(t *testing.T) {
	bp := &Breakpoint{
		ID:            1,
		SourceFile:    "/path/to/source.cdt",
		SourceLine:    10,
		GeneratedFile: "/path/to/generated.go",
		GeneratedLine: 15,
		Verified:      true,
	}

	str := bp.String()
	assert.Contains(t, str, "Breakpoint 1")
	assert.Contains(t, str, "/path/to/source.cdt:10")
	assert.Contains(t, str, "/path/to/generated.go:15")
	assert.Contains(t, str, "verified")
}

func TestBreakpoint_StringWithCondition(t *testing.T) {
	bp := &Breakpoint{
		ID:            1,
		SourceFile:    "/path/to/source.cdt",
		SourceLine:    10,
		GeneratedFile: "/path/to/generated.go",
		GeneratedLine: 15,
		Verified:      true,
		Condition:     "x > 5",
	}

	str := bp.String()
	assert.Contains(t, str, "condition: x > 5")
}

func TestBreakpoint_StringUnverified(t *testing.T) {
	bp := &Breakpoint{
		ID:            1,
		SourceFile:    "/path/to/source.cdt",
		SourceLine:    10,
		GeneratedFile: "/path/to/generated.go",
		GeneratedLine: 15,
		Verified:      false,
	}

	str := bp.String()
	assert.Contains(t, str, "unverified")
}

func TestBreakpointManager_ConcurrentAdd(t *testing.T) {
	bm := NewBreakpointManager()
	done := make(chan bool, 10)

	// Add breakpoints concurrently
	for i := 0; i < 10; i++ {
		go func(line int) {
			bp := &Breakpoint{
				SourceFile: fmt.Sprintf("/path/to/file%d.cdt", line),
				SourceLine: line,
			}
			bm.Add(bp)
			done <- true
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}

	// Verify all breakpoints were added
	bps := bm.List()
	assert.Len(t, bps, 10)

	// Verify unique IDs
	ids := make(map[int]bool)
	for _, bp := range bps {
		assert.False(t, ids[bp.ID], "Duplicate ID found")
		ids[bp.ID] = true
	}
}

func TestBreakpointManager_ConcurrentGetAndRemove(t *testing.T) {
	bm := NewBreakpointManager()

	// Add initial breakpoints
	ids := make([]int, 10)
	for i := 0; i < 10; i++ {
		bp := &Breakpoint{
			SourceFile: fmt.Sprintf("/path/to/file%d.cdt", i),
			SourceLine: i,
		}
		ids[i] = bm.Add(bp)
	}

	done := make(chan bool, 20)

	// Concurrently get and remove
	for i := 0; i < 10; i++ {
		go func(id int) {
			bm.Get(id)
			done <- true
		}(ids[i])

		go func(id int) {
			bm.Remove(id)
			done <- true
		}(ids[i])
	}

	// Wait for all goroutines
	for i := 0; i < 20; i++ {
		<-done
	}

	// After concurrent operations, breakpoints should be removed
	bps := bm.List()
	assert.Empty(t, bps)
}

func TestBreakpoint_ConcurrentVerification(t *testing.T) {
	bp := &Breakpoint{
		SourceFile: "/path/to/source.cdt",
		SourceLine: 10,
		Verified:   false,
	}

	done := make(chan bool, 100)

	// Concurrently set and check verification
	for i := 0; i < 50; i++ {
		go func() {
			bp.SetVerified(true)
			done <- true
		}()

		go func() {
			bp.IsVerified()
			done <- true
		}()
	}

	// Wait for all goroutines
	for i := 0; i < 100; i++ {
		<-done
	}

	// Final state should be verified
	assert.True(t, bp.IsVerified())
}

func TestBreakpoint_ConcurrentConditionUpdate(t *testing.T) {
	bp := &Breakpoint{
		SourceFile: "/path/to/source.cdt",
		SourceLine: 10,
	}

	done := make(chan bool, 100)

	// Concurrently set and get condition
	for i := 0; i < 50; i++ {
		go func(val int) {
			bp.SetCondition(fmt.Sprintf("x > %d", val))
			done <- true
		}(i)

		go func() {
			bp.GetCondition()
			bp.HasCondition()
			done <- true
		}()
	}

	// Wait for all goroutines
	for i := 0; i < 100; i++ {
		<-done
	}

	// Should have some condition set
	assert.True(t, bp.HasCondition())
}

func TestBreakpointManager_MultipleFiles(t *testing.T) {
	bm := NewBreakpointManager()

	files := []string{"a.cdt", "b.cdt", "c.cdt"}

	// Add breakpoints for each file
	for _, file := range files {
		for line := 10; line <= 30; line += 10 {
			bp := &Breakpoint{
				SourceFile: file,
				SourceLine: line,
			}
			bm.Add(bp)
		}
	}

	// Verify total count
	assert.Len(t, bm.List(), 9) // 3 files * 3 breakpoints each

	// Verify per-file counts
	for _, file := range files {
		bps := bm.GetBySourceLocation(file, 10)
		assert.Len(t, bps, 1)
		bps = bm.GetBySourceLocation(file, 20)
		assert.Len(t, bps, 1)
		bps = bm.GetBySourceLocation(file, 30)
		assert.Len(t, bps, 1)
	}
}
