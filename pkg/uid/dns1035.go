package uid

import (
	"fmt"
	"regexp"
	"strings"
)

var dns1035Pattern = regexp.MustCompile(`^[a-z][a-z0-9_]*$`)

// ToDNS1035 converts an identifier to DNS-1035 format by replacing underscores
// with dashes. Returns an error if the input contains invalid characters.
//
// DNS-1035 labels must start with a lowercase letter and contain only lowercase
// letters, digits, and dashes. Empty string returns empty string without error.
func ToDNS1035(s string) (string, error) {
	if s == "" {
		return "", nil
	}

	if !dns1035Pattern.MatchString(s) {
		return "", fmt.Errorf("%s can not be converted to DNS1035", s)
	}
	return strings.ReplaceAll(s, "_", "-"), nil
}

// FromDNS1035 converts a DNS-1035 label back to identifier format by replacing
// dashes with underscores. Does not validate the input.
func FromDNS1035(s string) string {
	return strings.ReplaceAll(s, "-", "_")
}
