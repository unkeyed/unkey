package privatenetwork

import (
	"regexp"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/uid"
	"k8s.io/apimachinery/pkg/util/validation"
)

var coreDNSRewrite = regexp.MustCompile(`^(unkey-connection-[0-9a-f]{32})\.([a-z0-9](?:[a-z0-9-]{0,61}[a-z0-9])?)\.unkey\.internal\.$`)

func TestConnectionServiceName(t *testing.T) {
	require.Equal(t, "unkey-connection-f6c4e4d755cbd1da513524142c77c589", ConnectionServiceName("dep_123", "worker"))
	require.Equal(t, "unkey-connection-47927618a328df71dd3580a677b4cc81", ConnectionServiceName("dep_123", SelfName))
}

func TestRegionalConnectionServiceName(t *testing.T) {
	require.Equal(t, "unkey-r-f6c4e4d755cbd1da513524142c77c589-us-east-1", RegionalConnectionServiceName("dep_123", "worker", "us-east-1"))

	caller, alias := uid.New(uid.DeploymentPrefix), "a"+strings.Repeat("b", 62)
	region := "r" + strings.Repeat("1", 21)
	name := RegionalConnectionServiceName(caller, alias, region)
	require.Len(t, name, 63)
	require.Empty(t, validation.IsDNS1123Label(name))
	require.Equal(t, strings.TrimPrefix(ConnectionServiceName(caller, alias), "unkey-connection-"), strings.TrimSuffix(strings.TrimPrefix(name, "unkey-r-"), "-"+region))
	require.NotEqual(t, name, RegionalConnectionServiceName(caller, alias, "r2"))
}

func TestConnectionServiceNameSeparatesCallersAndAliases(t *testing.T) {
	caller, other := uid.New(uid.DeploymentPrefix), uid.New(uid.DeploymentPrefix)
	names := map[string]struct{}{
		ConnectionServiceName(caller, "worker"):   {},
		ConnectionServiceName(other, "worker"):    {},
		ConnectionServiceName(caller, "worker-2"): {},
		ConnectionServiceName(caller, SelfName):   {},
		ConnectionServiceName(other, SelfName):    {},
	}
	require.Len(t, names, 5)
	require.Equal(t, ConnectionServiceName(caller, "worker"), ConnectionServiceName(caller, "worker"))
}

func TestConnectionHostMatchesCoreDNSRewrite(t *testing.T) {
	caller := uid.New(uid.DeploymentPrefix)
	for _, tt := range []struct {
		name      string
		alias     string
		namespace string
	}{
		{name: "short", alias: "worker", namespace: "a"},
		{name: "self", alias: SelfName, namespace: uid.DNS1035()},
		{name: "maximum lengths", alias: "a" + strings.Repeat("b", 62), namespace: "n" + strings.Repeat("0", 62)},
	} {
		t.Run(tt.name, func(t *testing.T) {
			host := ConnectionHost(caller, tt.alias, tt.namespace)
			match := coreDNSRewrite.FindStringSubmatch(host + ".")
			require.NotNil(t, match, "host %q must match the CoreDNS rewrite", host)
			require.Equal(t, ConnectionServiceName(caller, tt.alias), match[1])
			require.Equal(t, tt.namespace, match[2])
			require.Empty(t, validation.IsDNS1123Label(match[1]))
			require.Empty(t, validation.IsDNS1123Subdomain(host))
		})
	}
}
