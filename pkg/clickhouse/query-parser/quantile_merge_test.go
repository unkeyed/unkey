package queryparser

import (
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func gatewayParser() *Parser {
	return NewParser(Config{
		WorkspaceID: "ws_KEBAP",
		TableAliases: map[string]string{
			"gateway_requests_per_hour_v1": "default.frontline_requests_per_hour_v1",
		},
		AllowedTables: []string{
			"default.frontline_requests_per_hour_v1",
		},
	})
}

// TestParser_QuantileTDigestMerge covers the parametric aggregate form used to
// read the AggregateFunction states on the gateway rollups. The vendored parser
// models `f(0.95)(col)` as a ParamExprList carrying a ColumnArgList, so this
// also pins that the validation walk descends into the column argument.
func TestParser_QuantileTDigestMerge(t *testing.T) {
	tests := []struct {
		name       string
		query      string
		shouldFail bool
	}{
		{
			name:       "merge state on the rollup",
			query:      "SELECT quantileTDigestMerge(0.95)(latency_p95) AS p95 FROM gateway_requests_per_hour_v1",
			shouldFail: false,
		},
		{
			name:       "lowercase spelling is accepted",
			query:      "SELECT quantiletdigestmerge(0.5)(latency_p50) AS p50 FROM gateway_requests_per_hour_v1",
			shouldFail: false,
		},
		{
			// The parametric position must not become a hole in the function
			// allow-list. If this passes, the walk is not reaching ColumnArgList
			// and every disallowed function can be smuggled through it.
			name:       "disallowed function in the column argument",
			query:      "SELECT quantileTDigestMerge(0.95)(file('/etc/passwd')) FROM gateway_requests_per_hour_v1",
			shouldFail: true,
		},
		{
			name:       "disallowed function in the parameter argument",
			query:      "SELECT quantileTDigestMerge(file('/etc/passwd'))(latency_p95) FROM gateway_requests_per_hour_v1",
			shouldFail: true,
		},
		{
			name:       "unmerged state accessor stays blocked",
			query:      "SELECT quantileTDigestState(0.95)(latency_p95) FROM gateway_requests_per_hour_v1",
			shouldFail: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := gatewayParser().Parse(context.Background(), tt.query)
			if tt.shouldFail {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
		})
	}
}

// TestParser_QuantileTDigestMergeCanonicalSpelling guarantees the rewritten
// query reaches ClickHouse with the case-sensitive spelling it requires.
func TestParser_QuantileTDigestMergeCanonicalSpelling(t *testing.T) {
	parsed, err := gatewayParser().Parse(
		context.Background(),
		"SELECT quantiletdigestmerge(0.95)(latency_p95) AS p95 FROM gateway_requests_per_hour_v1",
	)
	require.NoError(t, err)
	require.Contains(t, parsed, "quantileTDigestMerge")
	require.NotContains(t, parsed, "quantiletdigestmerge")

	// The rewrite must not disturb the parametric argument or the workspace filter.
	require.Contains(t, parsed, "latency_p95")
	require.Contains(t, strings.ToLower(parsed), "workspace_id = 'ws_kebap'")
}
