package deploy

import (
	restate "github.com/restatedev/sdk-go"
	"github.com/unkeyed/unkey/svc/ctrl/internal/db"
)

func (w *Workflow) ConfigureRouting(ctx restate.ObjectContext, deployment db.FindDeploymentForDeployRow) ([]string, bool, error) {
	return w.configureRouting(ctx, deployment)
}
