package openapi_test

import (
	"slices"
	"sort"
	"testing"

	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"

	"github.com/unkeyed/unkey/svc/api/openapi"
)

// The bundled spec is what the published reference is built from, so an
// `x-excluded` annotation that survives in the split sources but is dropped
// during bundling would quietly publish an unlaunched product. Same failure mode
// as the redaction paths next door: the flag lives in one file and its effect is
// felt somewhere else entirely.
//
// The portal's data routes authenticate with the end user's session cookie and
// are internal to the hosted portal, so only the customer-facing operations are
// published. A portal path not listed as public must be excluded, so a new one
// that forgets the flag fails here rather than surfacing in the docs. Adding to
// the public list is the launch decision for that operation.
//
// Note what this does not do: `x-excluded` hides an operation from the reference,
// not from the SDK. Nothing in this repo reads it at runtime and the routes are
// registered and reachable regardless, so it is a documentation gate and never an
// access control.
func TestSpecExcludesInternalPortalOperations(t *testing.T) {
	t.Parallel()

	var doc struct {
		Paths map[string]map[string]struct {
			OperationID string `yaml:"operationId"`
			Excluded    bool   `yaml:"x-excluded"`
		} `yaml:"paths"`
	}
	require.NoError(t, yaml.Unmarshal(openapi.Spec, &doc))

	excluded := []string{}
	portal := []string{}
	for path, methods := range doc.Paths {
		for _, op := range methods {
			if op.OperationID == "" {
				continue
			}
			if op.Excluded {
				excluded = append(excluded, op.OperationID)
			}
			// Match on the path rather than the operationId prefix: a portal
			// operation that lands outside /v2/portal.* should still be caught.
			if len(path) >= len("/v2/portal.") && path[:len("/v2/portal.")] == "/v2/portal." {
				portal = append(portal, op.OperationID)
			}
		}
	}
	sort.Strings(excluded)
	sort.Strings(portal)

	require.Equal(t, []string{
		"liveness", // infrastructure endpoint, not part of the customer-facing reference
		"portal.exchangeCode",
		"portal.getDomain",
		"portal.getVerifications",
		"portal.listKeys",
		"portal.rerollKey",
	}, excluded, "every excluded operation is accounted for")

	public := []string{
		"portal.createDomain",
		"portal.createPortal",
		"portal.createSession",
		"portal.deletePortal",
		"portal.getPortal",
		"portal.listSessions",
		"portal.revokeSession",
		"portal.updatePortal",
	}
	for _, op := range portal {
		if !slices.Contains(public, op) {
			require.Contains(t, excluded, op,
				"portal operations stay out of the published reference unless listed as public")
		}
	}
}
