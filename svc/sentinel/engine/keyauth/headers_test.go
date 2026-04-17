package keyauth

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/internal/services/keys"
	"github.com/unkeyed/unkey/internal/services/ratelimit"
)

func TestFindMostRestrictive(t *testing.T) {
	t.Parallel()

	allowed := func(remaining int64) *ratelimit.RatelimitResponse {
		return &ratelimit.RatelimitResponse{Success: true, Remaining: remaining}
	}
	denied := func(remaining int64) *ratelimit.RatelimitResponse {
		return &ratelimit.RatelimitResponse{Success: false, Remaining: remaining}
	}
	entry := func(resp *ratelimit.RatelimitResponse) keys.RatelimitConfigAndResult {
		//nolint:exhaustruct
		return keys.RatelimitConfigAndResult{Response: resp}
	}

	tests := []struct {
		name            string
		results         map[string]keys.RatelimitConfigAndResult
		wantNil         bool
		wantSuccess     bool
		wantRemaining   int64
	}{
		{
			name:    "empty map",
			results: map[string]keys.RatelimitConfigAndResult{},
			wantNil: true,
		},
		{
			name:    "single nil response",
			results: map[string]keys.RatelimitConfigAndResult{"a": entry(nil)},
			wantNil: true,
		},
		{
			name:          "single allowed",
			results:       map[string]keys.RatelimitConfigAndResult{"a": entry(allowed(5))},
			wantSuccess:   true,
			wantRemaining: 5,
		},
		{
			name: "denial beats allowed",
			results: map[string]keys.RatelimitConfigAndResult{
				"a": entry(allowed(10)),
				"b": entry(denied(0)),
			},
			wantSuccess:   false,
			wantRemaining: 0,
		},
		{
			name: "lower remaining wins when both allowed",
			results: map[string]keys.RatelimitConfigAndResult{
				"a": entry(allowed(10)),
				"b": entry(allowed(3)),
			},
			wantSuccess:   true,
			wantRemaining: 3,
		},
		{
			name: "lower remaining wins when both denied",
			results: map[string]keys.RatelimitConfigAndResult{
				"a": entry(denied(2)),
				"b": entry(denied(0)),
			},
			wantSuccess:   false,
			wantRemaining: 0,
		},
		{
			name: "nil response entries are skipped",
			results: map[string]keys.RatelimitConfigAndResult{
				"a": entry(nil),
				"b": entry(allowed(7)),
				"c": entry(nil),
			},
			wantSuccess:   true,
			wantRemaining: 7,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := findMostRestrictive(tt.results)
			if tt.wantNil {
				require.Nil(t, got)
				return
			}
			require.NotNil(t, got)
			require.NotNil(t, got.Response)
			require.Equal(t, tt.wantSuccess, got.Response.Success)
			require.Equal(t, tt.wantRemaining, got.Response.Remaining)
		})
	}
}
