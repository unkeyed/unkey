package routes

import (
	"time"

	"github.com/unkeyed/unkey/gen/rpc/ctrl"
	"github.com/unkeyed/unkey/pkg/batch"
	"github.com/unkeyed/unkey/pkg/clickhouse/schema"
	"github.com/unkeyed/unkey/pkg/clock"
	"github.com/unkeyed/unkey/svc/frontline/internal/errorpage"
	"github.com/unkeyed/unkey/svc/frontline/internal/policies"
	"github.com/unkeyed/unkey/svc/frontline/internal/proxy"
	"github.com/unkeyed/unkey/svc/frontline/internal/router"
)

type Services struct {
	Region            string
	Platform          string
	FrontlineID       string
	RouterService     router.Service
	ProxyService      proxy.Service
	Engine            policies.Evaluator
	Clock             clock.Clock
	AcmeClient        ctrl.AcmeServiceClient
	ErrorPageRenderer errorpage.Renderer
	RequestTimeout    time.Duration
	// FrontlineRequests buffers per-request analytics for ClickHouse on the
	// local-instance path. Carries the same schema sentinel used so existing
	// dashboards and materialized views keep working through the cutover.
	FrontlineRequests *batch.BatchProcessor[schema.SentinelRequest]
}
