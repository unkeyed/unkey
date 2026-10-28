package openapi

import (
	"strings"

	"github.com/unkeyed/unkey/pkg/fault"
	"gopkg.in/yaml.v3"
)

// WorkspaceRateLimitExemptions contains operation-level exemptions. Its zero
// value exempts nothing, including operations absent from the spec.
type WorkspaceRateLimitExemptions struct {
	operations map[string]struct{}
}

// WorkspaceRateLimitExemptionsFromSpec reads x-unkey-workspace-rate-limit-exempt
// from a bundled spec. Only an explicit boolean true exempts an operation.
func WorkspaceRateLimitExemptionsFromSpec(spec []byte) (WorkspaceRateLimitExemptions, error) {
	var doc struct {
		Paths map[string]map[string]yaml.Node `yaml:"paths"`
	}
	if err := yaml.Unmarshal(spec, &doc); err != nil {
		return WorkspaceRateLimitExemptions{}, fault.Wrap(err, fault.Internal("failed to parse workspace rate limit exemptions"))
	}

	exemptions := WorkspaceRateLimitExemptions{operations: make(map[string]struct{})}
	for path, item := range doc.Paths {
		for method, node := range item {
			switch method {
			case "get", "put", "post", "delete", "options", "head", "patch", "trace":
			default:
				continue
			}

			var operation struct {
				Exempt yaml.Node `yaml:"x-unkey-workspace-rate-limit-exempt"`
			}
			if err := node.Decode(&operation); err != nil {
				return WorkspaceRateLimitExemptions{}, fault.Wrap(err, fault.Internal("failed to parse operation "+method+" "+path))
			}
			if operation.Exempt.Kind == 0 {
				continue
			}
			if operation.Exempt.Tag != "!!bool" {
				return WorkspaceRateLimitExemptions{}, fault.New("x-unkey-workspace-rate-limit-exempt must be a boolean: " + method + " " + path)
			}

			var exempt bool
			if err := operation.Exempt.Decode(&exempt); err != nil {
				return WorkspaceRateLimitExemptions{}, fault.Wrap(err, fault.Internal("failed to decode workspace rate limit exemption"))
			}
			if exempt {
				exemptions.operations[strings.ToUpper(method)+" "+path] = struct{}{}
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
