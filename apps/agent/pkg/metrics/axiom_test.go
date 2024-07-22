package metrics

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/apps/agent/pkg/logging"
	"github.com/unkeyed/unkey/apps/agent/pkg/uid"
)

type fakeMetric struct {
	Value   string
	Another bool
}

func (fm fakeMetric) Name() string {
	return "metric.fake"
}

func TestMerge(t *testing.T) {

	nodeId := uid.New("")
	region := "test"
	now := time.Now()

	ax, err := New(Config{
		Token:   "",
		NodeId:  nodeId,
		Region:  region,
		Logger:  logging.NewNoopLogger(),
		Dataset: "",
	})

	require.NoError(t, err)

	fm := fakeMetric{
		Value:   uid.New(""),
		Another: true,
	}

	merged := ax.merge(fm, now)
	b, err := json.Marshal(merged)
	require.NoError(t, err)

	t.Log(string(b))

	expected := map[string]any{
		"Metric": map[string]any{
			"Value":   fm.Value,
			"Another": fm.Another,
		},
		"Time":        now.UnixMilli(),
		"_time":       now.UnixMilli(),
		"NodeId":      nodeId,
		"Region":      "test",
		"Application": "agent",
	}

	e, err := json.Marshal(expected)
	require.NoError(t, err)

	require.JSONEq(t, string(e), string(b))

}
