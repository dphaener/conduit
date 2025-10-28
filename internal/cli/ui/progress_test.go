package ui

import (
	"bytes"
	"strings"
	"testing"
	"time"
)

// TestSpinnerStartStop tests basic spinner lifecycle and goroutine cleanup
func TestSpinnerStartStop(t *testing.T) {
	var buf bytes.Buffer
	spinner := NewSpinner(&buf, SpinnerOptions{
		Message:  "Testing",
		NoColor:  true,
		Interval: 50 * time.Millisecond,
	})

	// Start the spinner
	spinner.Start()

	// Let it animate for a bit
	time.Sleep(150 * time.Millisecond)

	// Stop the spinner
	spinner.Stop()

	// Verify the spinner was active
	if !strings.Contains(buf.String(), "Testing") {
		t.Errorf("Expected spinner to show message 'Testing', got: %s", buf.String())
	}

	// Verify clearing sequence was written
	if !strings.Contains(buf.String(), "\r\033[K") {
		t.Error("Expected spinner to clear the line on stop")
	}
}

// TestSpinnerSuccess tests the Success method
func TestSpinnerSuccess(t *testing.T) {
	var buf bytes.Buffer
	spinner := NewSpinner(&buf, SpinnerOptions{
		Message: "Processing",
		NoColor: true,
	})

	spinner.Start()
	time.Sleep(50 * time.Millisecond)
	spinner.Success("Operation completed")

	output := buf.String()

	// Check for success symbol and message
	if !strings.Contains(output, "✓") {
		t.Error("Expected success symbol ✓")
	}
	if !strings.Contains(output, "Operation completed") {
		t.Errorf("Expected success message, got: %s", output)
	}
}

// TestSpinnerError tests the Error method
func TestSpinnerError(t *testing.T) {
	var buf bytes.Buffer
	spinner := NewSpinner(&buf, SpinnerOptions{
		Message: "Processing",
		NoColor: true,
	})

	spinner.Start()
	time.Sleep(50 * time.Millisecond)
	spinner.Error("Operation failed")

	output := buf.String()

	// Check for error symbol and message
	if !strings.Contains(output, "❌") {
		t.Error("Expected error symbol ❌")
	}
	if !strings.Contains(output, "Operation failed") {
		t.Errorf("Expected error message, got: %s", output)
	}
}

// TestSpinnerNoColor verifies NoColor flag disables colors
func TestSpinnerNoColor(t *testing.T) {
	var buf bytes.Buffer
	spinner := NewSpinner(&buf, SpinnerOptions{
		Message: "Testing",
		NoColor: true,
	})

	spinner.Start()
	time.Sleep(100 * time.Millisecond)
	spinner.Stop()

	output := buf.String()

	// With NoColor=true, there should be no ANSI color codes (except clear sequence)
	// ANSI color codes start with \x1b[ or \033[
	lines := strings.Split(output, "\n")
	for _, line := range lines {
		// Skip the clear line sequence which is expected
		if line == "\r\033[K" || line == "" {
			continue
		}
		// Check for color codes (like \x1b[36m for cyan)
		if strings.Contains(line, "\x1b[3") && !strings.Contains(line, "\x1b[K") {
			t.Errorf("Expected no color codes with NoColor=true, but found them in: %q", line)
		}
	}
}

// TestSpinnerUpdateMessage tests changing the spinner message
func TestSpinnerUpdateMessage(t *testing.T) {
	var buf bytes.Buffer
	spinner := NewSpinner(&buf, SpinnerOptions{
		Message: "Initial message",
		NoColor: true,
	})

	spinner.Start()
	time.Sleep(50 * time.Millisecond)

	spinner.UpdateMessage("Updated message")
	time.Sleep(50 * time.Millisecond)

	spinner.Stop()

	output := buf.String()

	// Should contain the updated message
	if !strings.Contains(output, "Updated message") {
		t.Errorf("Expected updated message in output, got: %s", output)
	}
}

// TestWithSpinner tests the helper function for success case
func TestWithSpinner(t *testing.T) {
	var buf bytes.Buffer
	called := false

	err := WithSpinner(&buf, "Processing task", true, func() error {
		called = true
		return nil
	})

	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}

	if !called {
		t.Error("Expected function to be called")
	}

	output := buf.String()
	if !strings.Contains(output, "✓") {
		t.Error("Expected success symbol in output")
	}
	if !strings.Contains(output, "Processing task") {
		t.Errorf("Expected task message in output, got: %s", output)
	}
}

