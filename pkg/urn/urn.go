package urn

import (
	"errors"
	"fmt"
	"strings"
)

const (
	prefix  = "unkey"
	version = "v1"
)

// ErrInvalidResourceName is returned when a resource name cannot be parsed.
var ErrInvalidResourceName = errors.New("invalid resource name")

// V1 is a parsed v1 Unkey resource name.
type V1 struct {
	WorkspaceID string
	Resource    string
}

// String returns the canonical v1 resource-name string.
func (v V1) String() string {
	return fmt.Sprintf("%s:%s:%s:%s", prefix, version, v.WorkspaceID, v.Resource)
}

// Covers reports whether every resource represented by the target scope is
// also represented by the receiver. The workspace must match exactly. In each
// resource path, "*" represents any one path segment and a trailing "**"
// represents the base path and all descendants. A concrete resource name
// covers only itself. The standalone path "**" is the global scope covering
// every resource in the workspace; the standalone path "*" covers all
// single-segment resource names.
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
func (v V1) Covers(target V1) bool {
	if v.WorkspaceID != target.WorkspaceID {
		return false
	}
	coveringSegments := strings.Split(v.Resource, "/")
	targetSegments := strings.Split(target.Resource, "/")
	coveringDescendants := coveringSegments[len(coveringSegments)-1] == "**"
	targetDescendants := targetSegments[len(targetSegments)-1] == "**"

	if coveringDescendants {
		coveringSegments = coveringSegments[:len(coveringSegments)-1]
		// Resource paths are non-empty, so */** and ** denote the same scope.
		if len(coveringSegments) == 1 && coveringSegments[0] == "*" {
			coveringSegments = coveringSegments[:0]
		}
	}
	if targetDescendants {
		targetSegments = targetSegments[:len(targetSegments)-1]
		if len(targetSegments) == 1 && targetSegments[0] == "*" {
			targetSegments = targetSegments[:0]
		}
	}

	if targetDescendants && !coveringDescendants {
		return false
	}
	if coveringDescendants {
		return len(targetSegments) >= len(coveringSegments) &&
			segmentsCover(coveringSegments, targetSegments[:len(coveringSegments)])
	}

	return len(coveringSegments) == len(targetSegments) &&
		segmentsCover(coveringSegments, targetSegments)
}

// segmentsCover compares equal-length scope prefixes. A covering "*" contains
// every target segment, while a concrete segment cannot contain target "*".
func segmentsCover(covering []string, target []string) bool {
	for i := range covering {
		if covering[i] != "*" && covering[i] != target[i] {
			return false
		}
	}
	return true
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
//	unkey:v1:ws_123:projects/*/apps/app_123    concrete child below a wildcard parent
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
			continue
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
