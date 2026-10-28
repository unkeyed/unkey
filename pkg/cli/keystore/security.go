package keystore

import (
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

// chunkHeader starts the manifest written last. Pieces live at
// account/<slot>/<i> and are invisible until this one item is replaced. A write
// uses the slot the current manifest does not reference, so at most two slots
// ever hold pieces and a delete can sweep both.
const chunkHeader = "unkey-keystore-chunks:v1:"

const chunkPiecesMax = 1024

var chunkSlots = [2]string{"a", "b"}

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
		part, err := readItem(bin, service, chunkAccount(account, ref.slot, i))
		if err != nil {
			if errors.Is(err, ErrNotFound) {
				_ = deleteItem(bin, service, account)
				_ = discardSlot(bin, service, account, ref.slot, ref.n)
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
// is split across several items in the slot the current manifest does not
// reference. The primary item is replaced only after those pieces are stored,
// so a crash keeps the previous secret. The write is confirmed by reading the value back. A mismatch, or a
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
	oldRaw := ""
	oldRef := chunkRef{slot: "", n: 0}
	raw, readErr := readItem(bin, service, account)
	if readErr == nil {
		oldRaw = raw
		if ref, parseErr := parseChunkHeader(raw); parseErr == nil {
			oldRef = ref
		}
	} else if !errors.Is(readErr, ErrNotFound) {
		return readErr
	}
	slot := chunkSlots[0]
	if oldRef.slot == chunkSlots[0] {
		slot = chunkSlots[1]
	}
	parts, chunked, err := planChunks(service, account, slot, secret)
	if err != nil {
		return err
	}
	if chunked {
		if err := sweepSlot(bin, service, account, slot); err != nil {
			return err
		}
	}
	if err := storeChunks(bin, service, account, slot, parts, chunked); err != nil {
		if chunked {
			_ = discardSlot(bin, service, account, slot, len(parts))
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
			_ = discardSlot(bin, service, account, slot, len(parts))
		}
		return withDetail(ErrUnavailable, "keychain did not keep the secret")
	}
	if oldRef.n > 0 {
		_ = discardSlot(bin, service, account, oldRef.slot, oldRef.n)
	}
	return nil
}

func planChunks(service, account, slot, secret string) ([]string, bool, error) {
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
		name := chunkAccount(account, slot, len(parts))
		for n > 0 && !lineFits(service, name, rest[:n]) {
			n = cutUTF8(rest, n-1)
		}
		if n <= 0 {
			return nil, false, fmt.Errorf("security: secret cannot be stored")
		}
		parts = append(parts, rest[:n])
		rest = rest[n:]
		if len(parts) > chunkPiecesMax {
			return nil, false, fmt.Errorf("security: secret cannot be stored")
		}
	}
	if !lineFits(service, account, chunkManifest(slot, len(parts))) {
		return nil, false, fmt.Errorf("security: secret cannot be stored")
	}
	return parts, true, nil
}

func storeChunks(bin, service, account, slot string, parts []string, chunked bool) error {
	if !chunked {
		return writeItem(bin, service, account, parts[0])
	}
	for i, part := range parts {
		if err := writeItem(bin, service, chunkAccount(account, slot, i), part); err != nil {
			return err
		}
	}
	return writeItem(bin, service, account, chunkManifest(slot, len(parts)))
}

func chunkManifest(slot string, n int) string {
	return chunkHeader + slot + ":" + strconv.Itoa(n)
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
	slot string
	n    int
}

func chunkAccount(account, slot string, index int) string {
	return account + "/" + slot + "/" + strconv.Itoa(index)
}

func parseChunkHeader(raw string) (chunkRef, error) {
	if !strings.HasPrefix(raw, chunkHeader) {
		return chunkRef{slot: "", n: 0}, nil
	}
	slot, count, ok := strings.Cut(strings.TrimPrefix(raw, chunkHeader), ":")
	if !ok || (slot != chunkSlots[0] && slot != chunkSlots[1]) {
		return chunkRef{}, fmt.Errorf("security: unparseable keychain chunk header")
	}
	n, err := strconv.Atoi(count)
	if err != nil || n <= 0 || n > chunkPiecesMax {
		return chunkRef{}, fmt.Errorf("security: unparseable keychain chunk header")
	}
	return chunkRef{slot: slot, n: n}, nil
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
	if _, err := readItem(bin, service, account); err != nil && !errors.Is(err, ErrNotFound) {
		return err
	}
	if err := deleteItem(bin, service, account); err != nil {
		return err
	}
	for _, slot := range chunkSlots {
		if err := sweepSlot(bin, service, account, slot); err != nil {
			return err
		}
	}
	return nil
}

// sweepSlot removes every piece in a slot whose count is unknown. Pieces are
// written from index 0 up and removed from the top down, so a slot always
// holds a contiguous run starting at 0 and the first missing index ends it.
func sweepSlot(bin, service, account, slot string) error {
	n := 0
	for ; n < chunkPiecesMax; n++ {
		_, err := readItem(bin, service, chunkAccount(account, slot, n))
		if errors.Is(err, ErrNotFound) {
			break
		}
		if err != nil {
			return err
		}
	}
	return discardSlot(bin, service, account, slot, n)
}

func discardSlot(bin, service, account, slot string, n int) error {
	for i := n - 1; i >= 0; i-- {
		if err := deleteItem(bin, service, chunkAccount(account, slot, i)); err != nil {
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
