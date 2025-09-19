//go:build linux
// +build linux

package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"regexp"
	"strings"
	"syscall"
	"time"
)

// AIDEV-NOTE: This init wrapper is designed to be the PID 1 process in a microvm
// It handles:
// - Environment variable setup from kernel cmdline
// - Working directory changes
// - Signal forwarding to the actual process
// - Zombie process reaping
// - Proper exit code propagation

// Version information (set by build flags)
var (
	version   = "dev"
	buildTime = "unknown"
)

// AIDEV-BUSINESS_RULE: Security constants for safe operation
const (
	maxJSONSize    = 1024 * 1024 // 1MB limit for JSON files
	maxEnvKeyLen   = 256         // Maximum environment variable key length
	maxEnvValueLen = 4096        // Maximum environment variable value length
)

// AIDEV-BUSINESS_RULE: Valid environment variable name pattern
var validEnvKeyPattern = regexp.MustCompile(`^[A-Z][A-Z0-9_]*$`)

func main() {
	// Set up logging to stderr (stdout might be used by the child process)
	log.SetOutput(os.Stderr)
	log.SetPrefix("[init] ")

	// AIDEV-NOTE: Write debug file with secure permissions
	os.WriteFile("/init.started", []byte(fmt.Sprintf("Started at %s\n", time.Now())), 0o600)

	// AIDEV-NOTE: Mount /proc filesystem so we can read kernel command line
	if err := syscall.Mount("proc", "/proc", "proc", 0, ""); err != nil {
		log.Printf("warning: failed to mount /proc: %v", err)
		// Continue anyway - we have fallback logic
	}

	// Parse command line arguments
	if len(os.Args) < 2 {
		// No args provided, try to read command file
		if _, err := os.Stat("/container.cmd"); err == nil {
			// Add a dummy arg so we don't exit
			os.Args = append(os.Args, "dummy")
		} else {
			log.Fatal("usage: metald-init [--version] [--help] -- command [args...]")
		}
	}

	// Handle special flags
	if os.Args[1] == "--version" {
		fmt.Printf("metald-init version %s (built %s)\n", version, buildTime)
		os.Exit(0)
	}

	if os.Args[1] == "--help" {
		printHelp()
		os.Exit(0)
	}

	// Find the command separator
	cmdStart := -1
	for i, arg := range os.Args[1:] {
		if arg == "--" {
			cmdStart = i + 2 // +1 for skipping os.Args[0], +1 for the "--" itself
			break
		}
	}

	var command string
	var commandArgs []string

	if cmdStart == -1 || cmdStart >= len(os.Args) {
		// AIDEV-BUSINESS_RULE: Add size limits for JSON parsing to prevent memory exhaustion
		// No command on command line, try to read from container.cmd file
		cmdData, err := readFileSafely("/container.cmd", maxJSONSize)
		if err != nil {
			log.Fatal("no command specified after '--' and no /container.cmd file found")
		}

		var fullCmd []string
		if err := json.Unmarshal(cmdData, &fullCmd); err != nil {
			log.Fatalf("failed to parse /container.cmd: %v", err)
		}

		if len(fullCmd) == 0 {
			log.Fatal("empty command in /container.cmd")
		}

		command = fullCmd[0]
		if len(fullCmd) > 1 {
			commandArgs = fullCmd[1:]
		}
		log.Printf("loaded command from /container.cmd: %s %v", command, commandArgs)
	} else {
		// Extract the command and its arguments from command line
		command = os.Args[cmdStart]
		commandArgs = os.Args[cmdStart+1:]
	}

	log.Printf("preparing to execute: %s %v", command, commandArgs)

	// AIDEV-NOTE: Write debug info with secure permissions
	debugInfo := fmt.Sprintf("Command: %s\nArgs: %v\nEnv count: %d\nWorking dir: %s\n",
		command, commandArgs, len(os.Environ()), os.Getenv("PWD"))
	os.WriteFile("/init.command", []byte(debugInfo), 0o600)

	// AIDEV-NOTE: Load container environment configuration for complete runtime replication
	containerEnv, err := loadContainerEnvironment()
	if err != nil {
		log.Printf("warning: failed to load container environment: %v", err)
		// Continue with default environment - this is not fatal
	}

	// Parse kernel command line for our parameters
	kernelParams := parseKernelCmdline()

	// AIDEV-NOTE: Apply container environment first for complete runtime replication
	if err := applyContainerEnvironment(containerEnv); err != nil {
		log.Fatalf("critical: failed to apply container environment: %v", err)
	}

	// AIDEV-BUSINESS_RULE: Critical failures should be fatal, not warnings
	// Set up environment variables (kernel params can override container env)
	if err := setupEnvironment(kernelParams); err != nil {
		log.Fatalf("critical: failed to setup environment: %v", err)
	}

	// Change working directory if specified (kernel params can override container workdir)
	if err := changeWorkingDirectory(kernelParams); err != nil {
		log.Fatalf("critical: failed to change working directory: %v", err)
	}

	// Create common directories that containers expect
	createCommonDirectories()

	// Set up signal handling
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGTERM, syscall.SIGINT)

	// Start the command
	cmd := exec.Command(command, commandArgs...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	// Set up process attributes for proper signal handling
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Setpgid: true,
		Pgid:    0,
	}

	if err := cmd.Start(); err != nil {
		log.Fatalf("failed to start command: %v", err)
	}

	log.Printf("started process with PID %d", cmd.Process.Pid)

	// Handle signals and zombie reaping in a goroutine
	go handleSignalsAndReaping(cmd, sigChan)

	// Wait for the command to finish
	err = cmd.Wait()

	// Extract exit code
	exitCode := 0
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			if status, ok := exitErr.Sys().(syscall.WaitStatus); ok {
				exitCode = status.ExitStatus()
			}
		} else {
			log.Printf("command failed: %v", err)
			exitCode = 1
		}
	}

	log.Printf("command exited with code %d", exitCode)
	os.Exit(exitCode)
}

