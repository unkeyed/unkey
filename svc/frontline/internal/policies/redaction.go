package policies

import (
	"strings"

	frontlinev1 "github.com/unkeyed/unkey/gen/proto/frontline/v1"
)

// SecretLocations inspects KeyAuth policies and reports the request locations
// that carry an API key, so request logging can redact them before persisting
// rows to ClickHouse. Header names are returned lowercased for case-insensitive
// matching against http.Header keys; query parameter names are returned
// verbatim because URL query keys are case-sensitive.
//
// All KeyAuth policies are considered regardless of their enabled flag: a
// configured location name identifies where a credential is expected to appear,
// and a client may send it even if the policy is currently disabled. The
// Authorization header (used by the Bearer location and the default) is always
// redacted by the logging middleware and is not returned here.
func SecretLocations(policies []*frontlinev1.Policy) (headers []string, queryParams []string) {
	for _, policy := range policies {
		cfg, ok := policy.GetConfig().(*frontlinev1.Policy_Keyauth)
		if !ok {
			continue
		}
		for _, loc := range cfg.Keyauth.GetLocations() {
			switch l := loc.GetLocation().(type) {
			case *frontlinev1.KeyLocation_Header:
				if name := l.Header.GetName(); name != "" {
					headers = append(headers, strings.ToLower(name))
				}
			case *frontlinev1.KeyLocation_QueryParam:
				if name := l.QueryParam.GetName(); name != "" {
					queryParams = append(queryParams, name)
				}
			}
		}
	}
	return headers, queryParams
}
