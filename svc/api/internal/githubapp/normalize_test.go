package githubapp

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNormalizeRepository(t *testing.T) {
	valid := map[string]string{
		"unkeyed/unkey":                                    "unkeyed/unkey",
		"  unkeyed/unkey  ":                                "unkeyed/unkey",
		"unkeyed/unkey/":                                   "unkeyed/unkey",
		"/unkeyed/unkey":                                   "unkeyed/unkey",
		"unkeyed/unkey.git":                                "unkeyed/unkey",
		"github.com/unkeyed/unkey":                         "unkeyed/unkey",
		"www.github.com/unkeyed/unkey":                     "unkeyed/unkey",
		"https://github.com/unkeyed/unkey":                 "unkeyed/unkey",
		"http://github.com/unkeyed/unkey":                  "unkeyed/unkey",
		"https://github.com/unkeyed/unkey.git":             "unkeyed/unkey",
		"https://github.com/unkeyed/unkey/":                "unkeyed/unkey",
		"git@github.com:unkeyed/unkey.git":                 "unkeyed/unkey",
		"https://github.com/unkeyed/unkey/tree/main":       "unkeyed/unkey",
		"https://github.com/unkeyed/unkey?tab=readme":      "unkeyed/unkey",
		"https://github.com/unkeyed/unkey#readme":          "unkeyed/unkey",
		"https://github.com/unkeyed/unkey/pull/12?foo=bar": "unkeyed/unkey",
		// "github.com" appearing inside a name (not at a host boundary) is preserved.
		"my-github.com-org/repo": "my-github.com-org/repo",
		"owner/some.github.com":  "owner/some.github.com",
	}

	for in, want := range valid {
		t.Run("valid/"+in, func(t *testing.T) {
			got, err := normalizeRepository(in)
			require.NoError(t, err)
			require.Equal(t, want, got)
		})
	}

	invalid := []string{
		"",
		"   ",
		"unkeyed",
		"unkeyed/",
		"/unkeyed",
		"https://github.com/",
		"https://github.com/unkeyed",
		"git@github.com:unkeyed",
	}

	for _, in := range invalid {
		t.Run("invalid/"+in, func(t *testing.T) {
			_, err := normalizeRepository(in)
			require.Error(t, err)
		})
	}
}
