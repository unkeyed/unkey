package keystore

import (
	"errors"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
)

// securityMaxLineLen is the number of bytes macOS `security -i` keeps from one
// command. readline() stores at most MAX_LINE_LEN-1 bytes and then stops, so a
// longer add-generic-password line is saved without its tail.
const securityMaxLineLen = 4095

// chunkHeader marks a primary keychain item whose value is the count of
// following items. It is not valid session JSON and is not a root key.
// Chunk i is stored under account + "/" + i.
const chunkHeader = "unkey-keystore-chunks:v1:"

func securityGet(bin, service, account string) (string, error) {
	raw, err := readItem(bin, service, account)
	if err != nil {
		return "", err
	}
	n := chunkCount(raw)
	if n == 0 {
		return raw, nil
	}
	var b strings.Builder
	for i := 0; i < n; i++ {
		part, err := readItem(bin, service, chunkAccount(account, i))
		if err != nil {
			_ = discardItems(bin, service, account, n)
			if errors.Is(err, ErrNotFound) {
				return "", unreadableSecret()
			}
			return "", err
		}
		b.WriteString(part)
	}
	return b.String(), nil
}

// securitySet writes a generic password through `security -i` so the secret
// travels on stdin instead of argv, where other local processes could read it.
//
// Interactive mode reads at most 4095 bytes of a command and does not reliably
// report that it dropped the rest. A secret that does not fit in one command
// is split across several items. The write is confirmed by reading the value
// back; a mismatch is deleted so a truncated item is not left behind.
func securitySet(bin, service, account, secret string) error {
	if strings.ContainsAny(secret, "\r\n") {
		return fmt.Errorf("security: secret must not contain line breaks")
	}
	parts, chunked, err := planChunks(service, account, secret)
	if err != nil {
		return err
	}
	oldN := 0
	if raw, readErr := readItem(bin, service, account); readErr == nil {
		oldN = chunkCount(raw)
	}
	if err := storeChunks(bin, service, account, parts, chunked); err != nil {
		_ = discardItems(bin, service, account, len(parts))
		return err
	}
	if !chunked {
		_ = discardChunkItems(bin, service, account, oldN)
	} else if oldN > len(parts) {
		_ = discardChunkRange(bin, service, account, len(parts), oldN)
	}
	stored, err := securityGet(bin, service, account)
	if err != nil || stored != secret {
		n := len(parts)
		if !chunked {
			n = 0
		}
		if oldN > n {
			n = oldN
		}
		_ = discardItems(bin, service, account, n)
		if err != nil && !errors.Is(err, ErrNotFound) {
			return err
		}
		return withDetail(ErrUnavailable, "keychain did not keep the secret")
	}
	return nil
}

func planChunks(service, account, secret string) ([]string, bool, error) {
	if lineFits(service, account, secret) {
		return []string{secret}, false, nil
	}
	var parts []string
	rest := secret
	for len(rest) > 0 {
		n := fitPrefix(service, chunkAccount(account, len(parts)), rest)
		if n <= 0 {
			return nil, false, fmt.Errorf("security: secret cannot be stored")
		}
		parts = append(parts, rest[:n])
		rest = rest[n:]
		if len(parts) > 1024 {
			return nil, false, fmt.Errorf("security: secret cannot be stored")
		}
	}
	header := chunkHeader + strconv.Itoa(len(parts))
	if !lineFits(service, account, header) {
		return nil, false, fmt.Errorf("security: secret cannot be stored")
	}
	return parts, true, nil
}

func storeChunks(bin, service, account string, parts []string, chunked bool) error {
	if !chunked {
		return writeItem(bin, service, account, parts[0])
	}
	for i, part := range parts {
		if err := writeItem(bin, service, chunkAccount(account, i), part); err != nil {
			return err
		}
	}
	return writeItem(bin, service, account, chunkHeader+strconv.Itoa(len(parts)))
}

func writeItem(bin, service, account, secret string) error {
	command := securityCommand(service, account, secret)
	if !lineFits(service, account, secret) {
		return fmt.Errorf("security: item exceeds keychain command limit")
	}
	_, stderr, err := runCommand(bin, []string{"-i"}, command)
	if err != nil || strings.TrimSpace(stderr) != "" {
		if err == nil {
			err = errors.New("add-generic-password failed")
		}
		return classifySecurity(err, stderr, false)
	}
	return nil
}

func securityCommand(service, account, secret string) string {
	return fmt.Sprintf("add-generic-password -U -s %s -a %s -w %s\n",
		quoteSecurityArg(service), quoteSecurityArg(account), quoteSecurityArg(secret))
}

func lineFits(service, account, secret string) bool {
	command := securityCommand(service, account, secret)
	if !strings.HasSuffix(command, "\n") {
		return false
	}
	return len(command)-1 <= securityMaxLineLen
}

func fitPrefix(service, account, secret string) int {
	if secret == "" {
		return 0
	}
	if lineFits(service, account, secret) {
		return len(secret)
	}
	lo, hi := 1, len(secret)-1
	best := 0
	for lo <= hi {
		mid := (lo + hi) / 2
		if lineFits(service, account, secret[:mid]) {
			best = mid
			lo = mid + 1
		} else {
			hi = mid - 1
		}
	}
	return best
}

func chunkAccount(account string, index int) string {
	return account + "/" + strconv.Itoa(index)
}

func chunkCount(raw string) int {
	if !strings.HasPrefix(raw, chunkHeader) {
		return 0
	}
	n, err := strconv.Atoi(strings.TrimPrefix(raw, chunkHeader))
	if err != nil || n <= 0 || n > 1024 {
		return 0
	}
	return n
}

func readItem(bin, service, account string) (string, error) {
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

func quoteSecurityArg(value string) string {
	escaped := strings.NewReplacer(`\`, `\\`, `"`, `\"`).Replace(value)
	return `"` + escaped + `"`
}

func securityDelete(bin, service, account string) error {
	raw, err := readItem(bin, service, account)
	n := 0
	if err == nil {
		n = chunkCount(raw)
	} else if !errors.Is(err, ErrNotFound) {
		return err
	}
	if err := deleteItem(bin, service, account); err != nil {
		return err
	}
	return discardChunkItems(bin, service, account, n)
}

func discardItems(bin, service, account string, n int) error {
	if err := deleteItem(bin, service, account); err != nil && !errors.Is(err, ErrNotFound) {
		return err
	}
	return discardChunkItems(bin, service, account, n)
}

func discardChunkItems(bin, service, account string, n int) error {
	return discardChunkRange(bin, service, account, 0, n)
}

func discardChunkRange(bin, service, account string, from, to int) error {
	for i := from; i < to; i++ {
		if err := deleteItem(bin, service, chunkAccount(account, i)); err != nil && !errors.Is(err, ErrNotFound) {
			return err
		}
	}
	return nil
}

func deleteItem(bin, service, account string) error {
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

func unreadableSecret() error {
	return fmt.Errorf("stored secret is unreadable; run unkey login again")
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
