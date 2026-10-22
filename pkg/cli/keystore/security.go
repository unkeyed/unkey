package keystore

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
	"unicode/utf8"
)

// securityMaxLineLen is the number of bytes macOS `security -i` keeps from one
// command. readline() stores at most MAX_LINE_LEN-1 bytes and then stops, so a
// longer add-generic-password line is saved without its tail.
const securityMaxLineLen = 4095

// chunkPieceBytes is the target size of one keychain item. It stays under the
// interactive line limit after the command and quoting are added.
const chunkPieceBytes = 3500

// chunkHeader is the legacy manifest: the primary item holds the piece count
// and piece i is stored at account/i.
const chunkHeader = "unkey-keystore-chunks:v1:"

// nonceHeader is the manifest written last. Pieces live at
// account/<nonce>/<i> and are invisible until this one item is replaced.
const nonceHeader = "unkey-keystore-nonce:v1:"

func securityGet(bin, service, account string) (string, error) {
	unlock, err := lockKeystore()
	if err != nil {
		return "", err
	}
	defer unlock()
	return securityGetUnlocked(bin, service, account)
}

func securityGetUnlocked(bin, service, account string) (string, error) {
	raw, err := readItem(bin, service, account)
	if err != nil {
		return "", err
	}
	ref, err := parseChunkHeader(raw)
	if err != nil {
		return "", err
	}
	if ref.n == 0 {
		return raw, nil
	}
	var b strings.Builder
	for i := 0; i < ref.n; i++ {
		part, err := readItem(bin, service, chunkAccount(account, ref.nonce, i))
		if err != nil {
			if errors.Is(err, ErrNotFound) {
				_ = deleteItem(bin, service, account)
				_ = discardNonce(bin, service, account, ref.nonce)
				return "", fmt.Errorf("stored secret is unreadable; run unkey login again: %w", ErrNotFound)
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
// is split across several items under a fresh nonce. The primary item is
// replaced only after those pieces are stored, so a crash keeps the previous
// secret. The write is confirmed by reading the value back. A mismatch, or a
// missing item, restores the previous manifest. Any other read error leaves
// the stored items in place.
func securitySet(bin, service, account, secret string) error {
	unlock, err := lockKeystore()
	if err != nil {
		return err
	}
	defer unlock()
	return securitySetUnlocked(bin, service, account, secret)
}

func securitySetUnlocked(bin, service, account, secret string) error {
	if strings.ContainsAny(secret, "\r\n") {
		return fmt.Errorf("security: secret must not contain line breaks")
	}
	nonce, err := newNonce()
	if err != nil {
		return err
	}
	parts, chunked, err := planChunks(service, account, nonce, secret)
	if err != nil {
		return err
	}
	oldRaw := ""
	oldRef := chunkRef{}
	if raw, readErr := readItem(bin, service, account); readErr == nil {
		oldRaw = raw
		if ref, parseErr := parseChunkHeader(raw); parseErr == nil {
			oldRef = ref
		}
	}
	if err := storeChunks(bin, service, account, nonce, parts, chunked); err != nil {
		if chunked {
			_ = discardNonce(bin, service, account, nonce)
		}
		return err
	}
	stored, err := securityGetUnlocked(bin, service, account)
	if err != nil && !errors.Is(err, ErrNotFound) {
		return err
	}
	if err != nil || stored != secret {
		if oldRaw != "" {
			_ = writeItem(bin, service, account, oldRaw)
		} else {
			_ = deleteItem(bin, service, account)
		}
		if chunked {
			_ = discardNonce(bin, service, account, nonce)
		}
		return withDetail(ErrUnavailable, "keychain did not keep the secret")
	}
	if oldRef.n > 0 && oldRef.nonce != nonce {
		_ = discardNonce(bin, service, account, oldRef.nonce)
	}
	return nil
}

func planChunks(service, account, nonce, secret string) ([]string, bool, error) {
	if lineFits(service, account, secret) {
		return []string{secret}, false, nil
	}
	var parts []string
	rest := secret
	for len(rest) > 0 {
		n := chunkPieceBytes
		if n > len(rest) {
			n = len(rest)
		}
		n = cutUTF8(rest, n)
		name := chunkAccount(account, nonce, len(parts))
		for n > 0 && !lineFits(service, name, rest[:n]) {
			n = cutUTF8(rest, n-1)
		}
		if n <= 0 {
			return nil, false, fmt.Errorf("security: secret cannot be stored")
		}
		parts = append(parts, rest[:n])
		rest = rest[n:]
		if len(parts) > 1024 {
			return nil, false, fmt.Errorf("security: secret cannot be stored")
		}
	}
	header := nonceHeader + nonce + ":" + strconv.Itoa(len(parts))
	if !lineFits(service, account, header) {
		return nil, false, fmt.Errorf("security: secret cannot be stored")
	}
	return parts, true, nil
}

func storeChunks(bin, service, account, nonce string, parts []string, chunked bool) error {
	if !chunked {
		return writeItem(bin, service, account, parts[0])
	}
	for i, part := range parts {
		if err := writeItem(bin, service, chunkAccount(account, nonce, i), part); err != nil {
			return err
		}
	}
	return writeItem(bin, service, account, nonceHeader+nonce+":"+strconv.Itoa(len(parts)))
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

type chunkRef struct {
	nonce string
	n     int
}

func chunkAccount(account, nonce string, index int) string {
	if nonce == "" {
		return account + "/" + strconv.Itoa(index)
	}
	return account + "/" + nonce + "/" + strconv.Itoa(index)
}

func parseChunkHeader(raw string) (chunkRef, error) {
	if strings.HasPrefix(raw, nonceHeader) {
		rest := strings.TrimPrefix(raw, nonceHeader)
		nonce, count, ok := strings.Cut(rest, ":")
		if !ok || nonce == "" || strings.ContainsAny(nonce, "/:") {
			return chunkRef{}, fmt.Errorf("security: unparseable keychain chunk header")
		}
		n, err := strconv.Atoi(count)
		if err != nil || n <= 0 || n > 1024 {
			return chunkRef{}, fmt.Errorf("security: unparseable keychain chunk header")
		}
		return chunkRef{nonce: nonce, n: n}, nil
	}
	if !strings.HasPrefix(raw, chunkHeader) {
		return chunkRef{}, nil
	}
	n, err := strconv.Atoi(strings.TrimPrefix(raw, chunkHeader))
	if err != nil || n <= 0 || n > 1024 {
		return chunkRef{}, fmt.Errorf("security: unparseable keychain chunk header")
	}
	return chunkRef{n: n}, nil
}

func newNonce() (string, error) {
	buf := make([]byte, 8)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf), nil
}

func cutUTF8(s string, n int) int {
	if n >= len(s) {
		return len(s)
	}
	if n < 0 {
		return 0
	}
	for n > 0 && !utf8.RuneStart(s[n]) {
		n--
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
	unlock, err := lockKeystore()
	if err != nil {
		return err
	}
	defer unlock()
	raw, readErr := readItem(bin, service, account)
	ref := chunkRef{}
	if readErr == nil {
		if parsed, parseErr := parseChunkHeader(raw); parseErr == nil {
			ref = parsed
		}
	} else if !errors.Is(readErr, ErrNotFound) {
		return readErr
	}
	if err := deleteItem(bin, service, account); err != nil {
		return err
	}
	if err := discardNonce(bin, service, account, ref.nonce); err != nil {
		return err
	}
	if ref.nonce != "" {
		return discardNonce(bin, service, account, "")
	}
	return nil
}

func discardNonce(bin, service, account, nonce string) error {
	for i := 0; i < 1024; i++ {
		err := removeItem(bin, service, chunkAccount(account, nonce, i))
		if errors.Is(err, ErrNotFound) {
			return nil
		}
		if err != nil {
			return err
		}
	}
	return nil
}

func deleteItem(bin, service, account string) error {
	err := removeItem(bin, service, account)
	if errors.Is(err, ErrNotFound) {
		return nil
	}
	return err
}

func removeItem(bin, service, account string) error {
	_, stderr, err := runCommand(bin, []string{
		"delete-generic-password",
		"-s", service,
		"-a", account,
	}, "")
	if err == nil {
		return nil
	}
	return classifySecurity(err, stderr, true)
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
