package debug

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"sync"
	"time"

	"github.com/go-delve/delve/service/api"
	"github.com/go-delve/delve/service/rpc2"
)

// DelveClient wraps the Delve debugger client
type DelveClient struct {
	client      *rpc2.RPCClient
	serverCmd   *exec.Cmd
	processID   int
	mutex       sync.Mutex
	breakpoints map[int]*api.Breakpoint
}

// DelveThread represents a thread from Delve
type DelveThread struct {
	ID   int
	Name string
}

// DelveStackFrame represents a stack frame from Delve
type DelveStackFrame struct {
	ID       int
	Function string
	File     string
	Line     int
	Column   int
}

// DelveVariable represents a variable from Delve
type DelveVariable struct {
	Name               string
	Value              string
	Type               string
	VariablesReference int
	Children           []DelveVariable
}

// EvaluateResult represents the result of an expression evaluation
type EvaluateResult struct {
	Value              string
	Type               string
	VariablesReference int
}

// NewDelveClient creates a new Delve client instance
func NewDelveClient(delvePath string) (*DelveClient, error) {
	if delvePath == "" {
		delvePath = "dlv"
	}

	return &DelveClient{
		breakpoints: make(map[int]*api.Breakpoint),
	}, nil
}

// LaunchWithArgs starts the debugged program using Delve with provided arguments
func (dc *DelveClient) LaunchWithArgs(launchArgs map[string]interface{}) error {
	dc.mutex.Lock()
	defer dc.mutex.Unlock()

	// Extract program path from args
	programPath, ok := launchArgs["program"].(string)
	if !ok || programPath == "" {
		return fmt.Errorf("program path not specified")
	}

	// Validate program exists and is executable
	info, err := os.Stat(programPath)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("program not found: %s", programPath)
		}
		return fmt.Errorf("failed to stat program: %w", err)
	}

	if info.IsDir() {
		return fmt.Errorf("program path is a directory: %s", programPath)
	}

	if info.Mode()&0111 == 0 {
		return fmt.Errorf("program is not executable: %s", programPath)
	}

	// Build command to start program under Delve on a random port
	cmd := exec.Command("dlv", "exec", programPath, "--headless", "--listen=:0", "--api-version=2", "--accept-multiclient")

	// Create pipes to capture Delve output for port parsing
	stderrPipe, err := cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("failed to create stderr pipe: %w", err)
	}

	// Tee stderr to os.Stderr for visibility while still parsing for port
	teeReader := io.TeeReader(stderrPipe, os.Stderr)

	// Set stdout to os.Stdout for visibility
	cmd.Stdout = os.Stdout

	// Start Delve
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start Delve: %w", err)
	}

	dc.serverCmd = cmd

	// Track error for deferred cleanup
	var launchErr error
	defer func() {
		if launchErr != nil && dc.serverCmd != nil && dc.serverCmd.Process != nil {
			dc.serverCmd.Process.Kill()
			// Wait for process to finish (non-blocking with timeout)
			done := make(chan error, 1)
			go func() {
				done <- dc.serverCmd.Wait()
			}()
			select {
			case <-done:
				// Process reaped successfully
			case <-time.After(2 * time.Second):
				log.Printf("Warning: Delve process did not exit cleanly during cleanup")
			}
			dc.serverCmd = nil
		}
	}()

	// Parse output for the actual port Delve is listening on
	port, parseErr := parseDelvePortFromPipe(teeReader, 5*time.Second)
	if parseErr != nil {
		launchErr = fmt.Errorf("failed to get Delve port: %w", parseErr)
		return launchErr
	}

	// Connect to Delve with the actual port
	client := rpc2.NewClient(fmt.Sprintf("localhost:%d", port))
	dc.client = client

	return nil
}

// parseDelvePortFromPipe reads from a pipe and extracts the port number from Delve's output
func parseDelvePortFromPipe(pipe io.Reader, timeout time.Duration) (int, error) {
	// Delve outputs: "API server listening at: 127.0.0.1:PORT"
	portRegex := regexp.MustCompile(`API server listening at: .*:(\d+)`)

	portChan := make(chan int, 1)
	errChan := make(chan error, 1)

	go func() {
		reader := bufio.NewReader(pipe)
		for {
			line, err := reader.ReadString('\n')
			if err != nil {
				if err != io.EOF {
					errChan <- fmt.Errorf("error reading Delve output: %w", err)
				}
				return
			}

			// Check if we found the port
			matches := portRegex.FindStringSubmatch(line)
			if len(matches) >= 2 {
				port, err := strconv.Atoi(matches[1])
				if err != nil {
					errChan <- fmt.Errorf("invalid port number: %w", err)
					return
				}
				portChan <- port
				return
			}
		}
	}()

	select {
	case port := <-portChan:
		return port, nil
	case err := <-errChan:
		return 0, err
	case <-time.After(timeout):
		return 0, fmt.Errorf("timeout waiting for Delve to report listening port")
	}
}

