package openapi

import (
	"github.com/getkin/kin-openapi/openapi3"
	"github.com/unkeyed/unkey/pkg/fault"
)

// WorkspaceRateLimitExemptions contains operation-level exemptions. Its zero
// value exempts nothing, including operations absent from the spec.
type WorkspaceRateLimitExemptions struct {
	operations map[string]struct{}
}

// WorkspaceRateLimitExemptionsFromSpec reads x-unkey-workspace-rate-limit-exempt
// from a bundled spec. Only an explicit boolean true exempts an operation.
func WorkspaceRateLimitExemptionsFromSpec(spec []byte) (WorkspaceRateLimitExemptions, error) {
	doc, err := openapi3.NewLoader().LoadFromData(spec)
	if err != nil {
		return WorkspaceRateLimitExemptions{}, fault.Wrap(err, fault.Internal("failed to parse workspace rate limit exemptions"))
	}

	exemptions := WorkspaceRateLimitExemptions{operations: make(map[string]struct{})}
	for path, item := range doc.Paths.Map() {
		for method, operation := range item.Operations() {
			value, present := operation.Extensions["x-unkey-workspace-rate-limit-exempt"]
			if !present {
				continue
			}
			exempt, ok := value.(bool)
			if !ok {
				return WorkspaceRateLimitExemptions{}, fault.New("x-unkey-workspace-rate-limit-exempt must be a boolean: " + method + " " + path)
			}
			if exempt {
				exemptions.operations[method+" "+path] = struct{}{}
			}
		}
	}
	return exemptions, nil
}

// Contains reports whether the exact method and path have an exemption.
func (e WorkspaceRateLimitExemptions) Contains(method, path string) bool {
	_, ok := e.operations[method+" "+path]
	return ok
}
