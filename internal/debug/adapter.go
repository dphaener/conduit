package debug

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"strings"
	"sync"

	"github.com/google/go-dap"
)

// DebugAdapter implements the Debug Adapter Protocol server for Conduit
type DebugAdapter struct {
	conn       io.ReadWriteCloser
	delve      *DelveClient
	sourceMaps *SourceMapRegistry
	session    *DebugSession
	seq        int
	seqMutex   sync.Mutex
	codec      *dap.Codec
}

// DebugSession tracks the current debugging session state
type DebugSession struct {
	SessionID   string
	Breakpoints map[string][]*Breakpoint
	Variables   map[int]*Variable
	Threads     map[int]*Thread
	mutex       sync.RWMutex
}

// Thread represents a thread in the debugged program
type Thread struct {
	ID   int
	Name string
}

// Variable represents a variable in the debugged program
type Variable struct {
	Name               string
	Value              string
	Type               string
	VariablesReference int
}

// LaunchArguments represents launch configuration
type LaunchArguments struct {
	Program string `json:"program"`
}

// NewDebugAdapter creates a new DAP adapter instance
func NewDebugAdapter(conn io.ReadWriteCloser, delvePath string, sourceMaps *SourceMapRegistry) (*DebugAdapter, error) {
	delveClient, err := NewDelveClient(delvePath)
	if err != nil {
		return nil, fmt.Errorf("failed to create Delve client: %w", err)
	}

	return &DebugAdapter{
		conn:       conn,
		delve:      delveClient,
		sourceMaps: sourceMaps,
		session: &DebugSession{
			SessionID:   "conduit-debug-session",
			Breakpoints: make(map[string][]*Breakpoint),
			Variables:   make(map[int]*Variable),
			Threads:     make(map[int]*Thread),
		},
		seq:   0, // Let nextSeq() return 1 on first call
		codec: dap.NewCodec(),
	}, nil
}

// Start begins processing DAP messages
func (da *DebugAdapter) Start() error {
	reader := bufio.NewReader(da.conn)

	for {
		msg, err := dap.ReadProtocolMessage(reader)
		if err != nil {
			if err == io.EOF {
				return nil
			}
			return fmt.Errorf("failed to read message: %w", err)
		}

		if err := da.handleMessage(msg); err != nil {
			// Check if error is fatal (protocol errors, connection errors, etc.)
			if isFatalError(err) {
				log.Printf("Fatal error handling message: %v", err)
				return fmt.Errorf("fatal protocol error: %w", err)
			}
			log.Printf("Error handling message (recoverable): %v", err)
		}
	}
}

// isFatalError determines if an error should terminate the debug session
func isFatalError(err error) bool {
	if err == nil {
		return false
	}

	// EOF and connection errors are fatal
	if errors.Is(err, io.EOF) {
		return true
	}

	errStr := err.Error()
	// Protocol violations and connection errors are fatal
	if strings.Contains(errStr, "protocol error") ||
		strings.Contains(errStr, "connection closed") ||
		strings.Contains(errStr, "broken pipe") ||
		strings.Contains(errStr, "connection reset") {
		return true
	}

	return false
}

// handleMessage routes DAP messages to appropriate handlers
func (da *DebugAdapter) handleMessage(msg dap.Message) error {
	switch m := msg.(type) {
	case *dap.InitializeRequest:
		return da.handleInitialize(m)
	case *dap.LaunchRequest:
		return da.handleLaunch(m)
	case *dap.SetBreakpointsRequest:
		return da.handleSetBreakpoints(m)
	case *dap.ContinueRequest:
		return da.handleContinue(m)
	case *dap.NextRequest:
		return da.handleNext(m)
	case *dap.StepInRequest:
		return da.handleStepIn(m)
	case *dap.StepOutRequest:
		return da.handleStepOut(m)
	case *dap.ThreadsRequest:
		return da.handleThreads(m)
	case *dap.StackTraceRequest:
		return da.handleStackTrace(m)
	case *dap.ScopesRequest:
		return da.handleScopes(m)
	case *dap.VariablesRequest:
		return da.handleVariables(m)
	case *dap.EvaluateRequest:
		return da.handleEvaluate(m)
	case *dap.DisconnectRequest:
		return da.handleDisconnect(m)
	default:
		log.Printf("Unsupported message type: %T", msg)
		return nil
	}
}

