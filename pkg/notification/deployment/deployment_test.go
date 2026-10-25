package deployment_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/notification"
	"github.com/unkeyed/unkey/pkg/notification/deployment"
)

func TestDeploymentReportsKeepDistinctOccurrencesAndLinks(t *testing.T) {
	resource := notification.Resource{ProjectID: "project", AppID: "app", EnvironmentID: "env"}
	payload := deployment.Payload{
		DeploymentID: "deployment", Change: deployment.RegionsStarted, Project: "shop", App: "api",
		Environment: "preview", EnvironmentURL: "https://exact-deployment.example.com", LogURL: "https://app.example.com/logs/deployment",
	}
	identities := make(map[string]bool)
	for _, change := range []deployment.Change{
		deployment.Preparing, deployment.BuildFailed, deployment.LoadFailed, deployment.Cancelled, deployment.Superseded,
		deployment.RegionsStarted, deployment.RegionsFailed, deployment.RoutingStarted,
		deployment.RoutingFailed, deployment.FinalizationFailed, deployment.Live, deployment.CompletedWithoutPromotion,
	} {
		payload.Change = change
		event, err := deployment.Event("workspace", resource, 12345, payload)
		require.NoError(t, err)
		require.False(t, identities[event.ID()], "distinct transition %s reused an occurrence", change)
		identities[event.ID()] = true
		decoded, err := deployment.Decode(event)
		require.NoError(t, err)
		require.Equal(t, payload, decoded)
		var matched bool
		for _, definition := range deployment.Definitions() {
			if definition.Type() != event.Type {
				continue
			}
			content, err := definition.Render(event)
			require.NoError(t, err)
			require.Contains(t, content.Text, payload.EnvironmentURL)
			require.Contains(t, content.Text, payload.LogURL)
			matched = true
		}
		require.True(t, matched)
		repeated, err := deployment.Event("workspace", resource, 12345, payload)
		require.NoError(t, err)
		require.Equal(t, event, repeated)
	}
	_, err := deployment.Event("workspace", resource, 12345, deployment.Payload{DeploymentID: "deployment", Change: "invented"})
	require.Error(t, err)
	payload.LogURL = "javascript:alert(1)"
	_, err = deployment.Event("workspace", resource, 12345, payload)
	require.Error(t, err)
}
