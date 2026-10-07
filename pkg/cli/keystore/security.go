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

// securitySet writes a generic password through `security -i` so the secret
// travels on stdin instead of argv, where other local processes could read it.
// Interactive mode does not reliably report failures in its exit status, so the
// write is confirmed by reading the item back.
func securitySet(bin, service, account, secret string) error {
	if strings.ContainsAny(secret, "\r\n") {
		return fmt.Errorf("security: secret must not contain line breaks")
	}
	command := fmt.Sprintf("add-generic-password -U -s %s -a %s -w %s\n",
		quoteSecurityArg(service), quoteSecurityArg(account), quoteSecurityArg(secret))
	_, stderr, err := runCommand(bin, []string{"-i"}, command)
	if err != nil || strings.TrimSpace(stderr) != "" {
		if err == nil {
			err = errors.New("add-generic-password failed")
		}
		return classifySecurity(err, stderr, false)
	}
	stored, err := securityGet(bin, service, account)
	if errors.Is(err, ErrNotFound) || (err == nil && stored != secret) {
		return withDetail(ErrUnavailable, "keychain did not keep the secret")
	}
	return err
}

func quoteSecurityArg(value string) string {
	escaped := strings.NewReplacer(`\`, `\\`, `"`, `\"`).Replace(value)
	return `"` + escaped + `"`
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
