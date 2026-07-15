package main

import (
	"fmt"
	"strconv"
	"strings"
)

// semver is a minimal SemVer representation good enough for ordering release
// tags. pre holds the dot-separated pre-release identifiers and is empty for a
// stable release. Build metadata is intentionally not modelled.
type semver struct {
	major int
	minor int
	patch int
	pre   []string
}

// parseSemver parses `vMAJOR.MINOR.PATCH[-pre]`. The leading `v` is required to
// match the repo's tag convention.
func parseSemver(s string) (semver, bool) {
	zero := semver{major: 0, minor: 0, patch: 0, pre: nil}
	if !strings.HasPrefix(s, "v") {
		return zero, false
	}
	s = strings.TrimPrefix(s, "v")

	// Drop build metadata, if any.
	if i := strings.Index(s, "+"); i >= 0 {
		s = s[:i]
	}

	core := s
	var pre []string
	if i := strings.Index(s, "-"); i >= 0 {
		core = s[:i]
		if rest := s[i+1:]; rest != "" {
			pre = strings.Split(rest, ".")
		}
	}

	parts := strings.Split(core, ".")
	if len(parts) != 3 {
		return zero, false
	}
	maj, err1 := strconv.Atoi(parts[0])
	min, err2 := strconv.Atoi(parts[1])
	pat, err3 := strconv.Atoi(parts[2])
	if err1 != nil || err2 != nil || err3 != nil {
		return zero, false
	}
	return semver{major: maj, minor: min, patch: pat, pre: pre}, true
}

func formatVersion(v semver) string {
	out := fmt.Sprintf("v%d.%d.%d", v.major, v.minor, v.patch)
	if len(v.pre) > 0 {
		out += "-" + strings.Join(v.pre, ".")
	}
	return out
}

// sameBase reports whether two versions share the same MAJOR.MINOR.PATCH.
func (v semver) sameBase(o semver) bool {
	return v.major == o.major && v.minor == o.minor && v.patch == o.patch
}

// cmp orders two versions per the SemVer precedence rules. A stable release
// outranks any pre-release of the same base version.
func cmp(a, b semver) int {
	if c := intCmp(a.major, b.major); c != 0 {
		return c
	}
	if c := intCmp(a.minor, b.minor); c != 0 {
		return c
	}
	if c := intCmp(a.patch, b.patch); c != 0 {
		return c
	}
	if len(a.pre) == 0 && len(b.pre) == 0 {
		return 0
	}
	if len(a.pre) == 0 {
		return 1
	}
	if len(b.pre) == 0 {
		return -1
	}
	n := len(a.pre)
	if len(b.pre) < n {
		n = len(b.pre)
	}
	for i := 0; i < n; i++ {
		ai, an, aNum := identifier(a.pre[i])
		bi, bn, bNum := identifier(b.pre[i])
		switch {
		case aNum && bNum:
			if c := intCmp(an, bn); c != 0 {
				return c
			}
		case aNum:
			return -1 // numeric identifiers rank below alphanumeric
		case bNum:
			return 1
		default:
			if c := strings.Compare(ai, bi); c != 0 {
				return c
			}
		}
	}
	return intCmp(len(a.pre), len(b.pre))
}

func identifier(s string) (str string, num int, isNum bool) {
	if n, err := strconv.Atoi(s); err == nil {
		return s, n, true
	}
	return s, 0, false
}

func intCmp(a, b int) int {
	switch {
	case a < b:
		return -1
	case a > b:
		return 1
	default:
		return 0
	}
}

// maxStable returns the highest stable (non-pre-release) version in the set.
func maxStable(versions []semver) (semver, bool) {
	best := semver{major: 0, minor: 0, patch: 0, pre: nil}
	found := false
	for _, v := range versions {
		if len(v.pre) > 0 {
			continue
		}
		if !found || cmp(v, best) > 0 {
			best = v
			found = true
		}
	}
	return best, found
}

// bumpVersion increments a stable version by the given kind, clearing any
// pre-release identifiers.
func bumpVersion(v semver, kind bumpKind) semver {
	v.pre = nil
	switch kind {
	case bumpMajor:
		v.major++
		v.minor = 0
		v.patch = 0
	case bumpMinor:
		v.minor++
		v.patch = 0
	case bumpPatch:
		v.patch++
	default:
		v.patch++
	}
	return v
}

// nextPreNumber returns the next pre-release number for the given base version
// and label (e.g. for label "rc" with an existing -rc.2 it returns 3; with no
// matching pre-release it returns 1).
func nextPreNumber(versions []semver, target semver, label string) int {
	highest := -1
	for _, v := range versions {
		if !v.sameBase(target) || len(v.pre) == 0 || v.pre[0] != label {
			continue
		}
		num := 0
		if len(v.pre) >= 2 {
			if n, err := strconv.Atoi(v.pre[1]); err == nil {
				num = n
			}
		}
		if num > highest {
			highest = num
		}
	}
	if highest < 0 {
		return 1
	}
	return highest + 1
}
