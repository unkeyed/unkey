// Package imageref parses and normalizes OCI image references used by deploys.
package imageref

import (
	"fmt"
	"strings"

	"github.com/google/go-containerregistry/pkg/name"
)

// Parse validates and normalizes an image reference. References must include
// an explicit tag or digest so app configuration never relies on an implicit
// latest tag.
func Parse(raw string) (name.Reference, error) {
	reference := strings.TrimSpace(raw)
	if reference == "" {
		return nil, fmt.Errorf("image reference is required")
	}

	lastSlash := strings.LastIndex(reference, "/")
	lastColon := strings.LastIndex(reference, ":")
	if !strings.Contains(reference, "@") && lastColon <= lastSlash {
		return nil, fmt.Errorf("image reference %q must include an explicit tag or digest", reference)
	}

	parsed, err := name.ParseReference(reference, name.WeakValidation)
	if err != nil {
		return nil, fmt.Errorf("invalid image reference %q: %w", reference, err)
	}

	return parsed, nil
}

// Normalize validates a reference and returns its canonical name.
func Normalize(raw string) (string, error) {
	parsed, err := Parse(raw)
	if err != nil {
		return "", err
	}
	return parsed.Name(), nil
}

// NormalizeHistorical canonicalizes an image stored before explicit tags were
// required. go-containerregistry makes the old implicit-latest behavior
// explicit in the normalized result.
func NormalizeHistorical(raw string) (string, error) {
	reference := strings.TrimSpace(raw)
	if reference == "" {
		return "", fmt.Errorf("image reference is required")
	}
	parsed, err := name.ParseReference(reference, name.WeakValidation)
	if err != nil {
		return "", fmt.Errorf("invalid image reference %q: %w", reference, err)
	}
	return parsed.Name(), nil
}

// IsDigest reports whether a parsed reference is already immutable.
func IsDigest(reference name.Reference) bool {
	_, ok := reference.(name.Digest)
	return ok
}
