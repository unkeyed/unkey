package testutil

import (
	"testing"

	apisv1 "github.com/unkeyed/unkey/apps/agent/gen/proto/apis/v1"
	authenticationv1 "github.com/unkeyed/unkey/apps/agent/gen/proto/authentication/v1"
	"github.com/unkeyed/unkey/apps/agent/pkg/cache"
	"github.com/unkeyed/unkey/apps/agent/pkg/logging"
	"github.com/unkeyed/unkey/apps/agent/pkg/server"
	"github.com/unkeyed/unkey/apps/agent/pkg/services/apis"
	"github.com/unkeyed/unkey/apps/agent/pkg/services/keys"
	"github.com/unkeyed/unkey/apps/agent/pkg/tracing"
)

func NewServer(t *testing.T, r resources) *server.Server {

	return server.New(server.Config{
		UnkeyAppAuthToken: r.UnkeyAppAuthToken,
		Logger:            logging.New(&logging.Config{Debug: true}),
		KeyCache:          cache.NewNoopCache[*authenticationv1.Key](),
		ApiCache:          cache.NewNoopCache[*apisv1.Api](),
		Database:          r.Database,
		Tracer:            tracing.NewNoop(),
		KeyService: keys.New(keys.Config{
			Logger:   logging.NewNoop(),
			Database: r.Database,
		}),
		ApiService: apis.New(apis.Config{
			Database: r.Database,
		}),
	})
}
