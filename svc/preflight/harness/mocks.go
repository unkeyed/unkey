package harness

import (
	restate "github.com/restatedev/sdk-go"
	hydrav1 "github.com/unkeyed/unkey/gen/proto/hydra/v1"
)

// NullRestateServices returns a slice of accept-and-discard Restate
// service implementations covering the services the ctrl API may send
// to during a preflight run. Callers pass these into
// harness.Config.MockRestateServices when they do not care about the
// downstream workflow behaviour and just want Restate to not complain.
//
// Probe unit tests that need to inspect incoming requests replace the
// relevant entry with their own mock that captures into a channel.
//
// Adding a new service here is cheaper than adding a new flag to
// Config, because it preserves the "caller provides exactly what they
// need" shape.
func NullRestateServices() []restate.ServiceDefinition {
	return []restate.ServiceDefinition{
		//nolint:exhaustruct // Unimplemented server embeds by value
		hydrav1.NewGitHubWebhookServiceServer(nullGitHubWebhook{}),
		//nolint:exhaustruct // Unimplemented server embeds by value
		hydrav1.NewDeployServiceServer(nullDeploy{}),
	}
}

type nullGitHubWebhook struct {
	hydrav1.UnimplementedGitHubWebhookServiceServer
}

func (nullGitHubWebhook) HandlePush(_ restate.ObjectContext, _ *hydrav1.HandlePushRequest) (*hydrav1.HandlePushResponse, error) {
	//nolint:exhaustruct // empty response on purpose
	return &hydrav1.HandlePushResponse{}, nil
}

type nullDeploy struct {
	hydrav1.UnimplementedDeployServiceServer
}

func (nullDeploy) Deploy(_ restate.ObjectContext, _ *hydrav1.DeployRequest) (*hydrav1.DeployResponse, error) {
	//nolint:exhaustruct // empty response on purpose
	return &hydrav1.DeployResponse{}, nil
}
