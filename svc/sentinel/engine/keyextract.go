package engine

import (
	"net/http"
	"strings"

	sentinelv1 "github.com/unkeyed/unkey/gen/proto/sentinel/v1"
)

// extractKey tries each location in order and returns the first non-empty key.
// If no locations are configured, it defaults to extracting a Bearer token from
// the Authorization header.
func extractKey(req *http.Request, locations []*sentinelv1.KeyLocation) string {
	if len(locations) == 0 {
		return extractBearer(req)
	}

	for _, loc := range locations {
		var key string
		switch l := loc.GetLocation().(type) {
		case *sentinelv1.KeyLocation_Bearer:
			key = extractBearer(req)
		case *sentinelv1.KeyLocation_Header:
			key = extractHeader(req, l.Header)
		case *sentinelv1.KeyLocation_QueryParam:
			key = extractQueryParam(req, l.QueryParam)
		}
		if key != "" {
			return key
		}
	}
	return ""
}

// extractBearer extracts the token from "Authorization: Bearer <token>".
func extractBearer(req *http.Request) string {
	auth := req.Header.Get("Authorization")
	if auth == "" {
		return ""
	}
	const prefix = "Bearer "
	if len(auth) > len(prefix) && strings.EqualFold(auth[:len(prefix)], prefix) {
		return auth[len(prefix):]
	}
	return ""
}

// extractHeader extracts the key from a named header, optionally stripping a prefix.
func extractHeader(req *http.Request, loc *sentinelv1.HeaderKeyLocation) string {
	if loc == nil {
		return ""
	}
	val := req.Header.Get(loc.GetName())
	if val == "" {
		return ""
	}
	if sp := loc.GetStripPrefix(); sp != "" {
		if len(val) > len(sp) && strings.EqualFold(val[:len(sp)], sp) {
			return val[len(sp):]
		}
		return ""
	}
	return val
}

// extractQueryParam extracts the key from a URL query parameter.
func extractQueryParam(req *http.Request, loc *sentinelv1.QueryParamKeyLocation) string {
	if loc == nil {
		return ""
	}
	return req.URL.Query().Get(loc.GetName())
}
