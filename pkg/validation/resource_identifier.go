package validation

import (
	"regexp"
	"strings"
)

// idRegex matches a prefixed resource identifier: a lowercase prefix, an
// underscore, then a body.
//
// The body is deliberately permissive. Ids are opaque, and this repo mints
// several shapes: pkg/uid produces `pc_3ZfN2abcd`, while the dev seeder writes
// readable ones like `portal_awesome`, `portal_my-team`, and
// `ks_local_root_keys`. Pinning the body tighter would reject real ids to buy
// nothing, because the strictness that matters here is on the slug branch.
var idRegex = regexp.MustCompile(`^[a-z][a-z0-9]*_[a-zA-Z0-9_-]+$`)

// IDMaxLength bounds a resource id at the width of the `id` columns it is
// compared against. A longer value cannot match a stored row, so rejecting it
// as invalid input beats a misleading not-found.
const IDMaxLength = 48

// ErrMsgInvalidResourceIdentifier describes both accepted shapes. Callers
// prefix it with the field name.
const ErrMsgInvalidResourceIdentifier = "must be either a prefixed resource id (such as " +
	"`app_1234abcd`) or a slug of 3-64 characters, lowercase alphanumeric and hyphens, " +
	"not starting or ending with a hyphen and without consecutive hyphens"

// ValidateResourceIdentifier reports whether s is usable as an "id or slug"
// lookup key.
//
// The two shapes are validated separately rather than through one permissive
// pattern. A single pattern wide enough to admit both has to allow underscores
// and uppercase for the id shape, which silently drops every slug rule; a
// mistyped slug then reads as a well-formed identifier and surfaces as a
// not-found instead of an input error naming the rule it broke.
//
// The underscore is the discriminator, and it is unambiguous: a slug cannot
// contain one, and a prefixed id always does.
func ValidateResourceIdentifier(s string) bool {
	if strings.Contains(s, "_") {
		if len(s) > IDMaxLength {
			return false
		}
		return idRegex.MatchString(s)
	}

	return ValidateSlug(s)
}