// TestWithSpinnerError tests the helper function for error case
func TestWithSpinnerError(t *testing.T) {
	var buf bytes.Buffer
	testErr := &testError{msg: "test error"}

	err := WithSpinner(&buf, "Failing task", true, func() error {
		return testErr
	})

	if err != testErr {
		t.Errorf("Expected error to be returned, got: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "❌") {
		t.Error("Expected error symbol in output")
	}
	if !strings.Contains(output, "failed") {
		t.Errorf("Expected 'failed' in output, got: %s", output)
	}
}

// testError is a simple error type for testing
type testError struct {
	msg string
}

func (e *testError) Error() string {
	return e.msg
}

// TestProgressBarAdd tests incrementing progress
func TestProgressBarAdd(t *testing.T) {
	var buf bytes.Buffer
	bar := NewProgressBar(&buf, ProgressBarOptions{
		Total:   100,
		Width:   40,
		Message: "Loading",
		NoColor: true,
	})

	bar.Add(25)
	output := buf.String()

	// Should show 25%
	if !strings.Contains(output, "25%") {
		t.Errorf("Expected 25%% in output, got: %s", output)
	}

	buf.Reset()
	bar.Add(25)
	output = buf.String()

	// Should show 50%
	if !strings.Contains(output, "50%") {
		t.Errorf("Expected 50%% in output, got: %s", output)
	}
}

// TestProgressBarSet tests setting specific value
func TestProgressBarSet(t *testing.T) {
	var buf bytes.Buffer
	bar := NewProgressBar(&buf, ProgressBarOptions{
		Total:   100,
		Width:   40,
		NoColor: true,
	})

	bar.Set(75)
	output := buf.String()

	// Should show 75%
	if !strings.Contains(output, "75%") {
		t.Errorf("Expected 75%% in output, got: %s", output)
	}
}

// TestProgressBarFinish tests completion
func TestProgressBarFinish(t *testing.T) {
	var buf bytes.Buffer
	bar := NewProgressBar(&buf, ProgressBarOptions{
		Total:   100,
		Width:   40,
		NoColor: true,
	})

	bar.Set(50)
	buf.Reset() // Clear previous output

	bar.Finish()
	output := buf.String()

	// Should show 100%
	if !strings.Contains(output, "100%") {
		t.Errorf("Expected 100%% in output, got: %s", output)
	}

	// Should end with newline
	if !strings.HasSuffix(output, "\n") {
		t.Error("Expected output to end with newline")
	}
}

// TestProgressBarFinishWithMessage tests completion with success message
func TestProgressBarFinishWithMessage(t *testing.T) {
	var buf bytes.Buffer
	bar := NewProgressBar(&buf, ProgressBarOptions{
		Total:   100,
		Width:   40,
		NoColor: true,
	})

	bar.Set(50)
	bar.FinishWithMessage("Done!")

	output := buf.String()

	// Should show 100%
	if !strings.Contains(output, "100%") {
		t.Errorf("Expected 100%% in output, got: %s", output)
	}

	// Should show success message
	if !strings.Contains(output, "✓") {
		t.Error("Expected success symbol")
	}
	if !strings.Contains(output, "Done!") {
		t.Errorf("Expected 'Done!' in output, got: %s", output)
	}
}

// TestProgressBarRender tests output formatting
func TestProgressBarRender(t *testing.T) {
	var buf bytes.Buffer
	bar := NewProgressBar(&buf, ProgressBarOptions{
		Total:   100,
		Width:   20,
		Message: "Test",
		NoColor: true,
	})

	bar.Set(50)
	output := buf.String()

	// Should contain progress bar brackets
	if !strings.Contains(output, "[") || !strings.Contains(output, "]") {
		t.Errorf("Expected brackets in progress bar, got: %s", output)
	}

	// Should contain the message
	if !strings.Contains(output, "Test") {
		t.Errorf("Expected message 'Test' in output, got: %s", output)
	}

	// Should contain percentage
	if !strings.Contains(output, "50%") {
		t.Errorf("Expected 50%% in output, got: %s", output)
	}
}

// TestProgressBarNoColor verifies NoColor flag disables colors
func TestProgressBarNoColor(t *testing.T) {
	var buf bytes.Buffer
	bar := NewProgressBar(&buf, ProgressBarOptions{
		Total:   100,
		Width:   20,
		NoColor: true,
	})

	bar.Set(50)
	output := buf.String()

	// With NoColor=true, there should be no ANSI color codes
	// ANSI color codes start with \x1b[ or \033[
	if strings.Contains(output, "\x1b[3") {
		t.Errorf("Expected no color codes with NoColor=true, but found them in: %q", output)
	}
}

// TestProgressBarZeroTotal tests division by zero protection
func TestProgressBarZeroTotal(t *testing.T) {
	var buf bytes.Buffer
	bar := NewProgressBar(&buf, ProgressBarOptions{
		Total:   0,
		Width:   40,
		NoColor: true,
	})

	// Should not panic and should not render anything
	bar.Add(10)
	output := buf.String()

	if output != "" {
		t.Errorf("Expected no output with total=0, got: %s", output)
	}
}

// TestProgressBarCurrentExceedsTotal tests clamping behavior
func TestProgressBarCurrentExceedsTotal(t *testing.T) {
	var buf bytes.Buffer
	bar := NewProgressBar(&buf, ProgressBarOptions{
		Total:   100,
		Width:   40,
		NoColor: true,
	})

	// Try to set beyond total
	bar.Set(150)
	output := buf.String()

	// Should clamp to 100%
	if !strings.Contains(output, "100%") {
		t.Errorf("Expected 100%% when current exceeds total, got: %s", output)
	}

	buf.Reset()
	bar = NewProgressBar(&buf, ProgressBarOptions{
		Total:   100,
		Width:   40,
		NoColor: true,
	})

	// Try to add beyond total
	bar.Add(150)
	output = buf.String()

	// Should clamp to 100%
	if !strings.Contains(output, "100%") {
		t.Errorf("Expected 100%% when adding exceeds total, got: %s", output)
	}
}

// TestWithProgress tests the helper function
func TestWithProgress(t *testing.T) {
	var buf bytes.Buffer
	called := false

	err := WithProgress(&buf, "Processing items", 10, true, func(bar *ProgressBar) error {
		called = true
		for i := 0; i < 10; i++ {
			bar.Add(1)
		}
		return nil
	})

	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}

	if !called {
		t.Error("Expected function to be called")
	}

	output := buf.String()

	// Should show 100%
	if !strings.Contains(output, "100%") {
		t.Errorf("Expected 100%% in output, got: %s", output)
	}

	// Should show success symbol
	if !strings.Contains(output, "✓") {
		t.Error("Expected success symbol in output")
	}

	// Should show message
	if !strings.Contains(output, "Processing items") {
		t.Errorf("Expected message in output, got: %s", output)
	}
}

