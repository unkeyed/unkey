package reference

import (
	"context"
	"fmt"

	"github.com/unkeyed/unkey/go/apps/api/openapi"
	"github.com/unkeyed/unkey/go/pkg/zen"
)

func New() zen.Route {

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

	return zen.NewRoute("GET", "/reference", func(ctx context.Context, s *zen.Session) error {

		s.AddHeader("Content-Type", "text/html")
		return s.Send(200, []byte(html))
	})
}