// parseKernelCmdline reads and parses /proc/cmdline
func parseKernelCmdline() map[string]string {
	params := make(map[string]string)

	cmdline, err := os.ReadFile("/proc/cmdline")
	if err != nil {
		log.Printf("warning: failed to read /proc/cmdline: %v", err)
		return params
	}

	// Parse space-separated key=value pairs
	for param := range strings.FieldsSeq(string(cmdline)) {
		if strings.Contains(param, "=") {
			parts := strings.SplitN(param, "=", 2)
			params[parts[0]] = parts[1]
		}
	}

	return params
}

// AIDEV-BUSINESS_RULE: Secure file reading with size limits
func readFileSafely(path string, maxSize int64) ([]byte, error) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, fmt.Errorf("failed to stat file: %w", err)
	}

	if info.Size() > maxSize {
		return nil, fmt.Errorf("file size %d exceeds maximum allowed size %d", info.Size(), maxSize)
	}

	return os.ReadFile(path)
}

// AIDEV-BUSINESS_RULE: Validate metadata file paths to prevent path traversal
func validateMetadataPath(path string) error {
	// Only allow absolute paths under specific safe directories
	if !filepath.IsAbs(path) {
		return fmt.Errorf("metadata path must be absolute")
	}

	// Clean the path to remove any .. components
	cleanPath := filepath.Clean(path)
	if cleanPath != path {
		return fmt.Errorf("metadata path contains invalid components")
	}

	// Whitelist allowed directories
	allowedPrefixes := []string{
		"/metadata/",
		"/var/metadata/",
		"/tmp/metadata/",
	}

	for _, prefix := range allowedPrefixes {
		if strings.HasPrefix(cleanPath, prefix) {
			return nil
		}
	}

	return fmt.Errorf("metadata path %s is not in an allowed directory", cleanPath)
}

