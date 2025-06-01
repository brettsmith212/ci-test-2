package output

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
	"time"
)

// Spinner represents a spinning progress indicator
type Spinner struct {
	frames   []string
	interval time.Duration
	message  string
	writer   io.Writer
	mu       sync.Mutex
	active   bool
	done     chan struct{}
}

// Common spinner styles
var (
	SpinnerDots = []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}
	SpinnerBars = []string{"▁", "▃", "▄", "▅", "▆", "▇", "█", "▇", "▆", "▅", "▄", "▃"}
	SpinnerArrow = []string{"←", "↖", "↑", "↗", "→", "↘", "↓", "↙"}
	SpinnerSimple = []string{"|", "/", "-", "\\"}
	SpinnerClock = []string{"🕐", "🕑", "🕒", "🕓", "🕔", "🕕", "🕖", "🕗", "🕘", "🕙", "🕚", "🕛"}
)

// NewSpinner creates a new spinner with the given message
func NewSpinner(message string) *Spinner {
	frames := SpinnerDots
	if !IsColorEnabled() {
		frames = SpinnerSimple
	}

	return &Spinner{
		frames:   frames,
		interval: 100 * time.Millisecond,
		message:  message,
		writer:   os.Stderr,
		done:     make(chan struct{}),
	}
}

// NewSpinnerWithStyle creates a spinner with a specific style
func NewSpinnerWithStyle(message string, frames []string) *Spinner {
	return &Spinner{
		frames:   frames,
		interval: 100 * time.Millisecond,
		message:  message,
		writer:   os.Stderr,
		done:     make(chan struct{}),
	}
}

// SetWriter sets the output writer for the spinner
func (s *Spinner) SetWriter(w io.Writer) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.writer = w
}

// SetMessage updates the spinner message
func (s *Spinner) SetMessage(message string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.message = message
}

// Start begins the spinner animation
func (s *Spinner) Start() {
	s.mu.Lock()
	if s.active {
		s.mu.Unlock()
		return
	}
	s.active = true
	s.mu.Unlock()

	go s.run()
}

// Stop stops the spinner and optionally shows a final message
func (s *Spinner) Stop() {
	s.mu.Lock()
	if !s.active {
		s.mu.Unlock()
		return
	}
	s.active = false
	s.mu.Unlock()

	close(s.done)
	s.clearLine()
}

// Success stops the spinner and shows a success message
func (s *Spinner) Success(message string) {
	s.Stop()
	fmt.Fprintf(s.writer, "%s %s\n", Success("✓"), message)
}

// Error stops the spinner and shows an error message
func (s *Spinner) Error(message string) {
	s.Stop()
	fmt.Fprintf(s.writer, "%s %s\n", Error("✗"), message)
}

// Warning stops the spinner and shows a warning message
func (s *Spinner) Warning(message string) {
	s.Stop()
	fmt.Fprintf(s.writer, "%s %s\n", Warning("⚠"), message)
}

// Info stops the spinner and shows an info message
func (s *Spinner) Info(message string) {
	s.Stop()
	fmt.Fprintf(s.writer, "%s %s\n", Info("ℹ"), message)
}

func (s *Spinner) run() {
	frameIndex := 0
	ticker := time.NewTicker(s.interval)
	defer ticker.Stop()

	for {
		select {
		case <-s.done:
			return
		case <-ticker.C:
			s.mu.Lock()
			if !s.active {
				s.mu.Unlock()
				return
			}

			frame := s.frames[frameIndex%len(s.frames)]
			message := s.message

			if IsColorEnabled() {
				fmt.Fprintf(s.writer, "\r%s %s", Primary(frame), message)
			} else {
				fmt.Fprintf(s.writer, "\r%s %s", frame, message)
			}

			frameIndex++
			s.mu.Unlock()
		}
	}
}

func (s *Spinner) clearLine() {
	fmt.Fprintf(s.writer, "\r%s\r", strings.Repeat(" ", 80))
}

// ProgressBar represents a progress bar
type ProgressBar struct {
	total    int64
	current  int64
	width    int
	message  string
	writer   io.Writer
	mu       sync.Mutex
	showRate bool
	startTime time.Time
}

// NewProgressBar creates a new progress bar
func NewProgressBar(total int64, message string) *ProgressBar {
	return &ProgressBar{
		total:     total,
		current:   0,
		width:     50,
		message:   message,
		writer:    os.Stderr,
		showRate:  true,
		startTime: time.Now(),
	}
}

// SetWriter sets the output writer for the progress bar
func (pb *ProgressBar) SetWriter(w io.Writer) {
	pb.mu.Lock()
	defer pb.mu.Unlock()
	pb.writer = w
}

// Update updates the progress bar with the current value
func (pb *ProgressBar) Update(current int64) {
	pb.mu.Lock()
	defer pb.mu.Unlock()
	
	pb.current = current
	if pb.current > pb.total {
		pb.current = pb.total
	}
	
	pb.render()
}

