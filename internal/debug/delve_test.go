package debug

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewDelveClient(t *testing.T) {
	client, err := NewDelveClient("dlv")
	require.NoError(t, err)
	assert.NotNil(t, client)
	assert.NotNil(t, client.breakpoints)
}

func TestNewDelveClient_DefaultPath(t *testing.T) {
	client, err := NewDelveClient("")
	require.NoError(t, err)
	assert.NotNil(t, client)
}

func TestDelveClient_BreakpointTracking(t *testing.T) {
	client, err := NewDelveClient("dlv")
	require.NoError(t, err)

	// Verify breakpoints map is initialized
	assert.NotNil(t, client.breakpoints)
	assert.Empty(t, client.breakpoints)
}

func TestDelveThread_Structure(t *testing.T) {
	thread := DelveThread{
		ID:   1,
		Name: "Main Thread",
	}

	assert.Equal(t, 1, thread.ID)
	assert.Equal(t, "Main Thread", thread.Name)
}

func TestDelveStackFrame_Structure(t *testing.T) {
	frame := DelveStackFrame{
		ID:       0,
		Function: "main.main",
		File:     "/path/to/main.go",
		Line:     42,
		Column:   10,
	}

	assert.Equal(t, 0, frame.ID)
	assert.Equal(t, "main.main", frame.Function)
	assert.Equal(t, "/path/to/main.go", frame.File)
	assert.Equal(t, 42, frame.Line)
	assert.Equal(t, 10, frame.Column)
}

func TestDelveVariable_Structure(t *testing.T) {
	variable := DelveVariable{
		Name:               "counter",
		Value:              "10",
		Type:               "int",
		VariablesReference: 0,
	}

	assert.Equal(t, "counter", variable.Name)
	assert.Equal(t, "10", variable.Value)
	assert.Equal(t, "int", variable.Type)
	assert.Equal(t, 0, variable.VariablesReference)
}

func TestDelveVariable_WithChildren(t *testing.T) {
	parent := DelveVariable{
		Name:               "person",
		Value:              "{...}",
		Type:               "Person",
		VariablesReference: 1,
		Children: []DelveVariable{
			{Name: "Name", Value: "John", Type: "string"},
			{Name: "Age", Value: "30", Type: "int"},
		},
	}

	assert.Equal(t, "person", parent.Name)
	assert.Len(t, parent.Children, 2)
	assert.Equal(t, "Name", parent.Children[0].Name)
	assert.Equal(t, "Age", parent.Children[1].Name)
}

func TestEvaluateResult_Structure(t *testing.T) {
	result := EvaluateResult{
		Value:              "42",
		Type:               "int",
		VariablesReference: 0,
	}

	assert.Equal(t, "42", result.Value)
	assert.Equal(t, "int", result.Type)
	assert.Equal(t, 0, result.VariablesReference)
}

func TestDelveClient_ErrorHandling(t *testing.T) {
	client, err := NewDelveClient("dlv")
	require.NoError(t, err)

	// Test operations without connection should fail
	err = client.Continue(1)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not connected")

	err = client.Next(1)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not connected")

	err = client.StepIn(1)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not connected")

	err = client.StepOut(1)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not connected")

	_, err = client.ListThreads()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not connected")

	_, err = client.StackTrace(1)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not connected")

	_, err = client.ListVariables(1)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not connected")

	_, err = client.Evaluate("x", 0)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not connected")
}

func TestDelveClient_CreateBreakpoint_NoConnection(t *testing.T) {
	client, err := NewDelveClient("dlv")
	require.NoError(t, err)

	_, err = client.CreateBreakpoint("/path/to/file.go", 10, "")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not connected")
}

func TestDelveClient_ClearBreakpoint_NoConnection(t *testing.T) {
	client, err := NewDelveClient("dlv")
	require.NoError(t, err)

	err = client.ClearBreakpoint(1)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not connected")
}

func TestDelveClient_Close(t *testing.T) {
	client, err := NewDelveClient("dlv")
	require.NoError(t, err)

	err = client.Close()
	assert.NoError(t, err)
}

func TestDelveClient_Detach_NoConnection(t *testing.T) {
	client, err := NewDelveClient("dlv")
	require.NoError(t, err)

	err = client.Detach()
	assert.NoError(t, err) // Should not error when nothing to detach
}

// Mock tests for when we have a connection
// These would require integration testing with a real Delve instance

func TestDelveClient_ThreadManagement(t *testing.T) {
	// Verify thread structure
	threads := []DelveThread{
		{ID: 1, Name: "Thread 1"},
		{ID: 2, Name: "Thread 2"},
	}

	assert.Len(t, threads, 2)
	assert.Equal(t, 1, threads[0].ID)
	assert.Equal(t, 2, threads[1].ID)
}

func TestDelveClient_StackTraceFormat(t *testing.T) {
	// Verify stack frame structure
	frames := []DelveStackFrame{
		{
			ID:       0,
			Function: "main.processRequest",
			File:     "/app/handlers.go",
			Line:     45,
			Column:   5,
		},
		{
			ID:       1,
			Function: "main.main",
			File:     "/app/main.go",
			Line:     20,
			Column:   10,
		},
	}

	assert.Len(t, frames, 2)
	assert.Equal(t, "main.processRequest", frames[0].Function)
	assert.Equal(t, "main.main", frames[1].Function)
	assert.Equal(t, 45, frames[0].Line)
	assert.Equal(t, 20, frames[1].Line)
}

func TestDelveClient_VariableInspection(t *testing.T) {
	// Verify variable structure
	variables := []DelveVariable{
		{Name: "x", Value: "42", Type: "int"},
		{Name: "name", Value: "Alice", Type: "string"},
		{Name: "active", Value: "true", Type: "bool"},
	}

	assert.Len(t, variables, 3)
	assert.Equal(t, "x", variables[0].Name)
	assert.Equal(t, "42", variables[0].Value)
	assert.Equal(t, "int", variables[0].Type)
}

func TestDelveClient_ConditionalBreakpoint(t *testing.T) {
	client, err := NewDelveClient("dlv")
	require.NoError(t, err)

	// Attempt to create conditional breakpoint (will fail without connection)
	_, err = client.CreateBreakpoint("/path/to/file.go", 10, "x > 5")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not connected")
}
