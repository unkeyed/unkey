package v2EcsMeta

import (
	"fmt"
	"io"
	"net/http"
	"os"

	openapi "github.com/unkeyed/unkey/go/api"
	zen "github.com/unkeyed/unkey/go/pkg/zen"
)

type Response = openapi.V2LivenessResponseBody

func New() zen.Route {
	return zen.NewRoute("GET", "/v2/__ecs_meta", func(s *zen.Session) error {

		metaRes, err := http.Get(fmt.Sprintf("%s/task", os.Getenv("ECS_CONTAINER_METADATA_URI")))
		if err != nil {
			return s.Send(500, []byte(err.Error()))
		}
		defer metaRes.Body.Close()

		b, err := io.ReadAll(metaRes.Body)
		if err != nil {
			return s.Send(500, []byte(err.Error()))
		}

		return s.Send(http.StatusOK, b)
	})
}
