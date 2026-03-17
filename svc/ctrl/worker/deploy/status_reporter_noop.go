package deploy

import restate "github.com/restatedev/sdk-go"

// noopStatusReporter is a no-op implementation for deployments without a GitHub
// repo connection.
type noopStatusReporter struct{}

// NewNoopStatusReporter creates a no-op status reporter.
func NewNoopStatusReporter() deploymentStatusReporter {
	return noopStatusReporter{}
}

func (noopStatusReporter) Create(_ restate.ObjectSharedContext)                     {}
func (noopStatusReporter) Report(_ restate.ObjectSharedContext, _ string, _ string) {}