// CreateBreakpoint creates a breakpoint at the specified location
func (dc *DelveClient) CreateBreakpoint(file string, line int, condition string) (int, error) {
	dc.mutex.Lock()
	defer dc.mutex.Unlock()

	if dc.client == nil {
		return 0, fmt.Errorf("not connected to Delve")
	}

	bp := &api.Breakpoint{
		File: file,
		Line: line,
	}

	if condition != "" {
		bp.Cond = condition
	}

	createdBP, err := dc.client.CreateBreakpoint(bp)
	if err != nil {
		return 0, fmt.Errorf("failed to create breakpoint: %w", err)
	}

	dc.breakpoints[createdBP.ID] = createdBP
	return createdBP.ID, nil
}

// ClearBreakpoint removes a breakpoint by ID
func (dc *DelveClient) ClearBreakpoint(id int) error {
	dc.mutex.Lock()
	defer dc.mutex.Unlock()

	if dc.client == nil {
		return fmt.Errorf("not connected to Delve")
	}

	bp, ok := dc.breakpoints[id]
	if !ok {
		return fmt.Errorf("breakpoint %d not found", id)
	}

	_, err := dc.client.ClearBreakpoint(bp.ID)
	if err != nil {
		return fmt.Errorf("failed to clear breakpoint: %w", err)
	}

	delete(dc.breakpoints, id)
	return nil
}

