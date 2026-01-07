package handler_test

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/testutil"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/api/openapi"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_identities_update_identity"
)

// TestDeadlock reproduces the deadlock error that occurs when concurrent
// requests update ratelimits for the same identity.
//
// The deadlock happens because INSERT ... ON DUPLICATE KEY UPDATE with multiple
// value tuples acquires row locks in potentially different orders across
// concurrent transactions, causing MySQL error 1213.
func TestDeadlock(t *testing.T) {
	h := testutil.NewHarness(t)
	route := &handler.Handler{
		Logger:    h.Logger,
		DB:        h.DB,
		Keys:      h.Keys,
		Auditlogs: h.Auditlogs,
	}

	h.Register(route)

	rootKeyID := h.CreateRootKey(h.Resources().UserWorkspace.ID, "identity.*.update_identity")
	headers := http.Header{
		"Content-Type":  {"application/json"},
		"Authorization": {fmt.Sprintf("Bearer %s", rootKeyID)},
	}

	ctx := context.Background()
	workspaceID := h.Resources().UserWorkspace.ID
	identityID := uid.New(uid.IdentityPrefix)
	externalID := uid.New("")

	err := db.Query.InsertIdentity(ctx, h.DB.RW(), db.InsertIdentityParams{
		ID:          identityID,
		ExternalID:  externalID,
		WorkspaceID: workspaceID,
		Environment: "default",
		CreatedAt:   time.Now().UnixMilli(),
		Meta:        []byte("{}"),
	})
	require.NoError(t, err)

	ratelimitNames := []string{"main", "optional"}
	for _, name := range ratelimitNames {
		ratelimitID := uid.New(uid.RatelimitPrefix)
		err = db.Query.InsertIdentityRatelimit(ctx, h.DB.RW(), db.InsertIdentityRatelimitParams{
			ID:          ratelimitID,
			WorkspaceID: workspaceID,
			IdentityID:  sql.NullString{String: identityID, Valid: true},
			Name:        name,
			Limit:       50,
			Duration:    60000,
			CreatedAt:   time.Now().UnixMilli(),
		})
		require.NoError(t, err)
	}

	// Fire concurrent requests to update the same identity's ratelimits
	// This should trigger the deadlock condition
	numConcurrent := 10
	var wg sync.WaitGroup
	results := make(chan testutil.TestResponse[handler.Response], numConcurrent)
	errors := make(chan string, numConcurrent)

	for i := range numConcurrent {
		wg.Add(1)
		go func(iteration int) {
			defer wg.Done()

			// Each request updates the same ratelimits with slightly different values
			// to ensure the bulk INSERT ... ON DUPLICATE KEY UPDATE is triggered
			ratelimits := []openapi.RatelimitRequest{
				{
					Name:      "playground",
					Limit:     int64(50 + iteration),
					Duration:  60000,
					AutoApply: false,
				},
				{
					Name:      "serverless-inference",
					Limit:     int64(10 + iteration),
					Duration:  60000,
					AutoApply: false,
				},
			}

			req := handler.Request{
				Identity:   externalID,
				Ratelimits: &ratelimits,
			}

			res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)
			if res.Status != 200 {
				errors <- fmt.Sprintf("iteration %d: status=%d body=%s", iteration, res.Status, res.RawBody)
				return
			}
			results <- res
		}(i)
	}

	// Wait for all goroutines to finish
	wg.Wait()
	close(results)
	close(errors)

	// Collect errors - if deadlock occurs, we'll see error 1213
	var errorMessages []string
	for errMsg := range errors {
		errorMessages = append(errorMessages, errMsg)
	}

	// Collect successful results
	var successCount int
	for range results {
		successCount++
	}

	// Report results
	t.Logf("Successful requests: %d/%d", successCount, numConcurrent)
	for _, errMsg := range errorMessages {
		t.Logf("Error: %s", errMsg)
	}

	// The test should pass only if ALL concurrent requests succeed
	// If there are any errors (including deadlock), the test fails
	require.Empty(t, errorMessages, "Expected all concurrent requests to succeed, but got errors (likely deadlock)")
	require.Equal(t, numConcurrent, successCount, "Not all requests succeeded")
}
