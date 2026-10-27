package githubstatus

import (
	"time"

	restate "github.com/restatedev/sdk-go"
	"github.com/unkeyed/unkey/pkg/logger"
	"github.com/unkeyed/unkey/pkg/notification"
	"github.com/unkeyed/unkey/pkg/notification/deployment"
)

const deploymentServiceName = "notification.v1.GitHubDeploymentService"

type outcome string

const (
	accepted outcome = "accepted"
	unknown  outcome = "unknown"
	skipped  outcome = "skipped"
)

// ReportResult records independent GitHub outcomes: accepted, unknown, or skipped.
// Unknown means an operation failed without proving that GitHub rejected the write.
type ReportResult struct {
	Commit     outcome `json:"commit"`
	Deployment outcome `json:"deployment"`
	Comment    outcome `json:"comment"`
}

// Name identifies this integration to the notification dispatcher.
func (s *Service) Name() string { return "github" }

// Accepts selects the registered deployment event types.
func (s *Service) Accepts(eventType string) bool {
	for _, definition := range deployment.Definitions() {
		if definition.Type() == eventType {
			return true
		}
	}
	return false
}

// Dispatch waits for reporting, but individual unknown GitHub outcomes are not errors.
func (s *Service) Dispatch(ctx restate.Context, event notification.Event) error {
	payload, err := deployment.Decode(event)
	if err != nil {
		return restate.ToTerminalError(err)
	}
	_, err = restate.Object[ReportResult](ctx, deploymentServiceName, payload.DeploymentID, "Report").Request(event)
	return err
}

// DeploymentDefinition binds the private, deployment-keyed notification object.
func (s *Service) DeploymentDefinition() restate.ServiceDefinition {
	return restate.NewObject(deploymentServiceName).
		Handler("Report", restate.NewObjectHandler(s.ReportDeployment, restate.WithIngressPrivate(true), restate.WithInvocationRetryPolicy(
			restate.WithInitialRetryInterval(time.Second), restate.WithMaxRetryInterval(time.Minute),
			restate.WithMaxRetryAttempts(20), restate.PauseOnMaxAttempts(),
		)))
}

func operationOutcome(err error, eventID, operation string) outcome {
	if err != nil {
		logger.Error("GitHub notification outcome unknown", "notification_id", eventID, "operation", operation, "error", err)
		return unknown
	}
	return accepted
}
