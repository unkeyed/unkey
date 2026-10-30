package handler

import (
	"context"
	"sync"

	"github.com/unkeyed/unkey/pkg/fault"
	"github.com/unkeyed/unkey/pkg/zen"
	"github.com/unkeyed/unkey/svc/api/openapi"
	"sigs.k8s.io/yaml"
)

// Handler implements zen.Route interface for the API reference endpoint
type Handler struct {
	Format string
}

// We tried build-time JSON generation, but it added a second generated diff for
// every spec change and would make reviews larger than necessary. This converts
// it once at runtime to keep only the YAML in version control.
var specJSON = sync.OnceValues(func() ([]byte, error) {
	return yaml.YAMLToJSON(openapi.Spec)
})

// Method returns the HTTP method this route responds to
func (h *Handler) Method() string {
	return "GET"
}

// Path returns the URL path pattern this route matches
func (h *Handler) Path() string {
	return "/openapi." + h.Format
}

// Handle processes the HTTP request
func (h *Handler) Handle(ctx context.Context, s *zen.Session) error {
	s.SkipRequestLog()

	var spec []byte
	switch h.Format {
	case "yaml":
		spec = openapi.Spec
	case "json":
		var err error
		spec, err = specJSON()
		if err != nil {
			return fault.Wrap(err, fault.Internal("failed to encode OpenAPI specification as JSON"))
		}
	default:
		return fault.New("unsupported OpenAPI specification format")
	}

	s.AddHeader("Content-Type", "application/"+h.Format)
	s.AddHeader("Cache-Control", "public, max-age=3600")
	return s.Send(200, spec)
}
