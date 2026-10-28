package canonicalpath

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestClean(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		path     string
		expected string
	}{
		{"unchanged", "/admin", "/admin"},
		{"trailing slash", "/admin/", "/admin"},
		{"segment parameter", "/admin;jsessionid=x", "/admin"},
		{"parameters on several segments", "/admin;x/users;y", "/admin/users"},
		{"leading duplicate slash", "//admin", "/admin"},
		{"inner duplicate slash", "/a//b", "/a/b"},
		{"parent segment", "/x/../admin", "/admin"},
		{"current segment", "/./admin", "/admin"},
		{"parameter hides parent segment", "/public/..;/admin", "/admin"},
		{"parent above root", "/../../admin", "/admin"},
		{"root", "/", "/"},
		{"empty", "", "/"},
		{"parameter only", "/;x", "/"},
		{"case preserved", "/ADMIN/", "/ADMIN"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			require.Equal(t, tt.expected, Clean(tt.path))
		})
	}
}

func TestCandidates(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		path     string
		expected []string
	}{
		{"plain path", "/admin", []string{"/admin", "/admin/"}},
		{"raw path is never a candidate", "/admin;x/", []string{"/admin", "/admin/"}},
		{"root", "/", []string{"/"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			require.Equal(t, tt.expected, Candidates(tt.path))
		})
	}
}

func TestHasEncodedSeparator(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		escaped  string
		expected bool
	}{
		{"plain path", "/admin/users", false},
		{"encoded dots", "/public/%2e%2e/admin", false},
		{"other escapes", "/files/a%20b", false},
		{"encoded slash", "/files/a%2Fb", true},
		{"lowercase encoded slash", "/files/a%2fb", true},
		{"encoded backslash", "/files/a%5Cb", true},
		{"lowercase encoded backslash", "/files/a%5cb", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			require.Equal(t, tt.expected, HasEncodedSeparator(tt.escaped))
		})
	}
}
