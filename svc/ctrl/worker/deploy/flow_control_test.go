package deploy_test

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	restateadmin "github.com/unkeyed/unkey/pkg/restate/admin"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/ctrl/internal/db"
)

// TestCreateSyncsDeployConcurrencyRule checks the rule Create writes before it
// sends Deploy: the workspace limit when there is one, and a floor of one when
// the limit is zero or the row is missing, matching BuildSlotService.
func TestCreateSyncsDeployConcurrencyRule(t *testing.T) {
	ctx := context.Background()
	rules := newRuleRecorder(t)
	h := newCreateHarnessWithAdmin(t, ctx, restateadmin.New(restateadmin.Config{BaseURL: rules.server.URL, APIKey: ""}))
	pattern := h.workspaceID + "/deploy"

	t.Run("limit row is mirrored", func(t *testing.T) {
		h.setBuildsConcurrentMax(t, ctx, 3)
		h.create(t, ctx, uid.New(uid.DeploymentPrefix), h.imageRequest())
		require.Equal(t, uint32(3), rules.await(t, pattern))
	})

	t.Run("zero is floored to one", func(t *testing.T) {
		rules.reset()
		h.setBuildsConcurrentMax(t, ctx, 0)
		h.create(t, ctx, uid.New(uid.DeploymentPrefix), h.imageRequest())
		require.Equal(t, uint32(1), rules.await(t, pattern))
	})

	t.Run("missing row defaults to one", func(t *testing.T) {
		rules.reset()
		_, err := h.database.RW().ExecContext(ctx, "DELETE FROM limits WHERE workspace_id = ?", h.workspaceID)
		require.NoError(t, err)
		h.create(t, ctx, uid.New(uid.DeploymentPrefix), h.imageRequest())
		require.Equal(t, uint32(1), rules.await(t, pattern))
	})
}

func (h *createHarness) setBuildsConcurrentMax(t *testing.T, ctx context.Context, limit uint16) {
	t.Helper()
	require.NoError(t, h.database.UpsertLimit(ctx, db.UpsertLimitParams{
		WorkspaceID:                           h.workspaceID,
		ApiBillableOperationsCountMaxPerMonth: 150_000,
		ApiRequestsCountMaxPerMinute:          sql.NullInt32{Valid: false, Int32: 0},
		LogsRetentionDaysMax:                  7,
		LogsAuditRetentionDaysMax:             30,
		TeamEnabled:                           false,
		CpuCoresMax:                           10,
		CpuCoresMaxPerInstance:                2,
		MemoryMibMax:                          20_480,
		MemoryMibMaxPerInstance:               4_096,
		StorageMibMax:                         51_200,
		StorageMibMaxPerInstance:              10_240,
		BuildsConcurrentMax:                   limit,
		CustomDomainsMax:                      0,
		AutoscalingReplicasMax:                0,
	}))
}

// ruleRecorder stands in for the Restate admin API and keeps the last
// concurrency written per pattern.
type ruleRecorder struct {
	server *httptest.Server
	mu     sync.Mutex
	rules  map[string]uint32
}

func newRuleRecorder(t *testing.T) *ruleRecorder {
	t.Helper()
	r := &ruleRecorder{server: nil, mu: sync.Mutex{}, rules: map[string]uint32{}}
	r.server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if req.Method != http.MethodPut || req.URL.Path != "/limits/rules" {
			http.Error(w, "unexpected request "+req.Method+" "+req.URL.Path, http.StatusNotFound)
			return
		}
		var body []struct {
			Pattern string `json:"pattern"`
			Limits  struct {
				Concurrency uint32 `json:"concurrency"`
			} `json:"limits"`
		}
		if err := json.NewDecoder(req.Body).Decode(&body); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		r.mu.Lock()
		defer r.mu.Unlock()
		for _, rule := range body {
			r.rules[rule.Pattern] = rule.Limits.Concurrency
		}
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(r.server.Close)
	return r
}

func (r *ruleRecorder) reset() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.rules = map[string]uint32{}
}

func (r *ruleRecorder) await(t *testing.T, pattern string) uint32 {
	t.Helper()
	var got uint32
	require.Eventually(t, func() bool {
		r.mu.Lock()
		defer r.mu.Unlock()
		concurrency, ok := r.rules[pattern]
		got = concurrency
		return ok
	}, 15*time.Second, 50*time.Millisecond, "Create must write the rule %s", pattern)
	return got
}
