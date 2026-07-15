package main

import (
	"fmt"
	"strconv"
	"strings"
)

// plannedTag is a service/version pair to be tagged.
type plannedTag struct {
	service string
	version string
}

func (p plannedTag) String() string {
	return fmt.Sprintf("%s/%s", p.service, p.version)
}

// isPreRelease reports whether the tag is a SemVer pre-release (e.g. -rc.1).
func (p plannedTag) isPreRelease() bool {
	return strings.Contains(p.version, "-")
}

// buildTags turns the service args into a validated, de-duplicated list of
// tags, auto-numbering versions from existing tags where none is given.
func buildTags(opts options) ([]plannedTag, error) {
	seen := map[string]bool{}
	tags := make([]plannedTag, 0, len(opts.services))

	for _, arg := range opts.services {
		service, version := splitArg(arg)
		if version == "" {
			version = opts.version
		}

		if err := validateService(service); err != nil {
			return nil, err
		}

		if version == "" {
			// No explicit version: auto-number from the service's tags.
			computed, err := nextVersion(service, opts.bump, opts.pre)
			if err != nil {
				return nil, err
			}
			version = computed
		}

		if err := validSemver(version); err != nil {
			return nil, fmt.Errorf("invalid version %q for %s: %w", version, service, err)
		}

		tag := plannedTag{service: service, version: version}
		if seen[tag.String()] {
			continue
		}
		seen[tag.String()] = true
		tags = append(tags, tag)
	}
	return tags, nil
}

// splitArg parses `service`, `service@version`, or `service/vx.y.z`.
func splitArg(arg string) (service, version string) {
	if s, v, ok := strings.Cut(arg, "@"); ok {
		return s, v
	}
	if s, v, ok := strings.Cut(arg, "/"); ok {
		return s, v
	}
	return arg, ""
}

// nextVersion computes the next version string for a service from its existing
// tags. bump selects the stable increment; pre, when set, produces the next
// pre-release (e.g. -rc.N) for that bumped version.
func nextVersion(service string, bump bumpKind, pre string) (string, error) {
	versions := listServiceVersions(service)
	latest, ok := maxStable(versions)
	if !ok {
		return "", fmt.Errorf("no existing version tags for %q; pass an explicit version for the first release (e.g. %s@v0.1.0)", service, service)
	}

	target := bumpVersion(latest, bump)
	if pre == "" {
		return formatVersion(target), nil
	}

	next := nextPreNumber(versions, target, pre)
	target.pre = []string{pre, strconv.Itoa(next)}
	return formatVersion(target), nil
}

// tagNames returns the string form of each tag.
func tagNames(tags []plannedTag) []string {
	names := make([]string, len(tags))
	for i, t := range tags {
		names[i] = t.String()
	}
	return names
}

// hasStableTag reports whether any planned tag is a stable (non-pre-release)
// version.
func hasStableTag(tags []plannedTag) bool {
	for _, t := range tags {
		if !t.isPreRelease() {
			return true
		}
	}
	return false
}

// uniqueServices returns the distinct services in the planned tags, preserving
// first-seen order.
func uniqueServices(tags []plannedTag) []string {
	seen := map[string]bool{}
	var out []string
	for _, t := range tags {
		if !seen[t.service] {
			seen[t.service] = true
			out = append(out, t.service)
		}
	}
	return out
}
