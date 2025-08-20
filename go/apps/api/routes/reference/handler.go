package reference

import (
	"context"
	"fmt"

	"github.com/unkeyed/unkey/go/apps/api/openapi"
	"github.com/unkeyed/unkey/go/pkg/otel/logging"
	"github.com/unkeyed/unkey/go/pkg/zen"
)

// Handler implements zen.Route interface for the API reference endpoint
type Handler struct {
	// Services as public fields (even though not used in this handler, showing the pattern)
	Logger logging.Logger
}

// Method returns the HTTP method this route responds to
func (h *Handler) Method() string {
	return "GET"
}

// Path returns the URL path pattern this route matches
func (h *Handler) Path() string {
	return "/reference"
}

// Handle processes the HTTP request
func (h *Handler) Handle(ctx context.Context, s *zen.Session) error {
	s.DisableClickHouseLogging()

	html := fmt.Sprintf(`
<!doctype html>
<html>
  <head>
    <title>Unkey API Reference</title>
    <meta charset="utf-8" />
    <meta
      name="viewport"
      content="width=device-width, initial-scale=1" />
  </head>
  <body>
  <script
    id="api-reference"
    type="application/json">
   %s
  </script>
    <script src="https://cdn.jsdelivr.net/npm/@scalar/api-reference"></script>
  </body>
</html>
		`, string(openapi.Spec))

	s.AddHeader("Content-Type", "text/html")
	return s.Send(200, []byte(html))
}