// handleInitialize processes the initialize request
func (da *DebugAdapter) handleInitialize(request *dap.InitializeRequest) error {
	response := &dap.InitializeResponse{
		Response: dap.Response{
			ProtocolMessage: dap.ProtocolMessage{
				Seq:  da.nextSeq(),
				Type: "response",
			},
			RequestSeq: request.Seq,
			Success:    true,
			Command:    request.Command,
		},
		Body: dap.Capabilities{
			SupportsConfigurationDoneRequest:   true,
			SupportsEvaluateForHovers:          true,
			SupportsConditionalBreakpoints:     true,
			SupportsHitConditionalBreakpoints:  false,
			SupportsFunctionBreakpoints:        false,
			SupportsDelayedStackTraceLoading:   false,
			SupportsLogPoints:                  false,
			SupportsSetVariable:                false,
			SupportsRestartFrame:               false,
			SupportsGotoTargetsRequest:         false,
			SupportsStepInTargetsRequest:       false,
			SupportsCompletionsRequest:         false,
		},
	}

	if err := dap.WriteProtocolMessage(da.conn, response); err != nil {
		return err
	}

	// Send initialized event
	event := &dap.InitializedEvent{
		Event: dap.Event{
			ProtocolMessage: dap.ProtocolMessage{
				Seq:  da.nextSeq(),
				Type: "event",
			},
			Event: "initialized",
		},
	}

	return dap.WriteProtocolMessage(da.conn, event)
}

// handleLaunch processes the launch request
func (da *DebugAdapter) handleLaunch(request *dap.LaunchRequest) error {
	var args LaunchArguments
	if err := json.Unmarshal(request.Arguments, &args); err != nil {
		return da.sendErrorResponse(request.Seq, fmt.Sprintf("Invalid launch arguments: %v", err))
	}

	// Start the Delve debugger with the program
	launchArgs := map[string]interface{}{
		"program": args.Program,
	}

	if err := da.delve.LaunchWithArgs(launchArgs); err != nil {
		return da.sendErrorResponse(request.Seq, fmt.Sprintf("Failed to launch: %v", err))
	}

	response := &dap.LaunchResponse{
		Response: dap.Response{
			ProtocolMessage: dap.ProtocolMessage{
				Seq:  da.nextSeq(),
				Type: "response",
			},
			RequestSeq: request.Seq,
			Success:    true,
			Command:    request.Command,
		},
	}

	return dap.WriteProtocolMessage(da.conn, response)
}

// handleSetBreakpoints processes the setBreakpoints request
func (da *DebugAdapter) handleSetBreakpoints(request *dap.SetBreakpointsRequest) error {
	args := request.Arguments
	sourceFile := args.Source.Path

	// Build all breakpoints first (without lock)
	newBreakpoints := make([]*Breakpoint, 0, len(args.Breakpoints))
	dapBreakpoints := make([]dap.Breakpoint, 0, len(args.Breakpoints))

	for _, bp := range args.Breakpoints {
		// Translate source breakpoint to generated code
		translated, err := da.sourceMaps.TranslateBreakpoint(sourceFile, bp.Line)
		if err != nil {
			log.Printf("Failed to translate breakpoint: %v", err)
			dapBreakpoints = append(dapBreakpoints, dap.Breakpoint{
				Verified: false,
				Line:     bp.Line,
				Message:  fmt.Sprintf("Failed to map breakpoint: %v", err),
			})
			continue
		}

		// Set breakpoint in Delve
		delveID, err := da.delve.CreateBreakpoint(translated.GeneratedFile, translated.GeneratedLine, bp.Condition)
		if err != nil {
			log.Printf("Failed to set breakpoint in Delve: %v", err)
			dapBreakpoints = append(dapBreakpoints, dap.Breakpoint{
				Verified: false,
				Line:     bp.Line,
				Message:  fmt.Sprintf("Failed to set breakpoint: %v", err),
			})
			continue
		}

		translated.ID = delveID
		translated.Verified = true
		translated.Condition = bp.Condition

		newBreakpoints = append(newBreakpoints, translated)
		dapBreakpoints = append(dapBreakpoints, dap.Breakpoint{
			Id:       delveID,
			Verified: true,
			Line:     bp.Line,
		})
	}

	// Atomically replace all breakpoints for this file
	da.session.mutex.Lock()
	da.session.Breakpoints[sourceFile] = newBreakpoints
	da.session.mutex.Unlock()

	response := &dap.SetBreakpointsResponse{
		Response: dap.Response{
			ProtocolMessage: dap.ProtocolMessage{
				Seq:  da.nextSeq(),
				Type: "response",
			},
			RequestSeq: request.Seq,
			Success:    true,
			Command:    request.Command,
		},
		Body: dap.SetBreakpointsResponseBody{
			Breakpoints: dapBreakpoints,
		},
	}

	return dap.WriteProtocolMessage(da.conn, response)
}

