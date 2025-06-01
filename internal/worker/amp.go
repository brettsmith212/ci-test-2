package worker

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// ampOperations implements the AmpOperations interface
type ampOperations struct {
	ampPath string
}

// NewAmpOperations creates a new Amp operations instance
func NewAmpOperations(ampPath string) AmpOperations {
	if ampPath == "" {
		// Try to find amp in PATH
		if path, err := exec.LookPath("amp"); err == nil {
			ampPath = path
		}
	}
	
	return &ampOperations{
		ampPath: ampPath,
	}
}

// CheckInstallation verifies that Amp is available and working
func (a *ampOperations) CheckInstallation() error {
	if a.ampPath == "" {
		return fmt.Errorf("amp CLI not found in PATH")
	}
	
	// Check if the binary exists
	if _, err := os.Stat(a.ampPath); os.IsNotExist(err) {
		return fmt.Errorf("amp binary not found at: %s", a.ampPath)
	}
	
	// Try to run amp --version to verify it works
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	
	cmd := exec.CommandContext(ctx, a.ampPath, "--version")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("amp CLI check failed: %w (output: %s)", err, string(output))
	}
	
	return nil
}

// ExecutePrompt runs an Amp prompt in the specified repository directory
func (a *ampOperations) ExecutePrompt(ctx context.Context, repoDir, prompt string) (*AmpResult, error) {
	result := &AmpResult{
		Success: false,
	}
	
	// Verify amp is available
	if err := a.CheckInstallation(); err != nil {
		result.Error = err
		return result, err
	}
	
	// Change to repository directory
	originalDir, err := os.Getwd()
	if err != nil {
		result.Error = fmt.Errorf("failed to get current directory: %w", err)
		return result, err
	}
	
	if err := os.Chdir(repoDir); err != nil {
		result.Error = fmt.Errorf("failed to change to repo directory: %w", err)
		return result, err
	}
	
	defer func() {
		// Always restore original directory
		os.Chdir(originalDir)
	}()
	
	// Prepare amp command
	// Using a context with timeout to prevent hanging
	ampCtx, cancel := context.WithTimeout(ctx, 30*time.Minute)
	defer cancel()
	
	// Run amp with the prompt piped to stdin
	cmd := exec.CommandContext(ampCtx, a.ampPath)
	fmt.Printf("DEBUG AMP: Running amp from directory: %s\n", repoDir)
	fmt.Printf("DEBUG AMP: Amp path: %s\n", a.ampPath)
	fmt.Printf("DEBUG AMP: Prompt: %s\n", prompt)
	
	// Set up environment for amp
	cmd.Env = append(os.Environ(),
		"TERM=xterm-256color", // Ensure proper terminal support
	)
	
	// Pipe the prompt to amp's stdin
	cmd.Stdin = strings.NewReader(prompt)
	
	fmt.Printf("DEBUG AMP: Starting amp execution...\n")
	// Capture output
	output, err := cmd.CombinedOutput()
	fmt.Printf("DEBUG AMP: Amp finished. Output length: %d bytes\n", len(output))
	if err != nil {
		fmt.Printf("DEBUG AMP: Amp error: %v\n", err)
	}
	result.Output = string(output)
	
	if err != nil {
		result.Error = fmt.Errorf("amp command failed: %w", err)
		result.Message = fmt.Sprintf("Amp execution failed: %s", string(output))
		return result, err
	}
	
	// Parse the output to determine success and extract information
	if err := a.parseAmpOutput(result, string(output)); err != nil {
		result.Error = err
		return result, err
	}
	
	// Check for actual file changes
	changedFiles, err := a.detectChangedFiles(repoDir)
	if err != nil {
		result.Error = fmt.Errorf("failed to detect changed files: %w", err)
		return result, err
	}
	
	result.FilesChanged = changedFiles
	
	// Consider it successful if there are file changes or Amp indicated success
	if len(changedFiles) > 0 || result.Success {
		result.Success = true
		if result.Message == "" {
			result.Message = fmt.Sprintf("Amp completed successfully, %d files changed", len(changedFiles))
		}
	} else {
		result.Success = false
		if result.Message == "" {
			result.Message = "Amp completed but no files were changed"
		}
	}
	
	return result, nil
}

// parseAmpOutput analyzes Amp's output to determine success and extract information
func (a *ampOperations) parseAmpOutput(result *AmpResult, output string) error {
	lines := strings.Split(output, "\n")
	
	// Look for success/error indicators in Amp output
	for _, line := range lines {
		line = strings.TrimSpace(line)
		
		// Common success indicators
		if strings.Contains(strings.ToLower(line), "completed successfully") ||
		   strings.Contains(strings.ToLower(line), "task completed") ||
		   strings.Contains(strings.ToLower(line), "changes applied") {
			result.Success = true
			if result.Message == "" {
				result.Message = line
			}
		}
		
		// Common error indicators
		if strings.Contains(strings.ToLower(line), "error:") ||
		   strings.Contains(strings.ToLower(line), "failed:") ||
		   strings.Contains(strings.ToLower(line), "could not") {
			result.Success = false
			if result.Message == "" {
				result.Message = line
			}
		}
	}
	
	return nil
}

// detectChangedFiles uses git to detect what files have been modified
func (a *ampOperations) detectChangedFiles(repoDir string) ([]string, error) {
	// Use git status to detect changed files
	cmd := exec.Command("git", "status", "--porcelain")
	cmd.Dir = repoDir
	
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("git status failed: %w", err)
	}
	
	var changedFiles []string
	lines := strings.Split(string(output), "\n")
	
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		
		// Parse git status output format
		if len(line) > 3 {
			filename := strings.TrimSpace(line[2:])
			changedFiles = append(changedFiles, filename)
		}
	}
	
	return changedFiles, nil
}

// runAmpWithArgs executes amp with the given arguments
func (a *ampOperations) runAmpWithArgs(ctx context.Context, repoDir string, args []string) (string, error) {
	cmd := exec.CommandContext(ctx, a.ampPath, args...)
	cmd.Dir = repoDir
	
	// Set up environment
	cmd.Env = append(os.Environ(),
		"TERM=xterm-256color",
	)
	
	output, err := cmd.CombinedOutput()
	return string(output), err
}

// GetAmpVersion returns the version of the Amp CLI
func (a *ampOperations) GetAmpVersion(ctx context.Context) (string, error) {
	output, err := a.runAmpWithArgs(ctx, "", []string{"--version"})
	if err != nil {
		return "", fmt.Errorf("failed to get amp version: %w", err)
	}
	
	return strings.TrimSpace(output), nil
}

// ValidateRepository checks if the repository is suitable for Amp processing
func (a *ampOperations) ValidateRepository(ctx context.Context, repoDir string) error {
	// Check if it's a git repository
	gitDir := filepath.Join(repoDir, ".git")
	if _, err := os.Stat(gitDir); os.IsNotExist(err) {
		return fmt.Errorf("not a git repository: %s", repoDir)
	}
	
	// Check if there are any files to work with
	entries, err := os.ReadDir(repoDir)
	if err != nil {
		return fmt.Errorf("failed to read repository directory: %w", err)
	}
	
	// Count non-git files
	fileCount := 0
	for _, entry := range entries {
		if !entry.IsDir() && entry.Name() != ".git" {
			fileCount++
		}
	}
	
	if fileCount == 0 {
		return fmt.Errorf("repository appears to be empty")
	}
	
	return nil
}
