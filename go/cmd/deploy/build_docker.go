package deploy

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/unkeyed/unkey/go/pkg/git"
)

const (
	// Timeouts
	DockerBuildTimeout = 10 * time.Minute

	// Build arguments
	VersionBuildArg = "VERSION"

	// Progress messages
	ProgressBuilding = "Building..."

	// Limits
	MaxOutputLines = 5
	MaxErrorLines  = 3
)

var (
	ErrDockerNotFound     = errors.New("docker command not found - please install Docker")
	ErrDockerBuildFailed  = errors.New("docker build failed")
	ErrInvalidContext     = errors.New("invalid build context")
	ErrDockerfileNotFound = errors.New("dockerfile not found")
	ErrBuildTimeout       = errors.New("docker build timed out")
)

// sanitizeDockerTag sanitizes a string to be valid for Docker tags
// Official Docker tag grammar: /[\w][\w.-]{0,127}/
// - First char: word character (a-zA-Z0-9_)
// - Remaining chars: word characters, periods, or dashes
// - Maximum 128 characters total
func sanitizeDockerTag(input string) string {
	if input == "" {
		return "main"
	}

	// Convert to lowercase (Docker registries are case-insensitive)
	result := strings.ToLower(input)

	// Replace invalid characters with dashes
	// Keep only: a-z, A-Z, 0-9, _, ., -
	invalidChars := regexp.MustCompile(`[^a-zA-Z0-9._-]+`)
	result = invalidChars.ReplaceAllString(result, "-")

	// Ensure first character is a word character (a-zA-Z0-9_)
	// If it starts with . or -, prepend with a valid character
	if len(result) > 0 && !regexp.MustCompile(`^[a-zA-Z0-9_]`).MatchString(result) {
		result = "v" + result
	}

	// Remove consecutive dashes for cleaner tags
	multiDash := regexp.MustCompile(`-{2,}`)
	result = multiDash.ReplaceAllString(result, "-")

	// Limit to 64 characters (leaving room for SHA suffix in the full image tag)
	if len(result) > 64 {
		result = result[:64]
		// Ensure it doesn't end with dash after truncation
		result = strings.TrimRight(result, "-")
	}

	// Final safety check - ensure we have a valid tag
	if result == "" || !regexp.MustCompile(`^[a-zA-Z0-9_][a-zA-Z0-9._-]*$`).MatchString(result) {
		return "main"
	}

	return result
}

// generateImageTag creates a unique tag for the Docker image
func generateImageTag(opts DeployOptions, gitInfo git.Info) string {
	// Sanitize branch name for Docker tag compatibility
	cleanBranch := sanitizeDockerTag(opts.Branch)

	if gitInfo.ShortSHA != "" {
		return fmt.Sprintf("%s-%s", cleanBranch, gitInfo.ShortSHA)
	}
	return fmt.Sprintf("%s-%d", cleanBranch, time.Now().Unix())
}

// isDockerAvailable checks if Docker is installed and accessible
func isDockerAvailable() error {
	cmd := exec.Command("docker", "--version")
	if err := cmd.Run(); err != nil {
		return ErrDockerNotFound
	}
	return nil
}

// buildImage builds the Docker image with proper error hierarchy
func buildImage(ctx context.Context, opts DeployOptions, dockerImage string, ui *UI) error {
	// Sub-step 1: Validate inputs
	if err := validateImagePath(opts); err != nil {
		ui.PrintStepError("Validation failed")
		ui.PrintErrorDetails(err.Error())
		return err
	}
	ui.PrintStepSuccess("Build inputs validated")

	// Sub-step 2: Prepare build command
	buildArgs := []string{"build"}
	if opts.Dockerfile != DefaultDockerfile {
		buildArgs = append(buildArgs, "-f", opts.Dockerfile)
	}
	buildArgs = append(buildArgs,
		"-t", dockerImage,
		"--build-arg", fmt.Sprintf("%s=%s", VersionBuildArg, opts.Commit),
		opts.Context,
	)

	ui.PrintStepSuccess("Build command prepared")
	if opts.Verbose {
		ui.PrintBuildProgress(fmt.Sprintf("Running: docker %s", strings.Join(buildArgs, " ")))
	}

	// Sub-step 3: Execute Docker build
	ui.PrintStepSuccess("Starting Docker build")
	buildCtx, cancel := context.WithTimeout(ctx, DockerBuildTimeout)
	defer cancel()

	cmd := exec.CommandContext(buildCtx, "docker", buildArgs...)

	var buildErr error
	if opts.Verbose {
		buildErr = buildImageVerbose(cmd, buildCtx, ui)
	} else {
		buildErr = buildImageWithSpinner(cmd, buildCtx, ui)
	}

	if buildErr != nil {
		ui.PrintStepError("Docker build failed")
		ui.PrintErrorDetails(buildErr.Error())
		return buildErr
	}

	return nil
}

