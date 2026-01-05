package metrics

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/svc/agent/pkg/logging"
	"github.com/unkeyed/unkey/svc/agent/pkg/uid"
)

type fakeMetric struct {
	Value   string `json:"value"`
	Another bool   `json:"another"`
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

	expected := map[string]any{
		"value":       fm.Value,
		"another":     fm.Another,
		"_time":       now.UnixMilli(),
		"metric":      "metric.fake",
		"nodeId":      nodeId,
		"region":      "test",
		"application": "agent",
	}

	e, err := json.Marshal(expected)
	require.NoError(t, err)

	require.JSONEq(t, string(e), string(b))

}
