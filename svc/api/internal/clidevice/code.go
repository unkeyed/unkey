package clidevice

import (
	"crypto/rand"
	"encoding/json"
	"errors"
	"net/url"
	"strings"

	"github.com/unkeyed/unkey/pkg/uid"
)

const userCodeAlphabet = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789"

func newUserCode() (string, error) {
	buf := make([]byte, 8)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	chars := make([]byte, 8)
	for i, b := range buf {
		chars[i] = userCodeAlphabet[int(b)%len(userCodeAlphabet)]
	}
	return string(chars[:4]) + "-" + string(chars[4:]), nil
}

func newLoginID() string {
	return string(uid.DeviceAuthorizationPrefix) + "_" + uid.Secure(24)
}

func normalizeUserCode(raw string) (string, error) {
	code := strings.ToUpper(strings.TrimSpace(raw))
	if len(code) < 4 || len(code) > 32 {
		return "", invalid("Enter the code shown in the terminal.")
	}
	for _, r := range code {
		if (r < 'A' || r > 'Z') && (r < '0' || r > '9') && r != '-' {
			return "", invalid("Enter the code shown in the terminal.")
		}
	}
	return code, nil
}

func normalizeLoginID(raw string) (string, error) {
	id := strings.TrimSpace(raw)
	if len(id) < 8 || len(id) > 48 {
		return "", invalid("The login id is invalid.")
	}
	for _, r := range id {
		if (r < 'a' || r > 'z') && (r < 'A' || r > 'Z') && (r < '0' || r > '9') && r != '_' {
			return "", invalid("The login id is invalid.")
		}
	}
	return id, nil
}

func dashboardVerificationURL(base string, userCode string) (string, error) {
	parsed, err := url.Parse(base)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" || (parsed.Scheme != "http" && parsed.Scheme != "https") {
		return "", errors.New("dashboard base url must be an absolute http(s) url")
	}
	parsed.Path = strings.TrimRight(parsed.Path, "/") + "/cli/device"
	query := parsed.Query()
	query.Set("user_code", userCode)
	parsed.RawQuery = query.Encode()
	parsed.Fragment = ""
	return parsed.String(), nil
}

func truncate(value string, maxRunes int) string {
	if runes := []rune(value); len(runes) > maxRunes {
		return string(runes[:maxRunes])
	}
	return value
}

func decodeStrings(raw []byte) ([]string, error) {
	if len(raw) == 0 {
		return nil, nil
	}
	var values []string
	if err := json.Unmarshal(raw, &values); err != nil {
		return nil, err
	}
	return values, nil
}
