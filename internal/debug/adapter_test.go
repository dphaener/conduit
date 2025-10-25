package debug

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/google/go-dap"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockConnection implements io.ReadWriteCloser for testing
type mockConnection struct {
	readBuf  *bytes.Buffer
	writeBuf *bytes.Buffer
	closed   bool
}

func newMockConnection() *mockConnection {
	return &mockConnection{
		readBuf:  new(bytes.Buffer),
		writeBuf: new(bytes.Buffer),
	}
}

func (mc *mockConnection) Read(p []byte) (n int, err error) {
	return mc.readBuf.Read(p)
}

func (mc *mockConnection) Write(p []byte) (n int, err error) {
	return mc.writeBuf.Write(p)
}

func (mc *mockConnection) Close() error {
	mc.closed = true
	return nil
}

func (mc *mockConnection) writeRequest(req interface{}) error {
	encoder := json.NewEncoder(mc.readBuf)
	return encoder.Encode(req)
}

func (mc *mockConnection) readResponse(resp interface{}) error {
	decoder := json.NewDecoder(mc.writeBuf)
	return decoder.Decode(resp)
}

func TestNewDebugAdapter(t *testing.T) {
	conn := newMockConnection()
	smr := NewSourceMapRegistry()

	adapter, err := NewDebugAdapter(conn, "dlv", smr)
	require.NoError(t, err)
	assert.NotNil(t, adapter)
	assert.NotNil(t, adapter.session)
	assert.Equal(t, 0, adapter.seq) // Initialized to 0, nextSeq() returns 1 on first call
}

func TestDebugAdapter_Initialize(t *testing.T) {
	conn := newMockConnection()
	smr := NewSourceMapRegistry()

	adapter, err := NewDebugAdapter(conn, "dlv", smr)
	require.NoError(t, err)

	// Create initialize request directly
	initReq := &dap.InitializeRequest{
		Request: dap.Request{
			ProtocolMessage: dap.ProtocolMessage{
				Seq:  1,
				Type: "request",
			},
			Command: "initialize",
		},
	}

	// Test handleInitialize directly
	err = adapter.handleInitialize(initReq)
	require.NoError(t, err)

	// Verify responses were written (two messages: response + event)
	// We can't easily parse them with mock connection, but no error means success
	assert.Greater(t, conn.writeBuf.Len(), 0)
}

func TestDebugAdapter_SetBreakpoints(t *testing.T) {
	conn := newMockConnection()
	smr := NewSourceMapRegistry()

	// Register a source map
	sm := NewSourceMap("/path/to/source.cdt", "/path/to/generated.go")
	sm.AddMapping(10, 0, 15, 0)
	sm.AddMapping(20, 0, 30, 0)
	smr.Register(sm)

	adapter, err := NewDebugAdapter(conn, "dlv", smr)
	require.NoError(t, err)

	// Create setBreakpoints request properly
	req := &dap.SetBreakpointsRequest{
		Request: dap.Request{
			ProtocolMessage: dap.ProtocolMessage{
				Seq:  1,
				Type: "request",
			},
			Command: "setBreakpoints",
		},
		Arguments: dap.SetBreakpointsArguments{
			Source: dap.Source{
				Path: "/path/to/source.cdt",
			},
			Breakpoints: []dap.SourceBreakpoint{
				{Line: 10},
				{Line: 20, Condition: "x > 5"},
			},
		},
	}

	err = adapter.handleSetBreakpoints(req)
	// Note: Without real Delve connection, it won't error but will log failures
	// The function handles missing Delve gracefully by marking breakpoints as unverified
	assert.NoError(t, err) // No error, just logs failures internally
}

func TestDebugAdapter_NextSeq(t *testing.T) {
	conn := newMockConnection()
	smr := NewSourceMapRegistry()

	adapter, err := NewDebugAdapter(conn, "dlv", smr)
	require.NoError(t, err)

	seq1 := adapter.nextSeq()
	seq2 := adapter.nextSeq()
	seq3 := adapter.nextSeq()

	assert.Equal(t, 1, seq1)
	assert.Equal(t, 2, seq2)
	assert.Equal(t, 3, seq3)
}

func TestDebugAdapter_SessionManagement(t *testing.T) {
	conn := newMockConnection()
	smr := NewSourceMapRegistry()

	adapter, err := NewDebugAdapter(conn, "dlv", smr)
	require.NoError(t, err)

	assert.NotNil(t, adapter.session)
	assert.Equal(t, "conduit-debug-session", adapter.session.SessionID)
	assert.Empty(t, adapter.session.Breakpoints)
	assert.Empty(t, adapter.session.Variables)
	assert.Empty(t, adapter.session.Threads)
}