// Increment increments the progress bar by one
func (pb *ProgressBar) Increment() {
	pb.Update(pb.current + 1)
}

// Finish completes the progress bar
func (pb *ProgressBar) Finish() {
	pb.Update(pb.total)
	fmt.Fprintln(pb.writer)
}

func (pb *ProgressBar) render() {
	percentage := float64(pb.current) / float64(pb.total) * 100
	filled := int(float64(pb.width) * float64(pb.current) / float64(pb.total))
	
	bar := strings.Repeat("█", filled) + strings.Repeat("░", pb.width-filled)
	
	if IsColorEnabled() {
		bar = Primary(strings.Repeat("█", filled)) + Muted(strings.Repeat("░", pb.width-filled))
	}
	
	status := fmt.Sprintf("[%s] %.1f%% (%d/%d)", bar, percentage, pb.current, pb.total)
	
	if pb.message != "" {
		status = pb.message + " " + status
	}
	
	if pb.showRate && pb.current > 0 {
		elapsed := time.Since(pb.startTime)
		rate := float64(pb.current) / elapsed.Seconds()
		remaining := time.Duration(float64(pb.total-pb.current)/rate) * time.Second
		
		status += fmt.Sprintf(" [%s remaining]", Muted(remaining.Round(time.Second).String()))
	}
	
	fmt.Fprintf(pb.writer, "\r%s", status)
}

// TaskStatus represents the status of a long-running task
type TaskStatus struct {
	spinner *Spinner
	steps   []string
	current int
	writer  io.Writer
}

// NewTaskStatus creates a new task status tracker
func NewTaskStatus(taskName string, steps []string) *TaskStatus {
	return &TaskStatus{
		spinner: NewSpinner(fmt.Sprintf("%s...", taskName)),
		steps:   steps,
		current: 0,
		writer:  os.Stderr,
	}
}

// Start begins tracking the task
func (ts *TaskStatus) Start() {
	if len(ts.steps) > 0 {
		ts.spinner.SetMessage(fmt.Sprintf("Step 1/%d: %s", len(ts.steps), ts.steps[0]))
	}
	ts.spinner.SetWriter(ts.writer)
	ts.spinner.Start()
}

// NextStep moves to the next step
func (ts *TaskStatus) NextStep() {
	ts.current++
	if ts.current < len(ts.steps) {
		message := fmt.Sprintf("Step %d/%d: %s", ts.current+1, len(ts.steps), ts.steps[ts.current])
		ts.spinner.SetMessage(message)
	}
}

// UpdateStep updates the current step message
func (ts *TaskStatus) UpdateStep(message string) {
	if ts.current < len(ts.steps) {
		fullMessage := fmt.Sprintf("Step %d/%d: %s", ts.current+1, len(ts.steps), message)
		ts.spinner.SetMessage(fullMessage)
	} else {
		ts.spinner.SetMessage(message)
	}
}

// Success completes the task with success
func (ts *TaskStatus) Success(message string) {
	ts.spinner.Success(message)
}

// Error completes the task with error
func (ts *TaskStatus) Error(message string) {
	ts.spinner.Error(message)
}

// Warning completes the task with warning
func (ts *TaskStatus) Warning(message string) {
	ts.spinner.Warning(message)
}

// SetWriter sets the output writer
func (ts *TaskStatus) SetWriter(w io.Writer) {
	ts.writer = w
	ts.spinner.SetWriter(w)
}

// Utility functions for common progress patterns

// WithSpinner runs a function with a spinner
func WithSpinner(message string, fn func() error) error {
	spinner := NewSpinner(message)
	spinner.Start()
	
	err := fn()
	if err != nil {
		spinner.Error(fmt.Sprintf("Failed: %v", err))
		return err
	}
	
	spinner.Success("Done")
	return nil
}

// WithSpinnerContext runs a function with a spinner and context support
func WithSpinnerContext(ctx context.Context, message string, fn func(context.Context) error) error {
	spinner := NewSpinner(message)
	spinner.Start()
	
	done := make(chan error, 1)
	go func() {
		done <- fn(ctx)
	}()
	
	select {
	case err := <-done:
		if err != nil {
			spinner.Error(fmt.Sprintf("Failed: %v", err))
			return err
		}
		spinner.Success("Done")
		return nil
	case <-ctx.Done():
		spinner.Warning("Cancelled")
		return ctx.Err()
	}
}

// ShowProgress shows a simple progress indicator for a slice of items
func ShowProgress[T any](items []T, message string, fn func(T) error) error {
	pb := NewProgressBar(int64(len(items)), message)
	
	for i, item := range items {
		if err := fn(item); err != nil {
			pb.Finish()
			return err
		}
		pb.Update(int64(i + 1))
	}
	
	pb.Finish()
	return nil
}