// AIDEV-BUSINESS_RULE: Validate and sanitize environment variable names and values
func validateEnvVar(key, value string) error {
	if len(key) == 0 {
		return fmt.Errorf("environment variable key cannot be empty")
	}

	if len(key) > maxEnvKeyLen {
		return fmt.Errorf("environment variable key %s exceeds maximum length %d", key, maxEnvKeyLen)
	}

	if len(value) > maxEnvValueLen {
		return fmt.Errorf("environment variable value for %s exceeds maximum length %d", key, maxEnvValueLen)
	}

	// Validate key format (uppercase letters, numbers, underscores only)
	if !validEnvKeyPattern.MatchString(key) {
		return fmt.Errorf("environment variable key %s contains invalid characters (must match %s)", key, validEnvKeyPattern.String())
	}

	// Check for dangerous environment variables
	dangerousVars := []string{"LD_PRELOAD", "LD_LIBRARY_PATH", "DYLD_INSERT_LIBRARIES"}
	for _, dangerous := range dangerousVars {
		if key == dangerous {
			return fmt.Errorf("environment variable %s is not allowed for security reasons", key)
		}
	}

	return nil
}

// setupEnvironment sets up environment variables from kernel parameters and metadata file
func setupEnvironment(params map[string]string) error {
	// AIDEV-BUSINESS_RULE: Validate metadata path to prevent path traversal
	// First, check if there's a metadata file specified
	if metadataPath, ok := params["metadata"]; ok {
		if err := validateMetadataPath(metadataPath); err != nil {
			return fmt.Errorf("invalid metadata path: %w", err)
		}

		if err := loadEnvironmentFromMetadata(metadataPath); err != nil {
			return fmt.Errorf("failed to load metadata from %s: %w", metadataPath, err)
		}
	}

	// AIDEV-BUSINESS_RULE: Validate and sanitize environment variables from kernel cmdline
	// Then apply env.KEY=VALUE parameters from kernel cmdline (these override metadata)
	for key, value := range params {
		if strings.HasPrefix(key, "env.") {
			envKey := strings.TrimPrefix(key, "env.")

			if err := validateEnvVar(envKey, value); err != nil {
				return fmt.Errorf("invalid environment variable from cmdline: %w", err)
			}

			if err := os.Setenv(envKey, value); err != nil {
				return fmt.Errorf("failed to set %s=%s: %w", envKey, value, err)
			}
			log.Printf("set environment from cmdline: %s=%s", envKey, value)
		}
	}

	// AIDEV-NOTE: Ensure PATH is set with a reasonable default if not provided
	if os.Getenv("PATH") == "" {
		defaultPath := "/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin"
		if err := os.Setenv("PATH", defaultPath); err != nil {
			return fmt.Errorf("failed to set default PATH: %w", err)
		}
		log.Printf("set default PATH (no PATH provided): %s", defaultPath)
	}

	return nil
}

// loadEnvironmentFromMetadata loads environment variables from a metadata JSON file
func loadEnvironmentFromMetadata(path string) error {
	// AIDEV-BUSINESS_RULE: Use safe file reading with size limits
	data, err := readFileSafely(path, maxJSONSize)
	if err != nil {
		return fmt.Errorf("failed to read metadata file: %w", err)
	}

	// Parse metadata JSON (compatible with builderd's ImageMetadata)
	var metadata struct {
		Env        map[string]string `json:"env"`
		WorkingDir string            `json:"working_dir"`
		Entrypoint []string          `json:"entrypoint"`
		Command    []string          `json:"command"`
	}

	if err := json.Unmarshal(data, &metadata); err != nil {
		return fmt.Errorf("failed to parse metadata: %w", err)
	}

	// AIDEV-BUSINESS_RULE: Validate environment variables from metadata
	// Set environment variables from metadata
	for key, value := range metadata.Env {
		// Skip PATH from metadata to avoid conflicts
		if key == "PATH" {
			continue
		}

		if err := validateEnvVar(key, value); err != nil {
			log.Printf("warning: skipping invalid environment variable from metadata: %v", err)
			continue
		}

		if err := os.Setenv(key, value); err != nil {
			return fmt.Errorf("failed to set %s=%s from metadata: %w", key, value, err)
		}
		log.Printf("set environment from metadata: %s=%s", key, value)
	}

	return nil
}

