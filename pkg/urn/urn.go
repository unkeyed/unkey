package urn

import (
	"errors"
	"fmt"
	"strings"
)

const (
	prefix            = "unkey"
	version           = "v1"
	resourceIDSegment = "{id}"
)

// ErrInvalidResourceName is returned when a resource name cannot be parsed.
var ErrInvalidResourceName = errors.New("invalid resource name")

// resourcePathShapes defines every public v1 resource.
// resourceIDSegment marks a segment that accepts one concrete ID or "*".
var resourcePathShapes = [][]string{
	{"github", "apps", resourceIDSegment},
	{"projects", resourceIDSegment},
	{"projects", resourceIDSegment, "portals", resourceIDSegment},
	{"projects", resourceIDSegment, "portals", resourceIDSegment, "sessions", resourceIDSegment},
	{"projects", resourceIDSegment, "apps", resourceIDSegment},
	{"projects", resourceIDSegment, "apps", resourceIDSegment, "environments", resourceIDSegment},
	{"projects", resourceIDSegment, "apps", resourceIDSegment, "environments", resourceIDSegment, "deployments", resourceIDSegment},
	{"projects", resourceIDSegment, "apps", resourceIDSegment, "environments", resourceIDSegment, "deployments", resourceIDSegment, "logs"},
	{"projects", resourceIDSegment, "apps", resourceIDSegment, "environments", resourceIDSegment, "domains", resourceIDSegment},
	{"projects", resourceIDSegment, "apps", resourceIDSegment, "environments", resourceIDSegment, "variables", resourceIDSegment},
	{"projects", resourceIDSegment, "apps", resourceIDSegment, "environments", resourceIDSegment, "gateway", "logs"},
	{"projects", resourceIDSegment, "apps", resourceIDSegment, "environments", resourceIDSegment, "gateway", "policies", resourceIDSegment},
	{"projects", resourceIDSegment, "identities", resourceIDSegment},
	{"projects", resourceIDSegment, "keyspaces", resourceIDSegment},
	{"projects", resourceIDSegment, "keyspaces", resourceIDSegment, "logs"},
	{"projects", resourceIDSegment, "keyspaces", resourceIDSegment, "keys", resourceIDSegment},
	{"projects", resourceIDSegment, "ratelimits", "namespaces", resourceIDSegment},
	{"projects", resourceIDSegment, "ratelimits", "namespaces", resourceIDSegment, "logs"},
	{"projects", resourceIDSegment, "ratelimits", "namespaces", resourceIDSegment, "overrides", resourceIDSegment},
	{"projects", resourceIDSegment, "rbac", "roles", resourceIDSegment},
	{"projects", resourceIDSegment, "rbac", "permissions", resourceIDSegment},
}

// resourceContainerPathShapes defines path containers that can anchor a
// descendant pattern but cannot identify a concrete resource.
var resourceContainerPathShapes = [][]string{
	{"projects", resourceIDSegment, "apps", resourceIDSegment, "environments", resourceIDSegment, "gateway"},
	{"projects", resourceIDSegment, "rbac"},
}

// V1 is a parsed v1 Unkey resource name.
type V1 struct {
	WorkspaceID string
	Resource    string
}

// String returns the canonical v1 resource-name string.
func (v V1) String() string {
	return fmt.Sprintf("%s:%s:%s:%s", prefix, version, v.WorkspaceID, v.Resource)
}

// Covers reports whether the receiver, treated as a resource-name pattern,
// covers the target resource name. The workspace must match exactly. In the
// resource path, "*" matches exactly one path segment and a trailing "**"
// matches the base path and all descendants. A concrete resource name covers
// only itself. The standalone path "**" is the global pattern covering every
// resource in the workspace.
//
// These patterns cover
// unkey:v1:ws_1:projects/proj_1/keyspaces/ks_1/keys/k_1:
//
//	unkey:v1:ws_1:projects/proj_1/keyspaces/ks_1/keys/k_1   (itself)
//	unkey:v1:ws_1:projects/proj_1/keyspaces/*/keys/*
//	unkey:v1:ws_1:projects/proj_1/keyspaces/ks_1/**
//	unkey:v1:ws_1:**
//
// and these do not:
//
//	unkey:v1:ws_1:projects/proj_1/keyspaces/ks_1       (not the same resource)
//	unkey:v1:ws_1:projects/proj_1/keyspaces/*          ("*" matches one segment)
//	unkey:v1:ws_1:projects/proj_2/**                   (different project)
//	unkey:v1:ws_2:**                                   (different workspace)
func (v V1) Covers(target V1) bool {
	if v.WorkspaceID != target.WorkspaceID {
		return false
	}
	patternSegments := strings.Split(v.Resource, "/")
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

// ParseV1 parses a v1 resource name of the form
//
//	unkey:v1:{workspace_id}:{resource_path}
//
// The resource path may be concrete or a pattern: "*" matches exactly one ID
// segment and a trailing "/**" matches the base path and all descendants. The
// resource path must match the public v1 resource catalog.
//
// Accepted:
//
//	unkey:v1:ws_123:projects/proj_1/keyspaces/ks_1/keys/k_1
//	unkey:v1:ws_123:projects/proj_1/keyspaces/*/keys/*
//	unkey:v1:ws_123:projects/proj_1/**
//	unkey:v1:ws_123:**
//
// Rejected with [ErrInvalidResourceName]:
//
//	unkey:v1:ws_123                                      missing resource path
//	unkey:v1:ws_123:keyspaces/ks_1                       missing project
//	unkey:v1:ws_123:projects/*/keyspaces/ks_1            narrows after "*"
//	unkey:v1:ws_123:projects/proj_1/keyspaces/ks_*       partial wildcard
//	unkey:v1:ws_123:projects/**/keyspaces/*              middle "**"
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

// validateResourcePath enforces five invariants on every "/"-separated path
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
//  5. The path matches the public catalog and never narrows an ID after "*".
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

	if len(segments) == 1 && segments[0] == "**" {
		return nil
	}

	descendantPattern := segments[len(segments)-1] == "**"
	if descendantPattern {
		segments = segments[:len(segments)-1]
	}

	for _, shape := range resourcePathShapes {
		if resourcePathMatchesShape(segments, shape) {
			return nil
		}
	}
	if descendantPattern {
		for _, shape := range resourceContainerPathShapes {
			if resourcePathMatchesShape(segments, shape) {
				return nil
			}
		}
	}

	return errors.New("must match a canonical resource path")
}

// resourcePathMatchesShape reports whether path matches one catalog shape.
// After an ID wildcard, all descendant ID segments must also use wildcards.
func resourcePathMatchesShape(path []string, shape []string) bool {
	if len(path) != len(shape) {
		return false
	}

	wildcardIDSeen := false
	for i, shapeSegment := range shape {
		if shapeSegment != resourceIDSegment {
			if path[i] != shapeSegment {
				return false
			}
			continue
		}

		if path[i] == "*" {
			wildcardIDSeen = true
			continue
		}
		if wildcardIDSeen {
			return false
		}
	}

	return true
}
