package ui

import (
	"fmt"
	"io"
	"strings"
	"sync"
	"time"

	"github.com/fatih/color"
)

// Spinner represents a simple text-based spinner for indeterminate operations
type Spinner struct {
	writer   io.Writer
	message  string
	frames   []string
	interval time.Duration
	active   bool
	done     chan bool
	noColor  bool
	mu       sync.RWMutex // Protects message field
}

// SpinnerOptions configures spinner behavior
type SpinnerOptions struct {
	Message  string
	NoColor  bool
	Interval time.Duration // Default: 100ms
}

var defaultFrames = []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}

// NewSpinner creates a new spinner
func NewSpinner(w io.Writer, opts SpinnerOptions) *Spinner {
	interval := opts.Interval
	if interval == 0 {
		interval = 100 * time.Millisecond
	}

	return &Spinner{
		writer:   w,
		message:  opts.Message,
		frames:   defaultFrames,
		interval: interval,
		done:     make(chan bool),
		noColor:  opts.NoColor,
	}
}

// Start begins the spinner animation
func (s *Spinner) Start() {
	s.active = true
	go s.animate()
}

// Stop stops the spinner and clears the line
func (s *Spinner) Stop() {
	if !s.active {
		return
	}
	s.active = false
	s.done <- true
	// Clear the line
	fmt.Fprint(s.writer, "\r\033[K")
}

// Success stops the spinner and shows a success message
func (s *Spinner) Success(message string) {
	s.Stop()
	green := color.New(color.FgGreen, color.Bold)
	if s.noColor {
		green.DisableColor()
	}
	green.Fprintf(s.writer, "✓ %s\n", message)
}

// Error stops the spinner and shows an error message
func (s *Spinner) Error(message string) {
	s.Stop()
	red := color.New(color.FgRed, color.Bold)
	if s.noColor {
		red.DisableColor()
	}
	red.Fprintf(s.writer, "❌ %s\n", message)
}

// UpdateMessage changes the spinner message
func (s *Spinner) UpdateMessage(message string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.message = message
}

func (s *Spinner) animate() {
	frameIndex := 0
	ticker := time.NewTicker(s.interval)
	defer ticker.Stop()

	cyan := color.New(color.FgCyan)
	if s.noColor {
		cyan.DisableColor()
	}

	for {
		select {
		case <-s.done:
			return
		case <-ticker.C:
			frame := s.frames[frameIndex]
			s.mu.RLock()
			msg := s.message
			s.mu.RUnlock()
			cyan.Fprintf(s.writer, "\r%s %s", frame, msg)
			frameIndex = (frameIndex + 1) % len(s.frames)
		}
	}
}

// ProgressBar represents a simple progress bar for determinate operations
type ProgressBar struct {
	writer  io.Writer
	total   int
	current int
	width   int
	message string
	noColor bool
}

// ProgressBarOptions configures progress bar behavior
type ProgressBarOptions struct {
	Total   int
	Width   int    // Default: 40
	Message string
	NoColor bool
}

// NewProgressBar creates a new progress bar
func NewProgressBar(w io.Writer, opts ProgressBarOptions) *ProgressBar {
	width := opts.Width
	if width == 0 {
		width = 40
	}

	return &ProgressBar{
		writer:  w,
		total:   opts.Total,
		current: 0,
		width:   width,
		message: opts.Message,
		noColor: opts.NoColor,
	}
}

// Add increments the progress by the given amount
func (p *ProgressBar) Add(n int) {
	p.current += n
	if p.current > p.total {
		p.current = p.total
	}
	p.render()
}

// Set sets the current progress to the given value
func (p *ProgressBar) Set(n int) {
	p.current = n
	if p.current > p.total {
		p.current = p.total
	}
	p.render()
}

// Finish completes the progress bar
func (p *ProgressBar) Finish() {
	p.current = p.total
	p.render()
	fmt.Fprintln(p.writer)
}

// FinishWithMessage completes the progress bar with a success message
func (p *ProgressBar) FinishWithMessage(message string) {
	p.Finish()
	green := color.New(color.FgGreen, color.Bold)
	if p.noColor {
		green.DisableColor()
	}
	green.Fprintf(p.writer, "✓ %s\n", message)
}

func (p *ProgressBar) render() {
	if p.total == 0 {
		return
	}

	percent := float64(p.current) / float64(p.total)
	filledWidth := int(float64(p.width) * percent)

	cyan := color.New(color.FgCyan)
	gray := color.New(color.FgHiBlack)
	if p.noColor {
		cyan.DisableColor()
		gray.DisableColor()
	}

	// Build the progress bar
	var bar strings.Builder
	bar.WriteString("[")

	// Filled portion
	cyan.Fprint(&bar, strings.Repeat("█", filledWidth))

	// Empty portion
	emptyWidth := p.width - filledWidth
	gray.Fprint(&bar, strings.Repeat("░", emptyWidth))

	bar.WriteString("]")

	// Format percentage
	percentStr := fmt.Sprintf("%3d%%", int(percent*100))

	// Format message
	message := ""
	if p.message != "" {
		message = " " + p.message
	}

	// Print the line
	fmt.Fprintf(p.writer, "\r%s %s%s", bar.String(), percentStr, message)
}

// Simple convenience functions for common operations

// WithSpinner runs a function with a spinner indicator
func WithSpinner(w io.Writer, message string, noColor bool, fn func() error) error {
	spinner := NewSpinner(w, SpinnerOptions{
		Message: message,
		NoColor: noColor,
	})
	spinner.Start()
	defer spinner.Stop()

	err := fn()
	if err != nil {
		spinner.Error(fmt.Sprintf("%s failed", message))
		return err
	}

	spinner.Success(message)
	return nil
}

// WithProgress runs a function with a progress bar
func WithProgress(w io.Writer, message string, total int, noColor bool, fn func(*ProgressBar) error) error {
	bar := NewProgressBar(w, ProgressBarOptions{
		Total:   total,
		Message: message,
		NoColor: noColor,
	})

	err := fn(bar)
	if err != nil {
		fmt.Fprintln(w)
		return err
	}

	bar.FinishWithMessage(message)
	return nil
}
