package clickhouse

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestAnomalyWindowBatches(t *testing.T) {
	t.Parallel()

	keys := make([]AnomalyGroupKey, 1_201)
	for i := range keys {
		keys[i] = AnomalyGroupKey{WorkspaceID: fmt.Sprintf("ws-%d", i)}
	}
	batches := anomalyWindowBatches(AnomalyWindowsRequest{GroupKeys: keys})
	require.Len(t, batches, 4)
	require.Empty(t, batches[0].GroupKeys)
	require.True(t, batches[0].IncludeFleet)
	require.Len(t, batches[1].GroupKeys, 500)
	require.Len(t, batches[2].GroupKeys, 500)
	require.Len(t, batches[3].GroupKeys, 201)
	require.False(t, batches[1].IncludeFleet)
	require.False(t, batches[2].IncludeFleet)
	require.False(t, batches[3].IncludeFleet)

	batches = anomalyWindowBatches(AnomalyWindowsRequest{GroupKeys: keys, SkipFleet: true})
	require.Len(t, batches, 3)
	for _, batch := range batches {
		require.False(t, batch.IncludeFleet)
	}
}

func TestAnomalyGroupKeysParamEscapesValues(t *testing.T) {
	t.Parallel()

	param := anomalyGroupKeysParam([]AnomalyGroupKey{{
		WorkspaceID: "ws'\\", ProjectID: "project", AppID: "app", EnvironmentID: "env",
	}})
	require.Equal(t, "[('ws\\'\\\\','project','app','env')]", param)
}

func TestGroupPhysicalShardsParam(t *testing.T) {
	t.Parallel()

	keys := []AnomalyGroupKey{
		{WorkspaceID: "workspace-a"},
		{WorkspaceID: "workspace-b"},
		{WorkspaceID: "workspace-a"},
	}
	param := groupPhysicalShardsParam(keys)
	require.Regexp(t, `^\[\d+(,\d+)?\]$`, param)
}