// handleContinue processes the continue request
func (da *DebugAdapter) handleContinue(request *dap.ContinueRequest) error {
	if err := da.delve.Continue(request.Arguments.ThreadId); err != nil {
		return da.sendErrorResponse(request.Seq, fmt.Sprintf("Continue failed: %v", err))
	}

	response := &dap.ContinueResponse{
		Response: dap.Response{
			ProtocolMessage: dap.ProtocolMessage{
				Seq:  da.nextSeq(),
				Type: "response",
			},
			RequestSeq: request.Seq,
			Success:    true,
			Command:    request.Command,
		},
		Body: dap.ContinueResponseBody{
			AllThreadsContinued: true,
		},
	}

	return dap.WriteProtocolMessage(da.conn, response)
}

// handleNext processes the next (step over) request
func (da *DebugAdapter) handleNext(request *dap.NextRequest) error {
	if err := da.delve.Next(request.Arguments.ThreadId); err != nil {
		return da.sendErrorResponse(request.Seq, fmt.Sprintf("Next failed: %v", err))
	}

	response := &dap.NextResponse{
		Response: dap.Response{
			ProtocolMessage: dap.ProtocolMessage{
				Seq:  da.nextSeq(),
				Type: "response",
			},
			RequestSeq: request.Seq,
			Success:    true,
			Command:    request.Command,
		},
	}

	return dap.WriteProtocolMessage(da.conn, response)
}

// handleStepIn processes the stepIn request
func (da *DebugAdapter) handleStepIn(request *dap.StepInRequest) error {
	if err := da.delve.StepIn(request.Arguments.ThreadId); err != nil {
		return da.sendErrorResponse(request.Seq, fmt.Sprintf("Step in failed: %v", err))
	}

	response := &dap.StepInResponse{
		Response: dap.Response{
			ProtocolMessage: dap.ProtocolMessage{
				Seq:  da.nextSeq(),
				Type: "response",
			},
			RequestSeq: request.Seq,
			Success:    true,
			Command:    request.Command,
		},
	}

	return dap.WriteProtocolMessage(da.conn, response)
}

// handleStepOut processes the stepOut request
func (da *DebugAdapter) handleStepOut(request *dap.StepOutRequest) error {
	if err := da.delve.StepOut(request.Arguments.ThreadId); err != nil {
		return da.sendErrorResponse(request.Seq, fmt.Sprintf("Step out failed: %v", err))
	}

	response := &dap.StepOutResponse{
		Response: dap.Response{
			ProtocolMessage: dap.ProtocolMessage{
				Seq:  da.nextSeq(),
				Type: "response",
			},
			RequestSeq: request.Seq,
			Success:    true,
			Command:    request.Command,
		},
	}

	return dap.WriteProtocolMessage(da.conn, response)
}

// handleThreads processes the threads request
func (da *DebugAdapter) handleThreads(request *dap.ThreadsRequest) error {
	threads, err := da.delve.ListThreads()
	if err != nil {
		return da.sendErrorResponse(request.Seq, fmt.Sprintf("Failed to list threads: %v", err))
	}

	dapThreads := make([]dap.Thread, len(threads))
	for i, t := range threads {
		dapThreads[i] = dap.Thread{
			Id:   t.ID,
			Name: t.Name,
		}
	}

	response := &dap.ThreadsResponse{
		Response: dap.Response{
			ProtocolMessage: dap.ProtocolMessage{
				Seq:  da.nextSeq(),
				Type: "response",
			},
			RequestSeq: request.Seq,
			Success:    true,
			Command:    request.Command,
		},
		Body: dap.ThreadsResponseBody{
			Threads: dapThreads,
		},
	}

	return dap.WriteProtocolMessage(da.conn, response)
}

