package validation

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestValidateSlug_ProtectsPortalSlugBoundary guarantees that portal slugs stay
// URL-safe and human-readable before route handlers use them for lookup.
func TestValidateSlug_ProtectsPortalSlugBoundary(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		slug string
		want bool
	}{
		{name: "minimum length", slug: "abc", want: true},
		{name: "maximum length", slug: strings.Repeat("a", SlugMaxLength), want: true},
		{name: "single hyphen inside", slug: "acme-prod", want: true},
		{name: "too short", slug: "ab", want: false},
		{name: "too long", slug: strings.Repeat("a", SlugMaxLength+1), want: false},
		{name: "uppercase", slug: "Acme", want: false},
		{name: "starts with hyphen", slug: "-acme", want: false},
		{name: "ends with hyphen", slug: "acme-", want: false},
		{name: "consecutive hyphens", slug: "acme--prod", want: false},
		{name: "underscore", slug: "acme_prod", want: false},
		{name: "space", slug: "acme prod", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			require.Equal(t, tt.want, ValidateSlug(tt.slug))
		})
	}
}
