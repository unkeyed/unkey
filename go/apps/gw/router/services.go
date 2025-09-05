package router

import (
	"github.com/unkeyed/unkey/go/apps/gw/services/certmanager"
	"github.com/unkeyed/unkey/go/apps/gw/services/routing"
	"github.com/unkeyed/unkey/go/apps/gw/services/validation"
	"github.com/unkeyed/unkey/go/gen/proto/ctrl/v1/ctrlv1connect"
	"github.com/unkeyed/unkey/go/internal/services/keys"
	"github.com/unkeyed/unkey/go/internal/services/ratelimit"
	"github.com/unkeyed/unkey/go/pkg/clickhouse"
	"github.com/unkeyed/unkey/go/pkg/otel/logging"
)

// Services holds all the services needed by the gateway routes.
type Services struct {
	Logger         logging.Logger
	CertManager    certmanager.Service
	RoutingService routing.Service
	Validation     validation.Validator  // For OpenAPI request validation
	ClickHouse     clickhouse.ClickHouse // For metrics middleware
	Keys           keys.KeyService
	Ratelimit      ratelimit.Service
	MainDomain     string // Main gateway domain for internal endpoints
	AcmeClient     ctrlv1connect.AcmeServiceClient
}