// TestWithProgressError tests the helper function with error
func TestWithProgressError(t *testing.T) {
	var buf bytes.Buffer
	testErr := &testError{msg: "progress error"}

	err := WithProgress(&buf, "Failing progress", 10, true, func(bar *ProgressBar) error {
		bar.Add(5)
		return testErr
	})

	if err != testErr {
		t.Errorf("Expected error to be returned, got: %v", err)
	}

	output := buf.String()

	// Should show 50% (where it stopped)
	if !strings.Contains(output, "50%") {
		t.Errorf("Expected 50%% in output, got: %s", output)
	}

	// Should NOT show success symbol (error path)
	if strings.Contains(output, "✓") {
		t.Error("Did not expect success symbol when error occurs")
	}
}

// TestSpinnerStopWithoutStart tests edge case of stopping before starting
func TestSpinnerStopWithoutStart(t *testing.T) {
	var buf bytes.Buffer
	spinner := NewSpinner(&buf, SpinnerOptions{
		Message: "Testing",
		NoColor: true,
	})

	// Stop without starting should not panic
	spinner.Stop()

	// No output expected
	if buf.Len() > 0 {
		t.Errorf("Expected no output when stopping inactive spinner, got: %s", buf.String())
	}
}

// TestSpinnerMultipleStops tests calling stop multiple times
func TestSpinnerMultipleStops(t *testing.T) {
	var buf bytes.Buffer
	spinner := NewSpinner(&buf, SpinnerOptions{
		Message: "Testing",
		NoColor: true,
	})

	spinner.Start()
	time.Sleep(50 * time.Millisecond)

	// First stop
	spinner.Stop()
	firstLen := buf.Len()

	// Second stop should be a no-op
	spinner.Stop()
	secondLen := buf.Len()

	if secondLen != firstLen {
		t.Error("Expected multiple stops to not produce additional output")
	}
}

// TestProgressBarDefaultWidth tests default width is set
func TestProgressBarDefaultWidth(t *testing.T) {
	var buf bytes.Buffer
	bar := NewProgressBar(&buf, ProgressBarOptions{
		Total:   100,
		NoColor: true,
		// Width not specified
	})

	if bar.width != 40 {
		t.Errorf("Expected default width of 40, got: %d", bar.width)
	}
}

// TestSpinnerDefaultInterval tests default interval is set
func TestSpinnerDefaultInterval(t *testing.T) {
	var buf bytes.Buffer
	spinner := NewSpinner(&buf, SpinnerOptions{
		Message: "Testing",
		NoColor: true,
		// Interval not specified
	})

	if spinner.interval != 100*time.Millisecond {
		t.Errorf("Expected default interval of 100ms, got: %v", spinner.interval)
	}
}
