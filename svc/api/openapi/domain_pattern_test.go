package openapi_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"

	"github.com/unkeyed/unkey/pkg/dns"
	"github.com/unkeyed/unkey/svc/api/openapi"
)

// The spec advertises these constraints and ctrl re-validates against
// [dns.IsValidFQDN]. Drift produces either a 500 on a domain the spec calls valid,
// or a rejection of one it accepts.
func TestSpecDomainConstraintsMatchDNS(t *testing.T) {
	t.Parallel()

	var doc struct {
		Components struct {
			Schemas map[string]struct {
				Properties struct {
					Domain struct {
						Pattern   string `yaml:"pattern"`
						MaxLength int    `yaml:"maxLength"`
					} `yaml:"domain"`
				} `yaml:"properties"`
			} `yaml:"schemas"`
		} `yaml:"components"`
	}
	require.NoError(t, yaml.Unmarshal(openapi.Spec, &doc))

	schema, ok := doc.Components.Schemas["V2DomainsCreateDomainRequestBody"]
	require.True(t, ok, "V2DomainsCreateDomainRequestBody missing from the bundled spec")

	require.Equal(t, dns.FQDNPattern, schema.Properties.Domain.Pattern)
	require.Equal(t, dns.MaxFQDNLength, schema.Properties.Domain.MaxLength)
}
