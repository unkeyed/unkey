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
		return classifySecretToolWrite(err, stderr)
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

func classifySecretToolWrite(err error, stderr string) error {
	if errors.Is(err, exec.ErrNotFound) {
		return ErrUnavailable
	}
	msg := strings.TrimSpace(stderr)
	if secretServiceUnavailable(msg) {
		return withDetail(ErrUnavailable, msg)
	}
	if msg == "" {
		return fmt.Errorf("secret-tool: %w", err)
	}
	return fmt.Errorf("secret-tool: %w: %s", err, msg)
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
	case secretServiceUnavailable(msg):
		return withDetail(ErrUnavailable, msg)
	case strings.Contains(lower, "not found") ||
		strings.Contains(lower, "no such") ||
		strings.Contains(lower, "no matching"):
		return ErrNotFound
	default:
		return fmt.Errorf("secret-tool: %w: %s", err, msg)
	}
}

// secretServiceUnavailable matches D-Bus failures that mean no Secret Service
// can answer. dbus-daemon reports a missing org.freedesktop.secrets as "not
// provided by any .service files"; dbus-broker, the default on Fedora, reports
// "The name is not activatable". "could not connect" means there is no
// session bus socket.
func secretServiceUnavailable(msg string) bool {
	lower := strings.ToLower(msg)
	for _, s := range []string{
		"not provided by any",
		"not activatable",
		"cannot autolaunch",
		"could not connect",
		"locked",
		"no session",
	} {
		if strings.Contains(lower, s) {
			return true
		}
	}
	return false
}