// handleStackTrace processes the stackTrace request
func (da *DebugAdapter) handleStackTrace(request *dap.StackTraceRequest) error {
	delveStack, err := da.delve.StackTrace(request.Arguments.ThreadId)
	if err != nil {
		return da.sendErrorResponse(request.Seq, fmt.Sprintf("Failed to get stack trace: %v", err))
	}

	frames := make([]dap.StackFrame, 0, len(delveStack))
	for i, df := range delveStack {
		// Translate generated location to source
		source, line, col, err := da.sourceMaps.TranslateLocation(df.File, df.Line)
		if err != nil {
			// Can't map - show generated code
			frames = append(frames, dap.StackFrame{
				Id:   i,
				Name: df.Function,
				Source: &dap.Source{
					Path: df.File,
				},
				Line:   df.Line,
				Column: df.Column,
			})
			continue
		}

		frames = append(frames, dap.StackFrame{
			Id:   i,
			Name: df.Function,
			Source: &dap.Source{
				Path: source,
			},
			Line:   line,
			Column: col,
		})
	}

	response := &dap.StackTraceResponse{
		Response: dap.Response{
			ProtocolMessage: dap.ProtocolMessage{
				Seq:  da.nextSeq(),
				Type: "response",
			},
			RequestSeq: request.Seq,
			Success:    true,
			Command:    request.Command,
		},
		Body: dap.StackTraceResponseBody{
			StackFrames: frames,
			TotalFrames: len(frames),
		},
	}

	return dap.WriteProtocolMessage(da.conn, response)
}

// handleScopes processes the scopes request
func (da *DebugAdapter) handleScopes(request *dap.ScopesRequest) error {
	scopes := []dap.Scope{
		{
			Name:               "Local",
			VariablesReference: request.Arguments.FrameId*1000 + 1,
			Expensive:          false,
		},
	}

	response := &dap.ScopesResponse{
		Response: dap.Response{
			ProtocolMessage: dap.ProtocolMessage{
				Seq:  da.nextSeq(),
				Type: "response",
			},
			RequestSeq: request.Seq,
			Success:    true,
			Command:    request.Command,
		},
		Body: dap.ScopesResponseBody{
			Scopes: scopes,
		},
	}

	return dap.WriteProtocolMessage(da.conn, response)
}

// handleVariables processes the variables request
func (da *DebugAdapter) handleVariables(request *dap.VariablesRequest) error {
	delveVars, err := da.delve.ListVariables(request.Arguments.VariablesReference)
	if err != nil {
		return da.sendErrorResponse(request.Seq, fmt.Sprintf("Failed to list variables: %v", err))
	}

	variables := make([]dap.Variable, len(delveVars))
	for i, dv := range delveVars {
		variables[i] = dap.Variable{
			Name:               dv.Name,
			Value:              dv.Value,
			Type:               dv.Type,
			VariablesReference: dv.VariablesReference,
		}
	}

	response := &dap.VariablesResponse{
		Response: dap.Response{
			ProtocolMessage: dap.ProtocolMessage{
				Seq:  da.nextSeq(),
				Type: "response",
			},
			RequestSeq: request.Seq,
			Success:    true,
			Command:    request.Command,
		},
		Body: dap.VariablesResponseBody{
			Variables: variables,
		},
	}

	return dap.WriteProtocolMessage(da.conn, response)
}

// handleEvaluate processes the evaluate request
func (da *DebugAdapter) handleEvaluate(request *dap.EvaluateRequest) error {
	result, err := da.delve.Evaluate(request.Arguments.Expression, request.Arguments.FrameId)
	if err != nil {
		return da.sendErrorResponse(request.Seq, fmt.Sprintf("Evaluation failed: %v", err))
	}

	response := &dap.EvaluateResponse{
		Response: dap.Response{
			ProtocolMessage: dap.ProtocolMessage{
				Seq:  da.nextSeq(),
				Type: "response",
			},
			RequestSeq: request.Seq,
			Success:    true,
			Command:    request.Command,
		},
		Body: dap.EvaluateResponseBody{
			Result:             result.Value,
			Type:               result.Type,
			VariablesReference: result.VariablesReference,
		},
	}

	return dap.WriteProtocolMessage(da.conn, response)
}

