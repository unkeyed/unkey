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

func TestEncodedSeparators(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		escaped  string
		found    bool
		multiple bool
	}{
		{"plain path", "/admin/users", false, false},
		{"encoded dots", "/public/%2e%2e/admin", false, false},
		{"other escapes", "/files/a%20b", false, false},
		{"encoded slash", "/files/a%2Fb", true, false},
		{"lowercase encoded slash", "/files/a%2fb", true, false},
		{"encoded backslash", "/files/a%5Cb", true, false},
		{"lowercase encoded backslash", "/files/a%5cb", true, false},
		{"double encoded slash", "/admin%252fevil", true, true},
		{"double encoded backslash", "/admin%255Cevil", true, true},
		{"triple encoded slash", "/admin%25252Fevil", true, true},
		{"single and double encoded", "/a%2Fb%252Fc", true, true},
		{"encoded percent", "/files/100%25", false, false},
		{"encoded percent before other escape", "/files/a%2520b", false, false},
		{"trailing percent", "/files/a%", false, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			found, multiple := EncodedSeparators(tt.escaped)
			require.Equal(t, tt.found, found)
			require.Equal(t, tt.multiple, multiple)
		})
	}
}

func TestHasDotSegment(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		decoded  string
		expected bool
	}{
		{"plain path", "/admin/users", false},
		{"parent segment", "/internal/../public", true},
		{"current segment", "/./admin", true},
		{"trailing parent segment", "/admin/..", true},
		{"parent behind a parameter", "/public/..;/admin", true},
		{"dots inside a name", "/files/..x/a.b", false},
		{"three dots", "/files/...", false},
		{"root", "/", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			require.Equal(t, tt.expected, HasDotSegment(tt.decoded))
		})
	}
}
