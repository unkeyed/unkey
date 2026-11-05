package clidevice

import (
	"fmt"
	"net/url"
	"slices"
	"strings"

	tokenjwt "github.com/unkeyed/unkey/pkg/jwt"
)

// ResourcePolicy decides which RFC 8707 resource indicators a client may
// request. Any URL on an allowed origin is a resource, so the MCP server can
// add or rename mounts without an API config change. The chosen resource is
// bound into the access token audience next to the API's own audience.
type ResourcePolicy struct {
	origins  []string
	fallback string
}

// NewResourcePolicy allows URLs on the given origins and uses fallback for
// clients that send no resource. A client that omits RFC 8707 would otherwise
// get a token no MCP server accepts.
func NewResourcePolicy(origins []string, fallback string) (ResourcePolicy, error) {
	if len(origins) == 0 {
		return ResourcePolicy{}, fmt.Errorf("at least one oauth resource origin is required")
	}
	policy := ResourcePolicy{origins: make([]string, 0, len(origins)), fallback: ""}
	for i, raw := range origins {
		parsed, err := url.Parse(raw)
		if err != nil || (parsed.Scheme != "https" && parsed.Scheme != "http") || parsed.Host == "" ||
			parsed.User != nil || strings.Trim(parsed.Path, "/") != "" || parsed.RawQuery != "" || strings.Contains(raw, "#") {
			return ResourcePolicy{}, fmt.Errorf("oauth resource origin %d must be an http(s) origin without a path, query, or fragment", i)
		}
		policy.origins = append(policy.origins, originOf(parsed))
	}
	resolved, ok := policy.normalize(fallback)
	if !ok {
		return ResourcePolicy{}, fmt.Errorf("the default oauth resource must be a URL on an allowed origin of at most %d bytes", tokenjwt.MaxAudienceEntrySize)
	}
	policy.fallback = resolved
	return policy, nil
}

// resolve returns the one resource a request names, or the fallback when it
// names none.
func (p ResourcePolicy) resolve(requested []string) (string, *OAuthError) {
	switch len(requested) {
	case 0:
		return p.fallback, nil
	case 1:
		resource, ok := p.normalize(requested[0])
		if !ok {
			return "", oauthError(OAuthInvalidTarget, "This resource is not served by this authorization server.")
		}
		return resource, nil
	default:
		return "", oauthError(OAuthInvalidTarget, "Request at most one resource.")
	}
}

// requireSame accepts a token request that repeats the authorized resource or
// names none.
func (p ResourcePolicy) requireSame(requested []string, authorized string) error {
	if len(requested) == 0 {
		return nil
	}
	resource, err := p.resolve(requested)
	if err != nil {
		return err
	}
	if resource != authorized {
		return oauthError(OAuthInvalidTarget, "resource does not match the authorization request.")
	}
	return nil
}

// normalize lowercases the origin and drops a trailing slash, so
// https://MCP.unkey.com/ and https://mcp.unkey.com bind the same audience.
func (p ResourcePolicy) normalize(raw string) (string, bool) {
	parsed, err := url.Parse(raw)
	if err != nil || !parsed.IsAbs() || parsed.User != nil || parsed.RawQuery != "" || parsed.ForceQuery || strings.Contains(raw, "#") {
		return "", false
	}
	origin := originOf(parsed)
	if !slices.Contains(p.origins, origin) {
		return "", false
	}
	resource := origin + strings.TrimRight(parsed.EscapedPath(), "/")
	if len(resource) > tokenjwt.MaxAudienceEntrySize {
		return "", false
	}
	return resource, true
}

func originOf(u *url.URL) string {
	return strings.ToLower(u.Scheme + "://" + u.Host)
}
