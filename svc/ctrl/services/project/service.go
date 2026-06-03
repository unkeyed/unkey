package project

import (
	"time"

	restateingress "github.com/restatedev/sdk-go/ingress"
	"github.com/unkeyed/unkey/gen/proto/ctrl/v1/ctrlv1connect"
	"github.com/unkeyed/unkey/pkg/clock"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/svc/ctrl/services/app"
)

// gracePeriod is the wall-clock window between a DeleteProject call and
// the cron sweep performing the actual cascade. Within this window
// RestoreProject can undo the deletion.
const gracePeriod = 72 * time.Hour

// Service implements the ProjectService ConnectRPC API. CreateProject
// delegates app creation to AppService. DeleteProject fires the
// ProjectService.SoftDelete VO chain which stamps delete_permanently_at
// on the project + every descendant and stops their deployments. The
// cron sweep in svc/ctrl/worker/cron performs the hard cascade once the
// grace period has elapsed. RestoreProject fires the matching Restore
// VO chain.
type Service struct {
	ctrlv1connect.UnimplementedProjectServiceHandler
	db         db.Database
	clock      clock.Clock
	restate    *restateingress.Client
	appService *app.Service
	bearer     string
}

// Config holds the configuration for creating a new [Service].
type Config struct {
	// Database provides read and write access for managing projects.
	Database db.Database

	// Clock provides timestamps. Optional; defaults to a real clock.
	Clock clock.Clock

	// Restate is the ingress client used to trigger the SoftDelete /
	// Restore VO chains.
	Restate *restateingress.Client

	// AppService is used to create the default app when a new project is created.
	AppService *app.Service

	// Bearer is the preshared token that callers must provide in the Authorization header.
	Bearer string
}

// New creates a new [Service] with the given configuration.
func New(cfg Config) *Service {
	if cfg.Clock == nil {
		cfg.Clock = clock.New()
	}
	return &Service{
		UnimplementedProjectServiceHandler: ctrlv1connect.UnimplementedProjectServiceHandler{},
		db:                                 cfg.Database,
		clock:                              cfg.Clock,
		restate:                            cfg.Restate,
		appService:                         cfg.AppService,
		bearer:                             cfg.Bearer,
	}
}