// AIDEV-BUSINESS_RULE: Validate working directory path for security
func validateWorkingDirectory(path string) error {
	if !filepath.IsAbs(path) {
		return fmt.Errorf("working directory must be absolute path")
	}

	// Clean the path to remove any .. components
	cleanPath := filepath.Clean(path)
	if cleanPath != path {
		return fmt.Errorf("working directory contains invalid path components")
	}

	return nil
}

// changeWorkingDirectory changes to the specified working directory
func changeWorkingDirectory(params map[string]string) error {
	// AIDEV-BUSINESS_RULE: Implement complete working directory change with metadata support
	var targetWorkdir string

	// First check kernel cmdline parameter
	if workdir, ok := params["workdir"]; ok {
		targetWorkdir = workdir
	} else if metadataPath, ok := params["metadata"]; ok {
		// Try to get working directory from metadata
		if err := validateMetadataPath(metadataPath); err == nil {
			data, err := readFileSafely(metadataPath, maxJSONSize)
			if err == nil {
				var metadata struct {
					WorkingDir string `json:"working_dir"`
				}
				if json.Unmarshal(data, &metadata) == nil && metadata.WorkingDir != "" {
					targetWorkdir = metadata.WorkingDir
					log.Printf("using working directory from metadata: %s", targetWorkdir)
				}
			}
		}
	}

	if targetWorkdir == "" {
		return nil // No working directory specified
	}

	// AIDEV-BUSINESS_RULE: Validate working directory path
	if err := validateWorkingDirectory(targetWorkdir); err != nil {
		return fmt.Errorf("invalid working directory: %w", err)
	}

	// Ensure the directory exists
	if _, err := os.Stat(targetWorkdir); os.IsNotExist(err) {
		return fmt.Errorf("working directory %s does not exist", targetWorkdir)
	}

	if err := os.Chdir(targetWorkdir); err != nil {
		return fmt.Errorf("failed to change to %s: %w", targetWorkdir, err)
	}
	log.Printf("changed working directory to: %s", targetWorkdir)
	return nil
}

// createCommonDirectories creates directories commonly expected by applications
func createCommonDirectories() {
	// List of directories that applications commonly expect to exist in a microvm
	commonDirs := []string{
		"/var/log",
		"/var/run",
		"/var/cache",
		"/tmp",
	}

	for _, dir := range commonDirs {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			log.Printf("warning: failed to create directory %s: %v", dir, err)
		} else {
			log.Printf("ensured directory exists: %s", dir)
		}
	}
}

// AIDEV-BUSINESS_RULE: Validate process group exists before signaling
func validateProcessGroup(pid int) error {
	// Check if the process group exists by getting its process group ID
	pgid, err := syscall.Getpgid(pid)
	if err != nil {
		return fmt.Errorf("failed to get process group for PID %d: %w", pid, err)
	}

	if pgid <= 0 {
		return fmt.Errorf("invalid process group ID %d for PID %d", pgid, pid)
	}

	return nil
}

// handleSignalsAndReaping handles signal forwarding and zombie process reaping
func handleSignalsAndReaping(cmd *exec.Cmd, sigChan chan os.Signal) {
	// Set up SIGCHLD handler for immediate zombie reaping
	sigChildChan := make(chan os.Signal, 1)
	signal.Notify(sigChildChan, syscall.SIGCHLD)

	// AIDEV-BUSINESS_RULE: Remove busy-wait loop, use proper blocking select
	for {
		select {
		case sig := <-sigChan:
			log.Printf("received signal: %v, forwarding to child process", sig)
			if cmd.Process != nil {
				// AIDEV-BUSINESS_RULE: Validate process group before signaling
				if err := validateProcessGroup(cmd.Process.Pid); err != nil {
					log.Printf("warning: cannot validate process group: %v", err)
					continue
				}

				// Forward signal to the entire process group
				if err := syscall.Kill(-cmd.Process.Pid, sig.(syscall.Signal)); err != nil {
					log.Printf("warning: failed to forward signal: %v", err)
				}
			}

		case <-sigChildChan:
			// SIGCHLD received, reap any zombie processes with bounds
			reapedCount := 0
			maxReapIterations := 100 // Prevent infinite loops

			for i := 0; i < maxReapIterations; i++ {
				var status syscall.WaitStatus
				pid, err := syscall.Wait4(-1, &status, syscall.WNOHANG, nil)
				if err != nil {
					if err != syscall.ECHILD {
						log.Printf("wait4 error: %v", err)
					}
					break
				}
				if pid <= 0 {
					// No more children to reap
					break
				}
				reapedCount++
				log.Printf("reaped zombie process: PID %d, status: %v", pid, status)
			}

			if reapedCount > 0 {
				log.Printf("reaped %d zombie processes", reapedCount)
			}
		}
		// AIDEV-NOTE: Removed default case to eliminate busy-wait loop
	}
}

