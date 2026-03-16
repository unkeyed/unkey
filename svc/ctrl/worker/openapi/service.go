package openapi

import (
	"net/http"
	"time"

	hydrav1 "github.com/unkeyed/unkey/gen/proto/hydra/v1"
	"github.com/unkeyed/unkey/pkg/db"
)

// Service implements the OpenapiService. It scrapes OpenAPI specs from running
// deployments (when openapi_spec_path is configured) and persists them in the database.
type Service struct {
	hydrav1.UnimplementedOpenapiServiceServer
	db         db.Database
	httpClient *http.Client
}

var _ hydrav1.OpenapiServiceServer = (*Service)(nil)

// Config holds the dependencies required to create a Service.
type Config struct {
	// DB is the main database connection.
	DB db.Database
}

// New creates a new Service from the given configuration.
func New(cfg Config) *Service {
	return &Service{
		UnimplementedOpenapiServiceServer: hydrav1.UnimplementedOpenapiServiceServer{},
		db:                                cfg.DB,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}
