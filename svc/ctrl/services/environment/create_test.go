package environment

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestPlanWritesCreatesCompleteEnvironment(t *testing.T) {
	planned := planWrites(CreateManyParams{
		App: AppScope{
			WorkspaceID: "ws_test",
			ProjectID:   "proj_test",
			AppID:       "app_test",
		},
		Environments: []CreateSpec{
			{ID: "env_test", Slug: "staging", Description: "Staging"},
		},
		Now: 1234,
	}, []string{"rgn_a", "rgn_b"})

	require.Len(t, planned.environments, 1)
	require.Len(t, planned.buildSettings, 1)
	require.Len(t, planned.runtimeSettings, 1)
	require.Len(t, planned.regionalSettings, 2)

	require.Equal(t, "env_test", planned.environments[0].ID)
	require.Equal(t, ".", planned.buildSettings[0].DockerContext)
	require.Equal(t, int32(8080), planned.runtimeSettings[0].Port)
	require.Equal(t, int32(250), planned.runtimeSettings[0].CpuMillicores)
	require.Equal(t, int32(256), planned.runtimeSettings[0].MemoryMib)
	require.ElementsMatch(
		t,
		[]string{"rgn_a", "rgn_b"},
		[]string{planned.regionalSettings[0].RegionID, planned.regionalSettings[1].RegionID},
	)
}