// ContainerEnvironment represents container runtime environment configuration
// AIDEV-NOTE: This matches the structure created by builderd's createContainerEnv function
type ContainerEnvironment struct {
	WorkingDir   string            `json:"working_dir,omitempty"`
	Env          map[string]string `json:"env,omitempty"`
	User         string            `json:"user,omitempty"`
	ExposedPorts []string          `json:"exposed_ports,omitempty"`
}

// loadContainerEnvironment loads container environment configuration from /container.env
// AIDEV-NOTE: This function provides complete container runtime environment replication
func loadContainerEnvironment() (*ContainerEnvironment, error) {
	envData, err := readFileSafely("/container.env", maxJSONSize)
	if err != nil {
		return nil, fmt.Errorf("failed to read container.env: %w", err)
	}

	var containerEnv ContainerEnvironment
	if err := json.Unmarshal(envData, &containerEnv); err != nil {
		return nil, fmt.Errorf("failed to parse container.env: %w", err)
	}

	log.Printf("loaded container environment: workdir=%s, env_vars=%d, user=%s",
		containerEnv.WorkingDir, len(containerEnv.Env), containerEnv.User)

	return &containerEnv, nil
}

// applyContainerEnvironment applies container environment configuration
// AIDEV-NOTE: This sets up the complete container runtime environment
func applyContainerEnvironment(containerEnv *ContainerEnvironment) error {
	if containerEnv == nil {
		// AIDEV-NOTE: No container.env file - environment will be set from kernel cmdline instead
		log.Printf("no container.env found - relying on kernel cmdline environment")
		return nil
	}

	// Set environment variables
	if containerEnv.Env != nil {
		for key, value := range containerEnv.Env {
			if err := validateEnvVar(key, value); err != nil {
				log.Printf("warning: skipping invalid env var %s: %v", key, err)
				continue
			}
			if err := os.Setenv(key, value); err != nil {
				return fmt.Errorf("failed to set env var %s: %w", key, err)
			}
		}
		log.Printf("applied %d environment variables", len(containerEnv.Env))
	}

	// Change working directory
	if containerEnv.WorkingDir != "" && containerEnv.WorkingDir != "/" {
		if err := os.Chdir(containerEnv.WorkingDir); err != nil {
			return fmt.Errorf("failed to change working directory to %s: %w", containerEnv.WorkingDir, err)
		}
		log.Printf("changed working directory to: %s", containerEnv.WorkingDir)
	}

	return nil
}

// printHelp prints usage information
func printHelp() {
	binaryName := filepath.Base(os.Args[0])
	help := fmt.Sprintf(`%s - Generic init process for microvms

Usage:
  %s [options] -- command [args...]

Options:
  --version    Show version information
  --help       Show this help message

Environment:
  The init process reads kernel command line parameters from /proc/cmdline:

  env.KEY=VALUE    Set environment variable KEY to VALUE
  workdir=/path    Change working directory to /path

Example:
  %s -- nginx -g "daemon off;"

  With kernel cmdline: env.NGINX_PORT=8080 workdir=/app
`, binaryName, binaryName, binaryName)
	fmt.Print(help)
}
