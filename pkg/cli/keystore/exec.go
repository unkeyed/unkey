package keystore

import (
	"bytes"
	"errors"
	"fmt"
	"os/exec"
	"strings"
)

func runCommand(bin string, args []string, stdin string) (string, string, error) {
	cmd := exec.Command(bin, args...)
	cmd.Stdin = strings.NewReader(stdin)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
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
