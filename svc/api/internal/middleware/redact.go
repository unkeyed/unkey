package middleware

import (
	"regexp"
	"sort"
	"strings"

	"github.com/unkeyed/unkey/pkg/uid"
)

// resourceIDPrefixes are the identifier prefixes a 403 detail must not reveal.
// Request ids are omitted so a caller can still quote meta.requestId.
var resourceIDPrefixes = []uid.Prefix{
	uid.KeyPrefix,
	uid.APIPrefix,
	uid.WorkspacePrefix,
	uid.KeySpacePrefix,
	uid.RolePrefix,
	uid.RatelimitNamespacePrefix,
	uid.RatelimitOverridePrefix,
	uid.PermissionPrefix,
	uid.IdentityPrefix,
	uid.RatelimitPrefix,
	uid.AuditLogPrefix,
	uid.CorrelationPrefix,
	uid.InstancePrefix,
	uid.FrontlinePrefix,
	uid.CiliumNetworkPolicyPrefix,
	uid.ClusterPrefix,
	uid.RegionPrefix,
	uid.OrgPrefix,
	uid.LogdrainPrefix,
	uid.PortalExchangeCodePrefix,
	uid.PortalAccessTokenPrefix,
	uid.PortalSessionPrefix,
	uid.PortalPrefix,
	uid.OpenApiSpecPrefix,
	uid.ProjectPrefix,
	uid.EnvironmentPrefix,
	uid.EnvironmentVariablePrefix,
	uid.AppPrefix,
	uid.ConnectionPrefix,
	uid.DomainPrefix,
	uid.DeploymentPrefix,
	uid.FrontlineRoutePrefix,
	uid.CertificatePrefix,
	uid.PolicyPrefix,
	uid.AutoscalingPolicyPrefix,
	uid.DeviceAuthorizationPrefix,
	uid.CLISessionPrefix,
	uid.TestPrefix,
}

var resourceIDPattern = newResourceIDPattern()

func newResourceIDPattern() *regexp.Regexp {
	prefixes := make([]string, len(resourceIDPrefixes))
	for i, prefix := range resourceIDPrefixes {
		prefixes[i] = regexp.QuoteMeta(string(prefix))
	}
	sort.Slice(prefixes, func(i, j int) bool {
		return len(prefixes[i]) > len(prefixes[j])
	})
	return regexp.MustCompile(`(?:^|[^A-Za-z0-9])(?:` + strings.Join(prefixes, "|") + `)_[A-Za-z0-9]+`)
}

// RedactResourceIDs replaces resource ids in a 403 detail with the prefix in
// braces. A request id (req_...) is left as written.
func isIDChar(b byte) bool {
	return (b >= 'A' && b <= 'Z') || (b >= 'a' && b <= 'z') || (b >= '0' && b <= '9')
}

func RedactResourceIDs(message string) string {
	return resourceIDPattern.ReplaceAllStringFunc(message, func(match string) string {
		start := 0
		if !isIDChar(match[0]) {
			start = 1
		}
		prefixEnd := strings.Index(match[start:], "_") + start
		return match[:start] + "{" + match[start:prefixEnd] + "}"
	})
}
