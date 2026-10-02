package urn

import "fmt"

const ratelimitOverridePathFormat = "projects/%s/ratelimits/namespaces/%s/overrides/%s"

var ratelimitOverridePattern = compileResourcePattern(ratelimitOverridePathFormat)

// RatelimitOverride is a rate limit override resource path.
type RatelimitOverride struct {
	WorkspaceID string
	ProjectID   string
	NamespaceID string
	OverrideID  string
}

// String returns the complete URN for this rate limit override.
func (r RatelimitOverride) String() string {
	return r.V1().String()
}

// ParseRatelimitOverride parses:
//
//	unkey:v1:ws_123:projects/proj_123/ratelimits/namespaces/ns_123/overrides/override_123
//
// into:
//
//	RatelimitOverride{
//		WorkspaceID: "ws_123",
//		ProjectID:   "proj_123",
//		NamespaceID: "ns_123",
//		OverrideID:  "override_123",
//	}
//
// Resource ID positions may contain "*".
func ParseRatelimitOverride(urn string) (RatelimitOverride, error) {
	matches := ratelimitOverridePattern.FindStringSubmatch(urn)
	if matches == nil {
		return RatelimitOverride{}, fmt.Errorf("%w: resource does not match rate limit override", ErrInvalidResourceName)
	}
	return RatelimitOverride{
		WorkspaceID: matches[1],
		ProjectID:   matches[2],
		NamespaceID: matches[3],
		OverrideID:  matches[4],
	}, nil
}

// V1 returns this rate limit override as a parsed v1 resource name.
func (r RatelimitOverride) V1() V1 {
	return V1{
		WorkspaceID: r.WorkspaceID,
		Resource:    fmt.Sprintf(ratelimitOverridePathFormat, r.ProjectID, r.NamespaceID, r.OverrideID),
	}
}
