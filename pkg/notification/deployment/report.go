package deployment

import restate "github.com/restatedev/sdk-go"

// ServiceName is the Restate virtual object service for deployment reports.
const ServiceName = "notification.v1.DeploymentService"

// ReportRequest identifies a deployment transition without resolving notification content.
type ReportRequest struct {
	DeploymentID string `json:"deploymentId"`
	Change       Change `json:"change"`
}

// Report journals a background report; GitHub or subscription delivery failures
// do not fail the deployment. Call it outside restate.Run.
func Report(ctx restate.Context, deploymentID string, change Change) {
	restate.Object[string](ctx, ServiceName, deploymentID, "Report").Send(
		ReportRequest{DeploymentID: deploymentID, Change: change},
		restate.WithIdempotencyKey(string(change)),
	)
}
