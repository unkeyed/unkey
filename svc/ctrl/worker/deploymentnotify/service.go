package deploymentnotify

import (
	"strings"
	"time"

	restate "github.com/restatedev/sdk-go"
	"github.com/unkeyed/unkey/pkg/assert"
	"github.com/unkeyed/unkey/pkg/clock"
	"github.com/unkeyed/unkey/pkg/notification/deployment"
	"github.com/unkeyed/unkey/svc/ctrl/internal/db"
	workernotification "github.com/unkeyed/unkey/svc/ctrl/worker/notification"
)

// Service serializes lifecycle notifications per deployment through Restate.
type Service struct {
	db           db.Database
	clock        clock.Clock
	dashboardURL string
	dispatcher   *workernotification.Dispatcher
}

// Config requires DB, Clock, and Dispatcher. DashboardURL is the base for inspect links.
type Config struct {
	DB           db.Database
	Clock        clock.Clock
	DashboardURL string
	Dispatcher   *workernotification.Dispatcher
}

// New rejects missing dependencies and removes trailing slashes from DashboardURL.
func New(cfg Config) (*Service, error) {
	if err := assert.All(assert.NotNil(cfg.DB), assert.NotNil(cfg.Clock), assert.NotNil(cfg.Dispatcher)); err != nil {
		return nil, err
	}
	return &Service{db: cfg.DB, clock: cfg.Clock, dashboardURL: strings.TrimRight(cfg.DashboardURL, "/"), dispatcher: cfg.Dispatcher}, nil
}

// Definition binds the private Report handler to a deployment-keyed object.
func (s *Service) Definition() restate.ServiceDefinition {
	return restate.NewObject(deployment.ServiceName).
		Handler("Report", restate.NewObjectHandler(s.Report, restate.WithIngressPrivate(true), restate.WithInvocationRetryPolicy(
			restate.WithInitialRetryInterval(time.Second), restate.WithMaxRetryInterval(time.Minute),
			restate.WithMaxRetryAttempts(20), restate.PauseOnMaxAttempts(),
		)))
}
