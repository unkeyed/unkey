package ipallowlist_test

import (
	"encoding/json"
	"fmt"
	"net/netip"
	"slices"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/auth/ipallowlist"
)

func TestAllows(t *testing.T) {
	for _, test := range []struct {
		name    string
		entries []string
		allowed []string
		denied  []string
	}{
		{"unrestricted", nil, []string{"0.0.0.0", "255.255.255.255", "::", "ffff:ffff:ffff:ffff:ffff:ffff:ffff:ffff"}, nil},
		{"empty", []string{}, []string{"192.0.2.1", "::1", "fe80::1%eth0"}, nil},
		{"explicit universes", []string{"0.0.0.0/0", "::/0"}, []string{"192.0.2.1", "::ffff:192.0.2.1", "::1"}, []string{"fe80::1%eth0"}},
		{"IPv4 host", []string{"192.0.2.7"}, []string{"192.0.2.7", "::ffff:192.0.2.7"}, []string{"192.0.2.6", "192.0.2.8", "::192.0.2.7", "64:ff9b::c000:207"}},
		{"IPv4 network endpoints", []string{"192.0.3.255/23"}, []string{"192.0.2.0", "192.0.2.255", "192.0.3.0", "192.0.3.255"}, []string{"192.0.1.255", "192.0.4.0"}},
		{"IPv4 minimum pair", []string{"0.0.0.1/31"}, []string{"0.0.0.0", "0.0.0.1"}, []string{"0.0.0.2", "255.255.255.255"}},
		{"IPv4 maximum pair", []string{"255.255.255.255/31"}, []string{"255.255.255.254", "255.255.255.255"}, []string{"255.255.255.253", "0.0.0.0"}},
		{"IPv4 universe", []string{"255.255.255.255/0"}, []string{"0.0.0.0", "128.0.0.0", "255.255.255.255", "::ffff:192.0.2.1"}, []string{"::", "2001:db8::1"}},
		{"IPv6 host", []string{"2001:db8::7/128"}, []string{"2001:db8::7"}, []string{"2001:db8::6", "2001:db8::8", "192.0.2.7", "2001:db8::7%eth0"}},
		{"IPv6 carry", []string{"2001:db8:0:1::1/63"}, []string{"2001:db8::", "2001:db8:0:1:ffff:ffff:ffff:ffff"}, []string{"2001:db7:ffff:ffff:ffff:ffff:ffff:ffff", "2001:db8:0:2::"}},
		{"IPv6 minimum pair", []string{"::1/127"}, []string{"::", "::1"}, []string{"::2", "ffff:ffff:ffff:ffff:ffff:ffff:ffff:ffff"}},
		{"IPv6 maximum pair", []string{"ffff:ffff:ffff:ffff:ffff:ffff:ffff:ffff/127"}, []string{"ffff:ffff:ffff:ffff:ffff:ffff:ffff:fffe", "ffff:ffff:ffff:ffff:ffff:ffff:ffff:ffff"}, []string{"ffff:ffff:ffff:ffff:ffff:ffff:ffff:fffd", "::"}},
		{"IPv6 universe excludes mapped IPv4", []string{"::/0"}, []string{"::", "ffff:ffff:ffff:ffff:ffff:ffff:ffff:ffff"}, []string{"192.0.2.1", "::ffff:192.0.2.1"}},
		{"mapped subnet", []string{"::ffff:192.0.2.129/121"}, []string{"192.0.2.128", "::ffff:192.0.2.255"}, []string{"192.0.2.127", "192.0.3.0"}},
		{"mapped universe", []string{"::ffff:192.0.2.1/96"}, []string{"0.0.0.0", "255.255.255.255", "::ffff:0.0.0.0"}, []string{"::", "2001:db8::1"}},
		{"mapped lower half", []string{"::ffff:0:0/97"}, []string{"0.0.0.0", "::ffff:127.255.255.255"}, []string{"128.0.0.0", "::ffff:255.255.255.255"}},
		{"mapped upper half", []string{"::ffff:128.0.0.1/97"}, []string{"128.0.0.0", "::ffff:255.255.255.255"}, []string{"0.0.0.0", "::ffff:127.255.255.255"}},
		{"native mapped boundary", []string{"::fffe:0:0/95"}, []string{"::fffe:0:0", "::fffe:ffff:ffff"}, []string{"::ffff:0.0.0.0", "0.0.0.0"}},
		{"link local zone", []string{"fe80::/10"}, []string{"fe80::1"}, []string{"fe80::1%eth0"}},
		{"mixed list and gap", []string{"192.0.2.0/26", "2001:db8::1", "192.0.2.128/25"}, []string{"192.0.2.0", "2001:db8::1", "192.0.2.255"}, []string{"192.0.2.64", "192.0.2.127", "2001:db8::2"}},
	} {
		t.Run(test.name, func(t *testing.T) {
			list, err := ipallowlist.Parse(test.entries)
			require.NoError(t, err)
			for _, address := range test.allowed {
				require.True(t, list.Allows(netip.MustParseAddr(address)), address)
			}
			for _, address := range test.denied {
				require.False(t, list.Allows(netip.MustParseAddr(address)), address)
			}
			require.Equal(t, len(test.entries) == 0, list.Allows(netip.Addr{}))
		})
	}
}

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
		{"broader IPv6 prefix is conservatively rejected", []string{"::fffe:0:0/96"}, []string{"::fffe:0:0/95"}, false},
		{"explicit mapped network adds IPv4 access", []string{"::fffe:0:0/95"}, []string{"::ffff:0:0/96", "::fffe:0:0/96"}, false},
		{"deep partitions", []string{"192.0.2.224/27", "192.0.2.0/25", "192.0.2.208/28", "192.0.2.192/28", "192.0.2.128/26"}, []string{"192.0.2.0/24"}, true},
		{"duplicates cannot fill gaps", []string{"192.0.2.0/25", "192.0.2.0/25"}, []string{"192.0.2.0/24"}, false},
		{"overlap with equal starts", []string{"192.0.2.64/26", "192.0.2.128/25", "192.0.2.0/27", "192.0.2.0/25", "192.0.2.192/26"}, []string{"192.0.2.0/24"}, true},
		{"contained prefix cannot obstruct cascade", []string{"192.0.2.0/25", "192.0.2.128/27", "192.0.2.128/26", "192.0.2.192/26"}, []string{"192.0.2.0/24"}, true},
		{"adjacent non siblings", []string{"192.0.2.64/26", "192.0.2.128/26"}, []string{"192.0.2.0/24"}, false},
		{"gap outside children", []string{"192.0.2.0/26", "192.0.2.128/25"}, []string{"192.0.2.192/26", "192.0.2.0/27"}, true},
		{"later child outside parent", []string{"192.0.2.0/25", "192.0.2.128/25"}, []string{"192.0.2.0/24", "192.0.3.1"}, false},
		{"mapped and native siblings", []string{"::ffff:192.0.2.128/121", "192.0.2.0/25"}, []string{"192.0.2.0/24"}, true},
		{"mixed universes", []string{"8000::/1", "128.0.0.0/1", "::/1", "0.0.0.0/1"}, []string{"::/0", "0.0.0.0/0"}, true},
		{"universes do not permit clearing", []string{"0.0.0.0/0", "::/0"}, nil, false},
		{"IPv6 64 bit cascade", []string{"2001:db8:0:3:8000::/65", "2001:db8:0:2::/64", "2001:db8:0:3::/65"}, []string{"2001:db8:0:2::/63"}, true},
		{"IPv6 64 bit gap", []string{"2001:db8:0:3:8000::/65", "2001:db8:0:2::/64"}, []string{"2001:db8:0:2::/63"}, false},
		{"IPv4 does not fill IPv6 gap", []string{"2001:db8::/125", "2001:db8::8/126", "0.0.0.0/0"}, []string{"2001:db8::/124"}, false},
		{"maximum IPv4 siblings", []string{"255.255.255.255", "255.255.255.254"}, []string{"255.255.255.254/31"}, true},
		{"maximum IPv6 siblings", []string{"ffff:ffff:ffff:ffff:ffff:ffff:ffff:ffff", "ffff:ffff:ffff:ffff:ffff:ffff:ffff:fffe"}, []string{"ffff:ffff:ffff:ffff:ffff:ffff:ffff:fffe/127"}, true},
	} {
		t.Run(test.name, func(t *testing.T) {
			parent, err := ipallowlist.Parse(test.parent)
			require.NoError(t, err)
			child, err := ipallowlist.Parse(test.child)
			require.NoError(t, err)
			parentBefore, childBefore := slices.Clone(parent), slices.Clone(child)
			require.Equal(t, test.allowed, child.IsSubsetOf(parent))
			require.Equal(t, parentBefore, parent)
			require.Equal(t, childBefore, child)
		})
	}
}

