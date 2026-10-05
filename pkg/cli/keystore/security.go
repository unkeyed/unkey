package keystore

import (
	"errors"
	"fmt"
	"os/exec"
	"strings"
)

func securityGet(bin, service, account string) (string, error) {
	stdout, stderr, err := runCommand(bin, []string{
		"find-generic-password",
		"-s", service,
		"-a", account,
		"-w",
	}, "")
	if err != nil {
		return "", classifySecurity(err, stderr, true)
	}
	return strings.TrimRight(stdout, "\r\n"), nil
}

// securitySet writes a generic password. The secret is an argument because
// the security tool cannot read it from stdin, so it is briefly visible in
// the process list.
func securitySet(bin, service, account, secret string) error {
	_, stderr, err := runCommand(bin, []string{
		"add-generic-password",
		"-U",
		"-s", service,
		"-a", account,
		"-w", secret,
	}, "")
	if err != nil {
		return classifySecurity(err, stderr, false)
	}
	return nil
}

func securityDelete(bin, service, account string) error {
	_, stderr, err := runCommand(bin, []string{
		"delete-generic-password",
		"-s", service,
		"-a", account,
	}, "")
	if err == nil {
		return nil
	}
	classified := classifySecurity(err, stderr, true)
	if errors.Is(classified, ErrNotFound) {
		return nil
	}
	return classified
}

func classifySecurity(err error, stderr string, allowNotFound bool) error {
	if errors.Is(err, exec.ErrNotFound) {
		return ErrUnavailable
	}
	lower := strings.ToLower(stderr)
	if strings.Contains(lower, "interaction is not allowed") {
		return withDetail(ErrUnavailable, stderr)
	}
	notFound := exitCode(err) == 44 ||
		strings.Contains(lower, "could not be found") ||
		strings.Contains(lower, "item not found")
	if allowNotFound && notFound {
		return ErrNotFound
	}
	detail := strings.TrimSpace(stderr)
	if detail == "" {
		return fmt.Errorf("security: %w", err)
	}
	return fmt.Errorf("security: %w: %s", err, detail)
}
