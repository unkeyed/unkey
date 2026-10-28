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
		decoded  string
		escaped  string
		expected []string
	}{
		{"plain path", "/admin", "/admin", []string{"/admin", "/admin/"}},
		{"raw path is never a candidate", "/admin;x/", "/admin;x/", []string{"/admin", "/admin/"}},
		{"escaped slash", "/files/a/b", "/files/a%2Fb", []string{"/files/a/b", "/files/a/b/", "/files/a%2Fb", "/files/a%2Fb/"}},
		{"escaped path that cleans to the decoded one", "/admin", "/admin/", []string{"/admin", "/admin/"}},
		{"root", "/", "/", []string{"/"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			require.Equal(t, tt.expected, Candidates(tt.decoded, tt.escaped))
		})
	}
}
