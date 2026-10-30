package ipallowlist_test

import (
	"fmt"
	"net/netip"
	"slices"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/auth/ipallowlist"
	"github.com/unkeyed/unkey/pkg/fuzz"
)

// FuzzAllowlistMatchesAddressSets checks authorization against exhaustive address
// sets, independent of prefix sorting or merging. Two separate 256-address
// spaces exercise overlapping ranges, gaps, and IPv4/IPv6 isolation.
func FuzzAllowlistMatchesAddressSets(f *testing.F) {
	fuzz.Seed(f)
	f.Fuzz(func(t *testing.T, data []byte) {
		c := fuzz.New(t, data)
		parentEntries, parentSet := consumeAddressSet(c)
		childEntries, childSet := consumeAddressSet(c)
		parent, err := ipallowlist.Parse(parentEntries)
		require.NoError(t, err)
		child, err := ipallowlist.Parse(childEntries)
		require.NoError(t, err)
		parentBefore, childBefore := slices.Clone(parent), slices.Clone(child)
		subset := true
		for i := range parentSet {
			if childSet[i] && !parentSet[i] {
				subset = false
			}
			address := netip.MustParseAddr(addressSetEntry(i))
			require.Equal(t, parentSet[i], parent.Allows(address), "parent=%v address=%s", parentEntries, address)
			require.Equal(t, childSet[i], child.Allows(address), "child=%v address=%s", childEntries, address)
			if i < 256 {
				mapped := netip.MustParseAddr("::ffff:" + address.String())
				require.Equal(t, parentSet[i], parent.Allows(mapped))
				require.Equal(t, childSet[i], child.Allows(mapped))
			}
		}
		// Empty means unrestricted, not just the two modeled spaces.
		if len(childEntries) == 0 && len(parentEntries) != 0 {
			subset = false
		}
		require.Equal(t, subset, child.IsSubsetOf(parent), "parent=%v child=%v", parentEntries, childEntries)
		require.Equal(t, parentBefore, parent)
		require.Equal(t, childBefore, child)
		for _, list := range []ipallowlist.Allowlist{parent, child} {
			for _, outside := range []netip.Addr{netip.MustParseAddr("192.0.3.1"), netip.MustParseAddr("2001:db8::101"), {}} {
				require.Equal(t, len(list) == 0, list.Allows(outside))
			}
			roundTrip, err := ipallowlist.Parse(list.Strings())
			require.NoError(t, err)
			require.Equal(t, list, roundTrip)
		}
	})
}

func consumeAddressSet(c *fuzz.Consumer) ([]string, [512]bool) {
	entries := make([]string, 0)
	var addresses [512]bool
	count := int(c.Uint8() % 17)
	if count == 0 {
		for i := range addresses {
			addresses[i] = true
		}
	}
	for range count {
		family := int(c.Uint8() % 2)
		host := int(c.Uint8())
		hostBits := int(c.Uint8() % 9)
		size := 1 << hostBits
		start := host / size * size
		for i := start; i < start+size; i++ {
			addresses[family*256+i] = true
		}
		entry := addressSetEntry(family*256 + host)
		bits := 32 - hostBits
		if family == 1 {
			bits = 128 - hostBits
		} else if c.Bool() {
			entry = "::ffff:" + entry
			bits += 96
		}
		entries = append(entries, fmt.Sprintf(" %s/%d ", entry, bits))
	}
	return entries, addresses
}

func addressSetEntry(i int) string {
	if i < 256 {
		return fmt.Sprintf("192.0.2.%d", i)
	}
	return fmt.Sprintf("2001:db8::%x", i-256)
}

// FuzzAllowlistPartitions checks both sides of every possible prefix boundary,
// including /0 and host prefixes. Only the mapped IPv6 half has no native IPv6
// members: Parse converts it to IPv4, whereas Allows unmaps query addresses.
func FuzzAllowlistPartitions(f *testing.F) {
	fuzz.Seed(f)
	for _, width := range []int{32, 128} {
		for bits := range width {
			seed := make([]byte, width/8+2)
			if width == 128 {
				seed[0] = 1
			}
			seed[len(seed)-1] = byte(bits)
			f.Add(seed)
		}
	}
	f.Add([]byte{1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 255, 254, 0, 0, 0, 0, 95})
	f.Fuzz(func(t *testing.T, data []byte) {
		c := fuzz.New(t, data)
		width := 32
		if c.Bool() {
			width = 128
		}
		address, ok := netip.AddrFromSlice(c.BytesN(width / 8))
		require.True(t, ok)
		if address.Is4In6() {
			t.Skip("mapped prefixes have separate normalization rules")
		}
		bits := int(c.Uint8()) % width
		prefix := netip.PrefixFrom(address, bits).Masked()
		left := netip.PrefixFrom(prefix.Addr(), bits+1)
		rightBytes := slices.Clone(prefix.Addr().AsSlice())
		rightBytes[bits/8] |= 1 << (7 - bits%8)
		rightAddress, ok := netip.AddrFromSlice(rightBytes)
		require.True(t, ok)
		right := netip.PrefixFrom(rightAddress, bits+1)
		whole, err := ipallowlist.Parse([]string{prefix.String()})
		require.NoError(t, err)
		parts, err := ipallowlist.Parse([]string{right.String(), left.String()})
		require.NoError(t, err)
		half, err := ipallowlist.Parse([]string{left.String()})
		require.NoError(t, err)
		require.True(t, whole.IsSubsetOf(parts))
		if rightAddress.Is4In6() {
			require.True(t, whole.IsSubsetOf(half))
			require.False(t, parts.IsSubsetOf(whole))
			return
		}
		require.True(t, parts.IsSubsetOf(whole))
		require.False(t, whole.IsSubsetOf(half))
		require.True(t, half.Allows(left.Addr()))
		require.False(t, half.Allows(right.Addr()))
	})
}

// FuzzParseRejectsAtomically checks that untrusted entries cannot leave a partial
// policy on error and that accepted policies retain their meaning when stored.
func FuzzParseRejectsAtomically(f *testing.F) {
	fuzz.Seed(f)
	for _, entry := range []string{"192.0.2.1", "::1", "::ffff:192.0.2.1/95", "fe80::1%eth0", "", "\xff", " 192.0.2.129/24 "} {
		seed := []byte{byte(len(entry) >> 8), byte(len(entry))}
		f.Add(append(seed, []byte(entry)...))
	}
	f.Fuzz(func(t *testing.T, data []byte) {
		c := fuzz.New(t, data)
		entry := c.String()
		list, err := ipallowlist.Parse([]string{entry})
		combined, combinedErr := ipallowlist.Parse([]string{"192.0.2.1", entry, "2001:db8::1"})
		if err != nil {
			require.Nil(t, list)
			require.Error(t, combinedErr)
			require.Nil(t, combined)
			return
		}
		require.NoError(t, combinedErr)
		require.Len(t, combined, 3)
		require.Len(t, list, 1)
		require.True(t, list[0].IsValid())
		require.False(t, list[0].Addr().Is4In6())
		require.Equal(t, list[0].Masked(), list[0])
		roundTrip, err := ipallowlist.Parse(list.Strings())
		require.NoError(t, err)
		require.Equal(t, list, roundTrip)
	})
}