// buildImageVerbose shows all Docker output in real-time
func buildImageVerbose(cmd *exec.Cmd, buildCtx context.Context, ui *UI) error {
	// Set up pipes for real-time output
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("failed to create stdout pipe: %w", err)
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("failed to create stderr pipe: %w", err)
	}

	// Start the command
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start docker build: %w", err)
	}

	// Stream stdout
	go func() {
		scanner := bufio.NewScanner(stdout)
		for scanner.Scan() {
			ui.PrintBuildProgress(scanner.Text())
		}
	}()

	// Stream stderr
	go func() {
		scanner := bufio.NewScanner(stderr)
		for scanner.Scan() {
			ui.PrintBuildError(scanner.Text())
		}
	}()

	// Wait for completion
	done := make(chan error, 1)
	go func() {
		done <- cmd.Wait()
	}()

	select {
	case <-buildCtx.Done():
		if buildCtx.Err() == context.DeadlineExceeded {
			return fmt.Errorf("%w after %v", ErrBuildTimeout, DockerBuildTimeout)
		}
		return buildCtx.Err()
	case err := <-done:
		if err != nil {
			return fmt.Errorf("%w: %v", ErrDockerBuildFailed, err)
		}
	}

	return nil
}

// buildImageWithSpinner shows progress spinner with current step
func buildImageWithSpinner(cmd *exec.Cmd, buildCtx context.Context, ui *UI) error {
	// Set up pipes to capture output for progress tracking
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("failed to create stdout pipe: %w", err)
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("failed to create stderr pipe: %w", err)
	}

	// Start the command
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start docker build: %w", err)
	}

	// Track all output for error reporting
	var outputMu sync.Mutex
	allOutput := []string{}

	// Start progress spinner
	ui.StartStepSpinner(ProgressBuilding)

	// Stream stdout and track progress
	go func() {
		scanner := bufio.NewScanner(stdout)
		for scanner.Scan() {
			line := scanner.Text()

			outputMu.Lock()
			allOutput = append(allOutput, line)
			outputMu.Unlock()

			// Update spinner with current step
			if step := extractDockerStep(line); step != "" {
				ui.UpdateStepSpinner(fmt.Sprintf("%s %s", ProgressBuilding, step))
			}
		}
	}()

	// Stream stderr
	go func() {
		scanner := bufio.NewScanner(stderr)
		for scanner.Scan() {
			line := scanner.Text()

			outputMu.Lock()
			allOutput = append(allOutput, line)
			outputMu.Unlock()
		}
	}()

	// Wait for completion
	done := make(chan error, 1)
	go func() {
		done <- cmd.Wait()
	}()

	select {
	case <-buildCtx.Done():
		ui.CompleteCurrentStep("Build timed out", false)
		if buildCtx.Err() == context.DeadlineExceeded {
			return fmt.Errorf("%w after %v", ErrBuildTimeout, DockerBuildTimeout)
		}
		return buildCtx.Err()
	case err := <-done:
		if err != nil {
			ui.CompleteCurrentStep("Build failed", false)

			outputMu.Lock()
			outputCopy := make([]string, len(allOutput))
			copy(outputCopy, allOutput)
			outputMu.Unlock()

			// Show last few lines of output for debugging
			if len(outputCopy) > 0 {
				ui.PrintBuildError("Last few lines of output:")
				lines := outputCopy
				if len(lines) > MaxOutputLines {
					lines = lines[len(lines)-MaxOutputLines:]
				}
				for _, line := range lines {
					ui.PrintBuildProgress(line)
				}
			}
			return fmt.Errorf("%w: %s", ErrDockerBuildFailed, classifyError(strings.Join(outputCopy, "\n")))
		}
	}

	ui.CompleteCurrentStep("Docker build completed successfully", true)
	return nil
}

