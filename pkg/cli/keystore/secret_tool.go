package keystore

import (
	"errors"
	"fmt"
	"os/exec"
	"strings"
)

const secretToolLabel = "Unkey root key"

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
		return secretToolUnavailable(err, strings.TrimSpace(stderr))
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

// classifySecretTool maps a failed lookup or clear. secret-tool exits with no
// output when no item matches, so empty stderr is [ErrNotFound].
func classifySecretTool(err error, stderr string) error {
	msg := strings.TrimSpace(stderr)
	missing := msg == "" || strings.Contains(strings.ToLower(msg), "no such secret item")
	if missing && !errors.Is(err, ErrUnavailable) && !errors.Is(err, exec.ErrNotFound) {
		return ErrNotFound
	}
	return secretToolUnavailable(err, msg)
}

// secretToolUnavailable reports every other secret-tool failure as
// [ErrUnavailable], so callers fall back to a file. The messages come from
// D-Bus and the Secret Service provider and differ between dbus-daemon,
// dbus-broker, and each provider, so no list of them can tell a store that
// could work from one that cannot.
func secretToolUnavailable(err error, msg string) error {
	if errors.Is(err, ErrUnavailable) {
		return err
	}
	if msg == "" {
		return fmt.Errorf("%w: secret-tool: %w", ErrUnavailable, err)
	}
	return fmt.Errorf("%w: secret-tool: %w: %s", ErrUnavailable, err, msg)
}
