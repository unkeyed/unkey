package ipallowlist_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/auth/ipallowlist"
)

func TestSubsetUsesNetworkUnion(t *testing.T) {
	for _, test := range []struct {
		name    string
		parent  []string
		child   []string
		allowed bool
	}{
		{"IPv4 siblings", []string{"192.0.2.128/25", "192.0.2.0/25"}, []string{"192.0.2.0/24"}, true},
		{"IPv6 siblings", []string{"2001:db8:8000::/33", "2001:db8::/33"}, []string{"2001:db8::/32"}, true},
		{"nested partitions", []string{"192.0.2.0/26", "192.0.2.64/26", "192.0.2.128/25"}, []string{"192.0.2.0/24"}, true},
		{"gap", []string{"192.0.2.0/26", "192.0.2.128/25"}, []string{"192.0.2.0/24"}, false},
		{"different families", []string{"::/0"}, []string{"192.0.2.1"}, false},
		{"empty child is unrestricted", []string{"192.0.2.0/24"}, nil, false},
		{"empty parent is unrestricted", nil, []string{"192.0.2.1"}, true},
	} {
		t.Run(test.name, func(t *testing.T) {
			parent, err := ipallowlist.Parse(test.parent)
			require.NoError(t, err)
			child, err := ipallowlist.Parse(test.child)
			require.NoError(t, err)
			require.Equal(t, test.allowed, child.IsSubsetOf(parent))
		})
	}
}
