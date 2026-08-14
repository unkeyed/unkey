package validation

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestValidateResourceIdentifier(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input string
		want  bool
	}{
		// Id shape.
		{name: "uid-generated id", input: "pc_3ZfN2abcd", want: true},
		{name: "dev seeder readable id", input: "portal_awesome", want: true},
		{name: "dev seeder id with a hyphenated slug", input: "portal_my-team", want: true},
		{name: "dev seeder id with extra underscores", input: "ks_local_root_keys", want: true},
		{name: "mixed-case suffix", input: "pc_aB3xY9zQ1", want: true},
		{name: "digits in prefix", input: "v2app_abc123", want: true},
		{name: "id at the length limit", input: "pc_" + strings.Repeat("a", IDMaxLength-3), want: true},

		{name: "id over the length limit", input: "pc_" + strings.Repeat("a", IDMaxLength-2), want: false},
		{name: "id with no suffix", input: "pc_", want: false},
		{name: "id with no prefix", input: "_abc123", want: false},
		{name: "id with uppercase prefix", input: "PC_abc123", want: false},
		{name: "underscores only", input: "___", want: false},

		// Slug shape.
		{name: "simple slug", input: "my-portal", want: true},
		{name: "digits in slug", input: "billing2", want: true},
		{name: "slug at min length", input: "abc", want: true},
		{name: "slug at max length", input: strings.Repeat("a", 64), want: true},

		{name: "slug over max length", input: strings.Repeat("a", 65), want: false},
		{name: "slug under min length", input: "ab", want: false},
		{name: "uppercase slug", input: "My-Portal", want: false},
		{name: "leading hyphen", input: "-portal", want: false},
		{name: "trailing hyphen", input: "portal-", want: false},
		{name: "consecutive hyphens", input: "my--portal", want: false},
		{name: "empty", input: "", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			require.Equal(t, tt.want, ValidateResourceIdentifier(tt.input))
		})
	}
}

// The split exists so slug-shaped input keeps its rules. A single permissive
// pattern would accept all of these, which is the regression this guards.
func TestValidateResourceIdentifier_SlugRulesSurviveTheIdShape(t *testing.T) {
	t.Parallel()

	for _, input := range []string{"My-Portal", "my--portal", "-portal-", strings.Repeat("a", 65)} {
		require.False(t, ValidateResourceIdentifier(input),
			"%q is slug-shaped and invalid; it must not be accepted as an identifier", input)
		require.Regexp(t, `^[a-zA-Z0-9-]+$`, input, "test input must not contain an underscore")
	}
}