// handleDisconnect processes the disconnect request
func (da *DebugAdapter) handleDisconnect(request *dap.DisconnectRequest) error {
	if err := da.delve.Detach(); err != nil {
		log.Printf("Error detaching from Delve: %v", err)
	}

	response := &dap.DisconnectResponse{
		Response: dap.Response{
			ProtocolMessage: dap.ProtocolMessage{
				Seq:  da.nextSeq(),
				Type: "response",
			},
			RequestSeq: request.Seq,
			Success:    true,
			Command:    request.Command,
		},
	}

	return dap.WriteProtocolMessage(da.conn, response)
}

// sendErrorResponse sends an error response
func (da *DebugAdapter) sendErrorResponse(requestSeq int, message string) error {
	response := &dap.ErrorResponse{
		Response: dap.Response{
			ProtocolMessage: dap.ProtocolMessage{
				Seq:  da.nextSeq(),
				Type: "response",
			},
			RequestSeq: requestSeq,
			Success:    false,
			Message:    message,
		},
	}

	return dap.WriteProtocolMessage(da.conn, response)
}

// nextSeq returns the next sequence number
func (da *DebugAdapter) nextSeq() int {
	da.seqMutex.Lock()
	defer da.seqMutex.Unlock()
	da.seq++
	return da.seq
}

// Close closes the debug adapter
func (da *DebugAdapter) Close() error {
	if da.delve != nil {
		da.delve.Close()
	}
	if da.conn != nil {
		return da.conn.Close()
	}
	return nil
}

// Server manages the DAP server lifecycle
type Server struct {
	listener    net.Listener
	sourceMaps  *SourceMapRegistry
	delvePath   string
	wg          sync.WaitGroup
	shutdown    chan struct{}
	activeConns map[net.Conn]struct{}
	connMutex   sync.Mutex
}

// NewServer creates a new DAP server instance
func NewServer(addr string, sourceMaps *SourceMapRegistry, delvePath string) (*Server, error) {
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, fmt.Errorf("failed to listen: %w", err)
	}

	return &Server{
		listener:    listener,
		sourceMaps:  sourceMaps,
		delvePath:   delvePath,
		shutdown:    make(chan struct{}),
		activeConns: make(map[net.Conn]struct{}),
	}, nil
}

// Serve starts accepting connections
func (s *Server) Serve() error {
	defer s.listener.Close()

	log.Printf("DAP server listening on %s", s.listener.Addr().String())

	for {
		select {
		case <-s.shutdown:
			return nil
		default:
		}

		conn, err := s.listener.Accept()
		if err != nil {
			select {
			case <-s.shutdown:
				return nil
			default:
				log.Printf("Failed to accept connection: %v", err)
				continue
			}
		}

		s.connMutex.Lock()
		s.activeConns[conn] = struct{}{}
		s.connMutex.Unlock()

		s.wg.Add(1)
		go s.handleConnection(conn)
	}
}

// handleConnection handles a single DAP connection
func (s *Server) handleConnection(conn net.Conn) {
	defer s.wg.Done()
	defer conn.Close()

	defer func() {
		s.connMutex.Lock()
		delete(s.activeConns, conn)
		s.connMutex.Unlock()
	}()

	adapter, err := NewDebugAdapter(conn, s.delvePath, s.sourceMaps)
	if err != nil {
		log.Printf("Failed to create adapter: %v", err)
		return
	}
	defer adapter.Close()

	if err := adapter.Start(); err != nil {
		log.Printf("Adapter error: %v", err)
	}
}

// Shutdown gracefully shuts down the server
func (s *Server) Shutdown() error {
	close(s.shutdown)
	s.listener.Close()

	s.connMutex.Lock()
	for conn := range s.activeConns {
		conn.Close()
	}
	s.connMutex.Unlock()

	s.wg.Wait()
	return nil
}

// Serve starts the DAP server on the specified address (legacy wrapper)
func Serve(addr string, sourceMaps *SourceMapRegistry, delvePath string) error {
	server, err := NewServer(addr, sourceMaps, delvePath)
	if err != nil {
		return err
	}
	return server.Serve()
}
