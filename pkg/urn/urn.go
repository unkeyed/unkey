package urn

import (
	"errors"
	"fmt"
	"strings"
)

const (
	prefix  = "unkey"
	version = "v1"

	// ResourceV1Format is the canonical v1 resource-name format.
	ResourceV1Format = "unkey:v1:%s:%s"
)

// ErrInvalidResourceName is returned when a resource name cannot be parsed.
var ErrInvalidResourceName = errors.New("invalid resource name")

// V1 is a parsed v1 Unkey resource name.
type V1 struct {
	WorkspaceID string
	Resource    string
}

// String returns the canonical v1 resource-name string.
func (r V1) String() string {
	return ResourceV1(r.WorkspaceID, r.Resource)
}

// Covers reports whether the receiver, treated as a resource-name pattern,
// covers the target resource name. The workspace must match exactly. In the
// resource path, "*" matches exactly one path segment and a trailing "**"
// matches the base path and all descendants. A concrete resource name covers
// only itself. The standalone path "**" is the global pattern covering every
// resource in the workspace; the standalone path "*" covers only resources
// with single-segment paths.
//
// These patterns cover unkey:v1:ws_1:keyspaces/ks_1/keys/k_1:
//
//	unkey:v1:ws_1:keyspaces/ks_1/keys/k_1   (itself)
//	unkey:v1:ws_1:keyspaces/*/keys/*
//	unkey:v1:ws_1:keyspaces/ks_1/**
//	unkey:v1:ws_1:**
//
// and these do not:
//
//	unkey:v1:ws_1:keyspaces/ks_1            (concrete name, not the same resource)
//	unkey:v1:ws_1:keyspaces/*               ("*" does not cross into keys/k_1)
//	unkey:v1:ws_1:*                         ("*" is one segment, not a global wildcard)
//	unkey:v1:ws_2:**                        (different workspace)
func (r V1) Covers(target V1) bool {
	if r.WorkspaceID != target.WorkspaceID {
		return false
	}
	patternSegments := strings.Split(r.Resource, "/")
	targetSegments := strings.Split(target.Resource, "/")

	if len(patternSegments) > 0 && patternSegments[len(patternSegments)-1] == "**" {
		patternSegments = patternSegments[:len(patternSegments)-1]
		return len(targetSegments) >= len(patternSegments) &&
			segmentsMatch(patternSegments, targetSegments[:len(patternSegments)])
	}

	return len(patternSegments) == len(targetSegments) &&
		segmentsMatch(patternSegments, targetSegments)
}

// segmentsMatch compares equal-length segment slices, treating "*" in the
// pattern as matching any single target segment.
func segmentsMatch(pattern []string, target []string) bool {
	for i := range pattern {
		if pattern[i] != "*" && pattern[i] != target[i] {
			return false
		}
	}
	return true
}

// ResourceV1 constructs a canonical v1 resource name.
func ResourceV1(workspaceID string, resource string) string {
	return fmt.Sprintf(ResourceV1Format, workspaceID, resource)
}

// ParseV1 parses a v1 resource name of the form
//
//	unkey:v1:{workspace_id}:{resource_path}
//
// The resource path may be concrete or a pattern: "*" matches exactly one
// path segment and a trailing "/**" matches the base path and all
// descendants. Whether a wildcard path is acceptable is the caller's concern;
// the parser only enforces the grammar.
//
// Accepted:
//
//	unkey:v1:ws_123:keyspaces/ks_1/keys/k_1    concrete resource name
//	unkey:v1:ws_123:keyspaces/*/keys/*         one wildcard per segment
//	unkey:v1:ws_123:ratelimits/**              descendant scope
//	unkey:v1:ws_123:**                         everything in the workspace
//
// Rejected with [ErrInvalidResourceName]:
//
//	unkey:v1:ws_123                            missing resource path
//	unkey:v1:ws_123:keyspaces/ks_1#read_key    "#" belongs to permissions, not URNs
//	unkey:v1:ws_123:keyspaces/ks_*             "*" must be a whole segment
//	unkey:v1:ws_123:ratelimits/**/overrides    "**" must be the last segment
func ParseV1(value string) (V1, error) {
	parts := strings.SplitN(value, ":", 4)
	if len(parts) != 4 {
		return V1{}, fmt.Errorf("%w: expected 4 colon-separated fields", ErrInvalidResourceName)
	}
	if parts[0] != prefix {
		return V1{}, fmt.Errorf("%w: prefix must be %q", ErrInvalidResourceName, prefix)
	}
	if parts[1] != version {
		return V1{}, fmt.Errorf("%w: version must be %q", ErrInvalidResourceName, version)
	}
	if err := validateWorkspaceID(parts[2]); err != nil {
		return V1{}, fmt.Errorf("%w: invalid workspace id: %v", ErrInvalidResourceName, err)
	}
	if err := validateResourcePath(parts[3]); err != nil {
		return V1{}, fmt.Errorf("%w: invalid resource path: %v", ErrInvalidResourceName, err)
	}

	return V1{
		WorkspaceID: parts[2],
		Resource:    parts[3],
	}, nil
}

// validateWorkspaceID enforces two invariants on the workspace field:
//
//  1. It is not empty.
//  2. It contains none of the reserved characters ":" (URN field separator),
//     "#" (permission action separator), and "/" (path segment separator).
func validateWorkspaceID(value string) error {
	if value == "" {
		return errors.New("must not be empty")
	}
	if strings.ContainsAny(value, ":#/") {
		return errors.New(`must not contain ":", "#", or "/"`)
	}
	return nil
}

// validateResourcePath enforces four invariants on every "/"-separated path
// segment:
//
//  1. No segment is empty. This subsumes rejecting an empty path and paths
//     with a leading, trailing, or doubled "/".
//  2. No segment contains ":" (URN field separator) or "#" (permission action
//     separator).
//  3. "*" appears only as a whole segment, never inside one, so a wildcard
//     can only ever expand to exactly one segment.
//  4. "**" appears only as the final segment, so a descendant scope cannot
//     have a suffix constraint the matcher would have to guess about.
func validateResourcePath(path string) error {
	segments := strings.Split(path, "/")
	for i, segment := range segments {
		isLastSegment := i == len(segments)-1

		switch {
		case segment == "":
			return errors.New("must not contain empty segments")
		case strings.ContainsAny(segment, ":#"):
			return errors.New(`must not contain ":" or "#"`)
		case segment == "*":
			// Single-segment wildcard, valid anywhere in the path.
		case segment == "**":
			if !isLastSegment {
				return errors.New(`"**" must be the last segment`)
			}
		case strings.Contains(segment, "*"):
			return errors.New(`"*" must be a whole segment`)
		}
	}
	return nil
}
