package validation

import "regexp"

// slugRegex matches lowercase alphanumeric strings with single hyphens,
// 3–64 characters, not starting or ending with a hyphen.
var slugRegex = regexp.MustCompile(`^[a-z0-9]([a-z0-9-]*[a-z0-9])?$`)

// consecutiveHyphens detects runs of two or more hyphens.
var consecutiveHyphens = regexp.MustCompile(`--`)

const (
	SlugMinLength     = 3
	SlugMaxLength     = 64
	ErrMsgInvalidSlug = "slug must be 3-64 characters, lowercase alphanumeric and hyphens, " +
		"must not start or end with a hyphen, and must not contain consecutive hyphens"
)

// ValidateSlug reports whether s is a valid portal configuration slug.
func ValidateSlug(s string) bool {
	if len(s) < SlugMinLength || len(s) > SlugMaxLength {
		return false
	}
	if consecutiveHyphens.MatchString(s) {
		return false
	}
	return slugRegex.MatchString(s)
}