// extractDockerStep extracts meaningful step info from Docker output
func extractDockerStep(line string) string {
	line = strings.TrimSpace(line)

	// Look for Docker build steps
	if strings.HasPrefix(line, "#") && strings.Contains(line, "[") {
		// Extract step like "#5 [builder 1/6] FROM docker.io/library/golang"
		if idx := strings.Index(line, "]"); idx > 0 {
			step := line[:idx+1]
			// Clean up the step display
			step = strings.ReplaceAll(step, "#", "Step ")
			return step
		}
	}

	// Look for other meaningful progress
	switch {
	case strings.Contains(line, "DONE"):
		return "Step completed"
	case strings.Contains(line, "CACHED"):
		return "Using cache"
	case strings.Contains(line, "exporting"):
		return "Exporting image"
	case strings.Contains(line, "naming to"):
		return "Tagging image"
	}

	return ""
}

// pushImage pushes the built Docker image to the registry
func pushImage(ctx context.Context, dockerImage, registry string) error {
	cmd := exec.CommandContext(ctx, "docker", "push", dockerImage)
	output, err := cmd.CombinedOutput()
	if err != nil {
		detailedMsg := classifyPushError(string(output), registry)
		return fmt.Errorf("%s: %w", detailedMsg, err)
	}

	// Show push output
	if len(output) > 0 {
		fmt.Printf("%s\n", string(output))
	}

	return nil
}

// validateImagePath - pre-flight checks using error constants
func validateImagePath(opts DeployOptions) error {
	// Context directory exists?
	if _, err := os.Stat(opts.Context); os.IsNotExist(err) {
		return fmt.Errorf("%w: directory '%s' does not exist", ErrInvalidContext, opts.Context)
	}

	// Dockerfile exists?
	dockerfilePath := opts.Dockerfile
	if !filepath.IsAbs(dockerfilePath) {
		dockerfilePath = filepath.Join(opts.Context, opts.Dockerfile)
	}

	if _, err := os.Stat(dockerfilePath); os.IsNotExist(err) {
		return fmt.Errorf("%w: '%s' not found", ErrDockerfileNotFound, dockerfilePath)
	}

	return nil
}

// classifyError provides helpful error messages
func classifyError(output string) string {
	output = strings.ToLower(output)

	switch {
	case strings.Contains(output, "dockerfile"):
		return "Dockerfile issue - check path and syntax"
	case strings.Contains(output, "no such file"):
		return "File not found - check COPY/ADD paths"
	case strings.Contains(output, "permission denied"):
		return "Permission denied - check file permissions"
	case strings.Contains(output, "network") || strings.Contains(output, "timeout"):
		return "Network error - check internet connection"
	case strings.Contains(output, "manifest unknown"):
		return "Base image not found - check FROM instruction"
	case strings.Contains(output, "space") || strings.Contains(output, "disk"):
		return "Insufficient disk space"
	default:
		// Return last few lines for debugging
		lines := strings.Split(strings.TrimSpace(output), "\n")
		if len(lines) > MaxErrorLines {
			return strings.Join(lines[len(lines)-MaxErrorLines:], "\n")
		}
		return strings.TrimSpace(output)
	}
}

// classifyPushError provides detailed error messages for push failures
func classifyPushError(output, registry string) string {
	output = strings.TrimSpace(output)
	registryHost := getRegistryHost(registry)

	switch {
	case strings.Contains(output, "denied"):
		return fmt.Sprintf("registry access denied. try: docker login %s", registryHost)
	case strings.Contains(output, "not found") || strings.Contains(output, "404"):
		return "registry not found. create repository or use --registry=your-registry/your-app"
	case strings.Contains(output, "unauthorized"):
		return fmt.Sprintf("authentication required. run: docker login %s", registryHost)
	default:
		return output
	}
}

// getRegistryHost extracts the registry hostname from a full registry path
func getRegistryHost(registry string) string {
	parts := strings.Split(registry, "/")
	if len(parts) > 0 {
		return parts[0]
	}
	return DefaultRegistry
}

// UI methods for build progress
func (ui *UI) PrintBuildProgress(message string) {
	ui.mu.Lock()
	defer ui.mu.Unlock()
	fmt.Printf("    %s\n", message)
}

func (ui *UI) PrintBuildError(message string) {
	ui.mu.Lock()
	defer ui.mu.Unlock()
	fmt.Printf("    %s⚠%s %s\n", ColorYellow, ColorReset, message)
}

// UpdateStepSpinner updates the current step spinner message
func (ui *UI) UpdateStepSpinner(message string) {
	ui.mu.Lock()
	defer ui.mu.Unlock()
	if ui.stepSpinning {
		ui.currentStep = message
	}
}