func TestParseNormalizesWithoutChangingOrder(t *testing.T) {
	for _, test := range []struct{ input, expected string }{
		{" \t192.0.2.129/24\n", "192.0.2.0/24"},
		{"\u00a0192.0.2.1\u2003", "192.0.2.1"},
		{"192.0.2.1/32", "192.0.2.1"},
		{"2001:0DB8:0000:0000:0000:0000:0000:0001/128", "2001:db8::1"},
		{"2001:DB8:ABCD:1234::1/48", "2001:db8:abcd::/48"},
		{"::ffff:192.0.2.129", "192.0.2.129"},
		{"::FFFF:C000:0281/120", "192.0.2.0/24"},
		{"::ffff:192.0.2.129/128", "192.0.2.129"},
		{"::ffff:192.0.2.129/96", "0.0.0.0/0"},
		{"::ffff:127.255.255.255/97", "0.0.0.0/1"},
		{"::ffff:128.0.0.1/97", "128.0.0.0/1"},
		{"::192.0.2.129/128", "::c000:281"},
		{"64:ff9b::192.0.2.129/128", "64:ff9b::c000:281"},
		{"255.255.255.255/0", "0.0.0.0/0"},
		{"ffff:ffff:ffff:ffff:ffff:ffff:ffff:ffff/0", "::/0"},
		{"2001:db8::1:443", "2001:db8::1:443"},
	} {
		t.Run(test.input, func(t *testing.T) {
			entries := []string{test.input, "198.51.100.7", test.input}
			before := slices.Clone(entries)
			list, err := ipallowlist.Parse(entries)
			require.NoError(t, err)
			require.Equal(t, before, entries)
			require.Equal(t, []string{test.expected, "198.51.100.7", test.expected}, list.Strings())
			roundTrip, err := ipallowlist.Parse(list.Strings())
			require.NoError(t, err)
			require.Equal(t, list, roundTrip)
		})
	}
}