// Continue continues execution until next breakpoint
func (dc *DelveClient) Continue(threadID int) error {
	dc.mutex.Lock()
	defer dc.mutex.Unlock()

	if dc.client == nil {
		return fmt.Errorf("not connected to Delve")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	stateChan := dc.client.Continue()
	select {
	case state := <-stateChan:
		if state.Err != nil {
			return fmt.Errorf("continue failed: %w", state.Err)
		}
		return nil
	case <-ctx.Done():
		return fmt.Errorf("continue operation timed out")
	}
}

// Next steps over to the next line
func (dc *DelveClient) Next(threadID int) error {
	dc.mutex.Lock()
	defer dc.mutex.Unlock()

	if dc.client == nil {
		return fmt.Errorf("not connected to Delve")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	type result struct {
		state *api.DebuggerState
		err   error
	}
	resultChan := make(chan result, 1)

	go func() {
		state, err := dc.client.Next()
		resultChan <- result{state, err}
	}()

	select {
	case res := <-resultChan:
		if res.err != nil {
			return fmt.Errorf("next failed: %w", res.err)
		}
		if res.state.Err != nil {
			return fmt.Errorf("next failed: %w", res.state.Err)
		}
		return nil
	case <-ctx.Done():
		return fmt.Errorf("next operation timed out")
	}
}

// StepIn steps into a function call
func (dc *DelveClient) StepIn(threadID int) error {
	dc.mutex.Lock()
	defer dc.mutex.Unlock()

	if dc.client == nil {
		return fmt.Errorf("not connected to Delve")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	type result struct {
		state *api.DebuggerState
		err   error
	}
	resultChan := make(chan result, 1)

	go func() {
		state, err := dc.client.Step()
		resultChan <- result{state, err}
	}()

	select {
	case res := <-resultChan:
		if res.err != nil {
			return fmt.Errorf("step in failed: %w", res.err)
		}
		if res.state.Err != nil {
			return fmt.Errorf("step in failed: %w", res.state.Err)
		}
		return nil
	case <-ctx.Done():
		return fmt.Errorf("step in operation timed out")
	}
}

// StepOut steps out of the current function
func (dc *DelveClient) StepOut(threadID int) error {
	dc.mutex.Lock()
	defer dc.mutex.Unlock()

	if dc.client == nil {
		return fmt.Errorf("not connected to Delve")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	type result struct {
		state *api.DebuggerState
		err   error
	}
	resultChan := make(chan result, 1)

	go func() {
		state, err := dc.client.StepOut()
		resultChan <- result{state, err}
	}()

	select {
	case res := <-resultChan:
		if res.err != nil {
			return fmt.Errorf("step out failed: %w", res.err)
		}
		if res.state.Err != nil {
			return fmt.Errorf("step out failed: %w", res.state.Err)
		}
		return nil
	case <-ctx.Done():
		return fmt.Errorf("step out operation timed out")
	}
}

// ListThreads returns all threads in the debugged program
func (dc *DelveClient) ListThreads() ([]DelveThread, error) {
	dc.mutex.Lock()
	defer dc.mutex.Unlock()

	if dc.client == nil {
		return nil, fmt.Errorf("not connected to Delve")
	}

	threads, err := dc.client.ListThreads()
	if err != nil {
		return nil, fmt.Errorf("failed to list threads: %w", err)
	}

	result := make([]DelveThread, len(threads))
	for i, t := range threads {
		result[i] = DelveThread{
			ID:   t.ID,
			Name: fmt.Sprintf("Thread %d", t.ID),
		}
	}

	return result, nil
}

// StackTrace returns the call stack for a thread
func (dc *DelveClient) StackTrace(threadID int) ([]DelveStackFrame, error) {
	dc.mutex.Lock()
	defer dc.mutex.Unlock()

	if dc.client == nil {
		return nil, fmt.Errorf("not connected to Delve")
	}

	frames, err := dc.client.Stacktrace(-1, 50, api.StacktraceReadDefers, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get stack trace: %w", err)
	}

	result := make([]DelveStackFrame, len(frames))
	for i, f := range frames {
		result[i] = DelveStackFrame{
			ID:       i,
			Function: f.Function.Name(),
			File:     f.File,
			Line:     f.Line,
			Column:   0,
		}
	}

	return result, nil
}

// ListVariables returns variables for a given scope
func (dc *DelveClient) ListVariables(scopeRef int) ([]DelveVariable, error) {
	dc.mutex.Lock()
	defer dc.mutex.Unlock()

	if dc.client == nil {
		return nil, fmt.Errorf("not connected to Delve")
	}

	// Extract frame ID from scopeRef
	frameID := scopeRef / 1000

	// Get local variables for the frame
	scope := api.EvalScope{
		GoroutineID: -1,
		Frame:       frameID,
	}

	vars, err := dc.client.ListLocalVariables(scope, api.LoadConfig{
		FollowPointers:     true,
		MaxVariableRecurse: 1,
		MaxStringLen:       64,
		MaxArrayValues:     64,
		MaxStructFields:    -1,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list variables: %w", err)
	}

	result := make([]DelveVariable, len(vars))
	for i, v := range vars {
		result[i] = DelveVariable{
			Name:               v.Name,
			Value:              v.Value,
			Type:               v.Type,
			VariablesReference: 0, // TODO: handle nested variables
		}
	}

	return result, nil
}

// Evaluate evaluates an expression in the current context
func (dc *DelveClient) Evaluate(expression string, frameID int) (*EvaluateResult, error) {
	dc.mutex.Lock()
	defer dc.mutex.Unlock()

	if dc.client == nil {
		return nil, fmt.Errorf("not connected to Delve")
	}

	scope := api.EvalScope{
		GoroutineID: -1,
		Frame:       frameID,
	}

	v, err := dc.client.EvalVariable(scope, expression, api.LoadConfig{
		FollowPointers:     true,
		MaxVariableRecurse: 1,
		MaxStringLen:       64,
		MaxArrayValues:     64,
		MaxStructFields:    -1,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to evaluate expression: %w", err)
	}

	return &EvaluateResult{
		Value:              v.Value,
		Type:               v.Type,
		VariablesReference: 0, // TODO: handle nested variables
	}, nil
}

// Detach detaches from the debugged program
func (dc *DelveClient) Detach() error {
	dc.mutex.Lock()
	defer dc.mutex.Unlock()

	if dc.client != nil {
		if err := dc.client.Detach(true); err != nil {
			return fmt.Errorf("failed to detach: %w", err)
		}
		dc.client = nil
	}

	if dc.serverCmd != nil && dc.serverCmd.Process != nil {
		if err := dc.serverCmd.Process.Kill(); err != nil {
			return fmt.Errorf("failed to kill Delve process: %w", err)
		}

		// Wait for process to finish (non-blocking with timeout)
		done := make(chan error, 1)
		go func() {
			done <- dc.serverCmd.Wait()
		}()

		select {
		case <-done:
			// Process reaped successfully
		case <-time.After(5 * time.Second):
			log.Printf("Warning: Delve process did not exit cleanly")
		}

		dc.serverCmd = nil
	}

	return nil
}

// Close closes the Delve client
func (dc *DelveClient) Close() error {
	return dc.Detach()
}
