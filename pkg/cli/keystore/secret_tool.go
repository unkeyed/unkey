package keystore

import (
	"errors"
	"fmt"
	"os/exec"
	"strings"
)

const secretToolLabel = "Unkey root key"

const secretToolBin = "secret-tool"

func secretToolGet(bin, service, account string) (string, error) {
	stdout, stderr, err := runCommand(bin, []string{
		"lookup",
		"service", service,
		"account", account,
	}, "")
	if err != nil {
		return "", classifySecretTool(err, stderr)
	}
	return strings.TrimRight(stdout, "\r\n"), nil
}

func secretToolSet(bin, service, account, secret string) error {
	_, stderr, err := runCommand(bin, []string{
		"store",
		"--label=" + secretToolLabel,
		"service", service,
		"account", account,
	}, secret)
	if err != nil {
		return classifySecretTool(err, stderr)
	}
	return nil
}

func secretToolDelete(bin, service, account string) error {
	_, stderr, err := runCommand(bin, []string{
		"clear",
		"service", service,
		"account", account,
	}, "")
	if err == nil {
		return nil
	}
	classified := classifySecretTool(err, stderr)
	if errors.Is(classified, ErrNotFound) {
		return nil
	}
	return classified
}

func classifySecretTool(err error, stderr string) error {
	if errors.Is(err, exec.ErrNotFound) {
		return ErrUnavailable
	}
	msg := strings.TrimSpace(stderr)
	lower := strings.ToLower(msg)
	switch {
	case msg == "":
		return ErrNotFound
	case strings.Contains(lower, "not provided by any") ||
		strings.Contains(lower, "cannot autolaunch") ||
		strings.Contains(lower, "locked") ||
		strings.Contains(lower, "no session"):
		return withDetail(ErrUnavailable, msg)
	case strings.Contains(lower, "not found") ||
		strings.Contains(lower, "no such") ||
		strings.Contains(lower, "no matching"):
		return ErrNotFound
	default:
		return fmt.Errorf("secret-tool: %w: %s", err, msg)
	}
}