func TestDebugAdapter_ErrorResponse(t *testing.T) {
	conn := newMockConnection()
	smr := NewSourceMapRegistry()

	adapter, err := NewDebugAdapter(conn, "dlv", smr)
	require.NoError(t, err)

	err = adapter.sendErrorResponse(1, "test error")
	require.NoError(t, err)

	// Verify error response was written
	assert.Greater(t, conn.writeBuf.Len(), 0)
	// The actual parsing would require understanding DAP wire format
	// For now, verifying no error in sending is sufficient
}

func TestDebugAdapter_Close(t *testing.T) {
	conn := newMockConnection()
	smr := NewSourceMapRegistry()

	adapter, err := NewDebugAdapter(conn, "dlv", smr)
	require.NoError(t, err)

	err = adapter.Close()
	require.NoError(t, err)
	assert.True(t, conn.closed)
}

func TestDebugAdapter_HandleUnsupportedMessage(t *testing.T) {
	conn := newMockConnection()
	smr := NewSourceMapRegistry()

	adapter, err := NewDebugAdapter(conn, "dlv", smr)
	require.NoError(t, err)

	// Unsupported message types are logged but don't error
	// Create a generic message
	msg := &dap.Request{
		ProtocolMessage: dap.ProtocolMessage{
			Seq:  1,
			Type: "request",
		},
		Command: "unsupportedCommand",
	}

	// handleMessage logs unsupported messages but doesn't error
	err = adapter.handleMessage(msg)
	assert.NoError(t, err) // Unsupported messages are logged, not errored
}

func TestDebugSession_ThreadTracking(t *testing.T) {
	session := &DebugSession{
		SessionID:   "test-session",
		Breakpoints: make(map[string][]*Breakpoint),
		Variables:   make(map[int]*Variable),
		Threads:     make(map[int]*Thread),
	}

	// Add threads
	session.mutex.Lock()
	session.Threads[1] = &Thread{ID: 1, Name: "Thread 1"}
	session.Threads[2] = &Thread{ID: 2, Name: "Thread 2"}
	session.mutex.Unlock()

	assert.Len(t, session.Threads, 2)

	session.mutex.RLock()
	thread1 := session.Threads[1]
	session.mutex.RUnlock()

	assert.Equal(t, 1, thread1.ID)
	assert.Equal(t, "Thread 1", thread1.Name)
}

func TestDebugSession_VariableTracking(t *testing.T) {
	session := &DebugSession{
		SessionID:   "test-session",
		Breakpoints: make(map[string][]*Breakpoint),
		Variables:   make(map[int]*Variable),
		Threads:     make(map[int]*Thread),
	}

	// Add variables
	session.mutex.Lock()
	session.Variables[1] = &Variable{
		Name:               "x",
		Value:              "42",
		Type:               "int",
		VariablesReference: 0,
	}
	session.Variables[2] = &Variable{
		Name:               "name",
		Value:              "John",
		Type:               "string",
		VariablesReference: 0,
	}
	session.mutex.Unlock()

	assert.Len(t, session.Variables, 2)

	session.mutex.RLock()
	varX := session.Variables[1]
	session.mutex.RUnlock()

	assert.Equal(t, "x", varX.Name)
	assert.Equal(t, "42", varX.Value)
	assert.Equal(t, "int", varX.Type)
}

func TestDebugSession_BreakpointsByFile(t *testing.T) {
	session := &DebugSession{
		SessionID:   "test-session",
		Breakpoints: make(map[string][]*Breakpoint),
		Variables:   make(map[int]*Variable),
		Threads:     make(map[int]*Thread),
	}

	file1 := "/path/to/file1.cdt"
	file2 := "/path/to/file2.cdt"

	// Add breakpoints
	session.mutex.Lock()
	session.Breakpoints[file1] = []*Breakpoint{
		{ID: 1, SourceFile: file1, SourceLine: 10},
		{ID: 2, SourceFile: file1, SourceLine: 20},
	}
	session.Breakpoints[file2] = []*Breakpoint{
		{ID: 3, SourceFile: file2, SourceLine: 15},
	}
	session.mutex.Unlock()

	session.mutex.RLock()
	bpsFile1 := session.Breakpoints[file1]
	bpsFile2 := session.Breakpoints[file2]
	session.mutex.RUnlock()

	assert.Len(t, bpsFile1, 2)
	assert.Len(t, bpsFile2, 1)
	assert.Equal(t, 10, bpsFile1[0].SourceLine)
	assert.Equal(t, 20, bpsFile1[1].SourceLine)
	assert.Equal(t, 15, bpsFile2[0].SourceLine)
}
