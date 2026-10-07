package keystore

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

// commandTimeout is generous because the OS may be waiting on the user to
// unlock the keychain or keyring.
const commandTimeout = 2 * time.Minute

func runCommand(bin string, args []string, stdin string) (string, string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), commandTimeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, bin, args...)
	cmd.Stdin = strings.NewReader(stdin)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	if errors.Is(ctx.Err(), context.DeadlineExceeded) {
		return stdout.String(), stderr.String(), withDetail(ErrUnavailable, bin+" timed out")
	}
	return stdout.String(), stderr.String(), err
}

func exitCode(err error) int {
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		return exitErr.ExitCode()
	}
	return -1
}

func withDetail(sentinel error, detail string) error {
	detail = strings.TrimSpace(detail)
	if detail == "" {
		return sentinel
	}
	return fmt.Errorf("%w: %s", sentinel, detail)
}
