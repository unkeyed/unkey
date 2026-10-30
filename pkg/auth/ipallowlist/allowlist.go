package ipallowlist

import (
	"cmp"
	"fmt"
	"net/netip"
	"slices"
	"strings"
)

// Allowlist is a union of IP networks. An empty list allows every address.
// Use [Parse] to normalize addresses and network prefixes before comparison.
type Allowlist []netip.Prefix

// Parse accepts IPv4 and IPv6 addresses or CIDR ranges with optional surrounding
// whitespace. It masks host bits and converts IPv4-mapped IPv6 entries to IPv4.
// It returns nil and an error for invalid entries, zone identifiers, or mapped
// prefixes shorter than /96. An empty input returns an unrestricted allowlist.
//
// Examples, showing the parsed prefixes:
//
//	[" 192.0.2.129/24 ", "2001:db8::1"] -> [192.0.2.0/24, 2001:db8::1/128]
//	["::ffff:192.0.2.1/120"]           -> [192.0.2.0/24]
//	["::ffff:192.0.2.1/95"]            -> nil, error
//	["192.0.2.1", "invalid"]           -> nil, error (no partial policy)
//	[]                                -> [], nil (unrestricted)
func Parse(entries []string) (Allowlist, error) {
	prefixes := make(Allowlist, 0, len(entries))
	for _, entry := range entries {
		entry = strings.TrimSpace(entry)
		prefix, err := netip.ParsePrefix(entry)
		if err != nil {
			address, addressErr := netip.ParseAddr(entry)
			if addressErr != nil || address.Zone() != "" {
				return nil, fmt.Errorf("invalid IP address or CIDR range: %q", entry)
			}
			address = address.Unmap()
			prefix = netip.PrefixFrom(address, address.BitLen())
		}
		if prefix.Addr().Is4In6() {
			if prefix.Bits() < 96 {
				return nil, fmt.Errorf("IPv4-mapped CIDR range must have at least 96 prefix bits: %q", entry)
			}
			prefix = netip.PrefixFrom(prefix.Addr().Unmap(), prefix.Bits()-96)
		}
		prefixes = append(prefixes, prefix.Masked())
	}
	return prefixes, nil
}

// Allows reports whether address belongs to an allowed network. It treats
// IPv4-mapped IPv6 addresses as IPv4. An empty list allows any address, including
// an invalid address; a nonempty list rejects invalid addresses.
//
// Examples use lists constructed with [Parse]:
//
//	[192.0.2.0/24].Allows(192.0.2.255)       -> true
//	[192.0.2.0/24].Allows(192.0.3.0)         -> false
//	[192.0.2.0/24].Allows(::ffff:192.0.2.7)  -> true
//	[::/0].Allows(::ffff:192.0.2.7)          -> false (the query becomes IPv4)
//	[].Allows(any address)                 -> true
func (a Allowlist) Allows(address netip.Addr) bool {
	if len(a) == 0 {
		return true
	}
	for _, prefix := range a {
		if prefix.Contains(address.Unmap()) {
			return true
		}
	}
	return false
}

// IsSubsetOf reports whether a permits no addresses outside parent. It compares
// the union of the parent networks, so adjacent ranges can cover a larger prefix.
// An empty parent allows any list. An empty a is unrestricted and is rejected
// when parent is nonempty. Neither list is modified.
//
// Examples use lists constructed with [Parse], with the child on the left:
//
//	[192.0.2.0/25].IsSubsetOf([192.0.2.0/24])                 -> true
//	[192.0.2.0/24].IsSubsetOf([192.0.2.0/25])                 -> false
//	[192.0.2.0/24].IsSubsetOf([192.0.2.0/25, 192.0.2.128/25]) -> true
//	[].IsSubsetOf([0.0.0.0/0, ::/0])                         -> false
//	[192.0.2.0/24].IsSubsetOf([])                            -> true
func (a Allowlist) IsSubsetOf(parent Allowlist) bool {
	if len(parent) == 0 {
		return true
	}
	if len(a) == 0 {
		return false
	}
	parent = parent.union()
	for _, child := range a {
		contained := false
		for _, prefix := range parent {
			if prefix.Bits() <= child.Bits() && prefix.Contains(child.Addr()) {
				contained = true
				break
			}
		}
		if !contained {
			return false
		}
	}
	return true
}

// Strings formats single-address prefixes as IP addresses and other prefixes
// as CIDR ranges. It preserves entry order and returns a non-nil empty slice for
// an unrestricted list, so JSON encoding produces [] rather than null.
//
// Examples, showing prefix values on the left and strings on the right:
//
//	[192.0.2.7/32, 2001:db8::/64] -> ["192.0.2.7", "2001:db8::/64"]
//	[]                          -> [] (not nil)
func (a Allowlist) Strings() []string {
	entries := make([]string, 0, len(a))
	for _, prefix := range a {
		if prefix.Bits() == prefix.Addr().BitLen() {
			entries = append(entries, prefix.Addr().String())
		} else {
			entries = append(entries, prefix.String())
		}
	}
	return entries
}

// union removes contained prefixes and merges adjacent sibling prefixes without
// modifying a. This lets subset checks recognize equivalent network partitions.
// It includes mapped IPv6 space as coverage because Allows unmaps those addresses
// before comparison, so no IPv6 prefix can grant access to that space.
//
// For example, [192.0.2.128/25, 192.0.2.0/25, 192.0.2.7/32] becomes
// [192.0.2.0/24, ::ffff:0:0/96]. The /32 is redundant; the two /25 ranges
// cover the /24. The extra mapped prefix contributes no allowed addresses.
// [::fffe:0:0/96] becomes [::fffe:0:0/95] for the same reason: its added
// mapped half cannot match any address after Allows unmaps the query.
func (a Allowlist) union() Allowlist {
	prefixes := make(Allowlist, 0, len(a)+1)
	prefixes = append(prefixes, a...)
	prefixes = append(prefixes, netip.MustParsePrefix("::ffff:0:0/96"))
	slices.SortFunc(prefixes, func(a, b netip.Prefix) int {
		if order := a.Addr().Compare(b.Addr()); order != 0 {
			return order
		}
		return cmp.Compare(a.Bits(), b.Bits())
	})
	merged := prefixes[:0]
	for _, prefix := range prefixes {
		if len(merged) > 0 {
			last := merged[len(merged)-1]
			if last.Bits() <= prefix.Bits() && last.Contains(prefix.Addr()) {
				continue
			}
		}
		merged = append(merged, prefix)
		for len(merged) >= 2 {
			left, right := merged[len(merged)-2], merged[len(merged)-1]
			if left.Bits() == 0 || left.Bits() != right.Bits() {
				break
			}
			parent := netip.PrefixFrom(left.Addr(), left.Bits()-1).Masked()
			if !parent.Contains(right.Addr()) {
				break
			}
			merged = append(merged[:len(merged)-2], parent)
		}
	}
	return merged
}