func TestParseRejectsInvalidEntriesAtomically(t *testing.T) {
	for _, entry := range []string{
		"", " \t\r\n", "\xff", "192.0.2.1\x00", "\u200b192.0.2.1", "１９２.0.2.1",
		"192.0. 2.1", "192.0.2.1 /24", "192.0.2.1/ 24", "192.0.2.1/", "192.0.2.1/-1", "192.0.2.1/+24",
		"192.0.2.1/024", "192.0.2.1/0x18", "192.0.2.1/999999999999999999999999999999", "192.0.2.1/24/32",
		"192.0.2.1/33", "::/129", "::/-1", "::/", "192.000.2.1", "0177.0.0.1", "127.1", "2130706433",
		"0x7f000001", "+127.0.0.1", "256.0.0.1", "192.0.2.1:443", "[2001:db8::1]:443", "[2001:db8::1]",
		"2001::db8::1", "1:2:3:4:5:6:7", "1:2:3:4:5:6:7:8:9", "fe80::1%eth0", "fe80::1%3", "fe80::1%",
		"fe80::1%eth0/64", "::ffff:192.0.2.1%eth0", "::ffff:192.0.2.1%eth0/120", "::ffff:192.0.2.1/95",
	} {
		t.Run(entry, func(t *testing.T) {
			for position := range 3 {
				entries := slices.Insert([]string{"0.0.0.0/0", "::/0"}, position, entry)
				before := slices.Clone(entries)
				list, err := ipallowlist.Parse(entries)
				require.Error(t, err)
				require.Nil(t, list)
				require.Equal(t, before, entries)
			}
		})
	}
}

func TestStringsEmptyIsJSONArray(t *testing.T) {
	for _, entries := range [][]string{nil, {}} {
		list, err := ipallowlist.Parse(entries)
		require.NoError(t, err)
		encoded, err := json.Marshal(list.Strings())
		require.NoError(t, err)
		require.Equal(t, "[]", string(encoded))
	}
}

// TestAllowsEveryPrefixBoundary checks inclusive endpoints and their neighbors
// at every prefix length, without using netip prefix operations as the oracle.
func TestAllowsEveryPrefixBoundary(t *testing.T) {
	for _, width := range []int{32, 128} {
		for bits := 0; bits <= width; bits++ {
			for _, upper := range []bool{false, true} {
				t.Run(fmt.Sprintf("width=%d/bits=%d/upper=%t", width, bits, upper), func(t *testing.T) {
					firstBytes, lastBytes := make([]byte, width/8), make([]byte, width/8)
					for bit := range width {
						mask := byte(1 << (7 - bit%8))
						if upper {
							lastBytes[bit/8] |= mask
							if bit < bits {
								firstBytes[bit/8] |= mask
							}
						} else if bit >= bits {
							lastBytes[bit/8] |= mask
						}
					}
					first, ok := netip.AddrFromSlice(firstBytes)
					require.True(t, ok)
					last, ok := netip.AddrFromSlice(lastBytes)
					require.True(t, ok)
					list, err := ipallowlist.Parse([]string{fmt.Sprintf("%s/%d", first, bits)})
					require.NoError(t, err)
					require.True(t, list.Allows(first))
					// The last address of ::/80 is mapped, so Allows treats it as IPv4.
					require.Equal(t, !last.Is4In6(), list.Allows(last))
					require.False(t, list.Allows(first.Prev()))
					require.False(t, list.Allows(last.Next()))
				})
			}
		}
	}
}

func TestSubsetRequiresEveryAddressInDeepPartition(t *testing.T) {
	for _, format := range []string{"192.0.2.%d", "2001:db8::%x"} {
		t.Run(format, func(t *testing.T) {
			entries := make([]string, 0, 256)
			for i := 255; i >= 0; i-- {
				entries = append(entries, fmt.Sprintf(format, i))
			}
			bits := 24
			if format == "2001:db8::%x" {
				bits = 120
			}
			whole, err := ipallowlist.Parse([]string{fmt.Sprintf(format, 0) + fmt.Sprintf("/%d", bits)})
			require.NoError(t, err)
			complete, err := ipallowlist.Parse(entries)
			require.NoError(t, err)
			require.True(t, whole.IsSubsetOf(complete))
			for _, index := range []int{0, 128, 255} {
				gap := slices.Delete(slices.Clone(entries), index, index+1)
				incomplete, err := ipallowlist.Parse(gap)
				require.NoError(t, err)
				require.False(t, whole.IsSubsetOf(incomplete), "missing %s", entries[index])
			}
		})
	}
}
