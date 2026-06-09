package main

import (
	"fmt"
	"regexp"
	"slices"
	"strings"
)

// bumpKind is the stable-version increment applied when auto-numbering.
type bumpKind string

const (
	bumpPatch bumpKind = "patch"
	bumpMinor bumpKind = "minor"
	bumpMajor bumpKind = "major"
)

// bumpKinds lists the accepted --bump values, used for flag validation.
func bumpKinds() []string {
	return []string{string(bumpPatch), string(bumpMinor), string(bumpMajor)}
}

// supportedServices is the set of service names that map to a release tag
// namespace. Keep this in sync with releases.mdx and the CI workflows.
var supportedServices = []string{
	"api",
	"frontline",
	"vault",
	"heimdall",
	"krane",
	"control-api",
	"control-worker",
	"cli",
}

// semverPattern matches `vMAJOR.MINOR.PATCH` with an optional SemVer
// pre-release suffix (e.g. v1.2.3, v1.2.3-rc.1, v0.4.0-beta, v2.0.0-alpha.3).
// The suffix is one or more dot-separated identifiers, each non-empty and made
// of [0-9A-Za-z-] (hyphens are legal inside an identifier per SemVer §9). The
// per-identifier `+` rejects empty identifiers, so `-rc..1` or a trailing `-rc.`
// are refused before parseSemver can split them into an empty-string element.
var semverPattern = regexp.MustCompile(`^v\d+\.\d+\.\d+(-[0-9A-Za-z-]+(\.[0-9A-Za-z-]+)*)?$`)

// validSemver is a cli validator for version strings.
func validSemver(value string) error {
	if !semverPattern.MatchString(value) {
		return fmt.Errorf("expected vMAJOR.MINOR.PATCH with optional -prerelease (e.g. v1.2.3 or v1.2.3-rc.1)")
	}
	return nil
}

// validateService reports whether a service name is releasable.
func validateService(service string) error {
	if slices.Contains(supportedServices, service) {
		return nil
	}
	return fmt.Errorf("unsupported service %q (supported: %s)", service, strings.Join(supportedServices, ", "))
}
