package usagelimiter_test

import (
	"context"
	"database/sql"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/internal/services/usagelimiter"
	"github.com/unkeyed/unkey/pkg/counter"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/hash"
	"github.com/unkeyed/unkey/pkg/mysql/sqlcomment"
	"github.com/unkeyed/unkey/pkg/testutil/containers"
	"github.com/unkeyed/unkey/pkg/uid"
)

func TestDBFallback_ConcurrentCreditAdmission(t *testing.T) {
	database := newDatabase(t)
	services := []usagelimiter.Service{newFallback(t, database), newFallback(t, database)}
	for _, tt := range []struct {
		name      string
		initial   int64
		cost      int64
		remaining []int64
		stored    int64
	}{
		{name: "last credit", initial: 1, cost: 1, remaining: []int64{0}, stored: 0},
		{name: "cost leaves an unusable remainder", initial: 7, cost: 2, remaining: []int64{5, 3, 1}, stored: 1},
	} {
		t.Run(tt.name, func(t *testing.T) {
			keyID := insertKey(t, database, sql.NullInt64{Int64: tt.initial, Valid: true})
			ctx, cancel := context.WithTimeout(t.Context(), 15*time.Second)
			t.Cleanup(cancel)
			type result struct {
				response usagelimiter.UsageResponse
				err      error
			}
			results := make(chan result, 32)
			start := make(chan struct{})
			var workers sync.WaitGroup
			for i := range cap(results) {
				workers.Add(1)
				go func() {
					defer workers.Done()
					<-start
					response, err := services[i%len(services)].Limit(ctx, usagelimiter.UsageRequest{KeyID: keyID, Cost: tt.cost})
					results <- result{response: response, err: err}
				}()
			}
			close(start)
			workers.Wait()
			close(results)

			var remaining []int64
			for result := range results {
				require.NoError(t, result.err)
				if result.response.Valid {
					remaining = append(remaining, result.response.Remaining)
				}
			}
			require.ElementsMatch(t, tt.remaining, remaining, "successful requests must each spend distinct credits")
			stored, err := db.Query.FindKeyCredits(t.Context(), database.RW(), keyID)
			require.NoError(t, err)
			require.Equal(t, sql.NullInt64{Int64: tt.stored, Valid: true}, stored)
		})
	}
}

func TestDBFallback_CreditBoundaries(t *testing.T) {
	database := newDatabase(t)
	service := newFallback(t, database)
	for _, tt := range []struct {
		name     string
		initial  sql.NullInt64
		cost     int64
		response usagelimiter.UsageResponse
		stored   sql.NullInt64
	}{
		{
			name: "exact cost", initial: sql.NullInt64{Int64: 3, Valid: true}, cost: 3,
			response: usagelimiter.UsageResponse{Valid: true, Remaining: 0}, stored: sql.NullInt64{Int64: 0, Valid: true},
		},
		{
			name: "insufficient credit does not debit", initial: sql.NullInt64{Int64: 2, Valid: true}, cost: 3,
			response: usagelimiter.UsageResponse{Valid: false, Remaining: 0}, stored: sql.NullInt64{Int64: 2, Valid: true},
		},
		{
			name: "zero cost on exhausted key", initial: sql.NullInt64{Int64: 0, Valid: true}, cost: 0,
			response: usagelimiter.UsageResponse{Valid: true, Remaining: 0}, stored: sql.NullInt64{Int64: 0, Valid: true},
		},
		{
			name: "unlimited key", initial: sql.NullInt64{Int64: 0, Valid: false}, cost: 3,
			response: usagelimiter.UsageResponse{Valid: true, Remaining: -1}, stored: sql.NullInt64{Int64: 0, Valid: false},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			keyID := insertKey(t, database, tt.initial)
			response, err := service.Limit(t.Context(), usagelimiter.UsageRequest{KeyID: keyID, Cost: tt.cost})
			require.NoError(t, err)
			require.Equal(t, tt.response, response)
			stored, err := db.Query.FindKeyCredits(t.Context(), database.RW(), keyID)
			require.NoError(t, err)
			require.Equal(t, tt.stored, stored)
		})
	}

	t.Run("negative cost cannot add credits", func(t *testing.T) {
		initial := sql.NullInt64{Int64: 2, Valid: true}
		keyID := insertKey(t, database, initial)
		response, err := service.Limit(t.Context(), usagelimiter.UsageRequest{KeyID: keyID, Cost: -1})
		require.Error(t, err)
		require.False(t, response.Valid)
		stored, err := db.Query.FindKeyCredits(t.Context(), database.RW(), keyID)
		require.NoError(t, err)
		require.Equal(t, initial, stored)
	})

	t.Run("canceled request cannot spend credits", func(t *testing.T) {
		initial := sql.NullInt64{Int64: 2, Valid: true}
		keyID := insertKey(t, database, initial)
		ctx, cancel := context.WithCancel(t.Context())
		cancel()
		response, err := service.Limit(ctx, usagelimiter.UsageRequest{KeyID: keyID, Cost: 1})
		require.ErrorIs(t, err, context.Canceled)
		require.False(t, response.Valid)
		stored, err := db.Query.FindKeyCredits(t.Context(), database.RW(), keyID)
		require.NoError(t, err)
		require.Equal(t, initial, stored)
	})

	t.Run("missing key fails closed", func(t *testing.T) {
		response, err := service.Limit(t.Context(), usagelimiter.UsageRequest{KeyID: uid.New(uid.KeyPrefix), Cost: 1})
		require.Error(t, err)
		require.True(t, db.IsNotFound(err))
		require.False(t, response.Valid)
	})
}

func newDatabase(t *testing.T) db.Database {
	t.Helper()
	mysql := containers.MySQL(t)
	database, err := db.New(db.Config{PrimaryDSN: mysql.DSN, ReadOnlyDSN: "", Tags: sqlcomment.Disabled()})
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, database.Close()) })
	return database
}

func newFallback(t *testing.T, database db.Database) usagelimiter.Service {
	t.Helper()
	service, err := usagelimiter.NewCounter(usagelimiter.CounterConfig{
		DB: database.RW(),
		FindKeyCredits: func(ctx context.Context, keyID string) (int64, bool, error) {
			credits, err := db.Query.FindKeyCredits(ctx, database.RO(), keyID)
			return credits.Int64, credits.Valid, err
		},
		DecrementKeyCredits: func(ctx context.Context, keyID string, cost int64) error {
			return db.Query.UpdateKeyCreditsDecrement(ctx, database.RW(), db.UpdateKeyCreditsDecrementParams{
				ID: keyID, Credits: sql.NullInt64{Int64: cost, Valid: true},
			})
		},
		Counter:       unavailableCounter{Counter: counter.NewMemory()},
		TTL:           time.Minute,
		ReplayWorkers: 1,
	})
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, service.Close()) })
	return service
}

func insertKey(t *testing.T, database db.Database, credits sql.NullInt64) string {
	t.Helper()
	workspaceID := uid.New(uid.WorkspacePrefix)
	keySpaceID := uid.New(uid.KeySpacePrefix)
	keyID := uid.New(uid.KeyPrefix)
	require.NoError(t, db.Query.InsertWorkspace(t.Context(), database.RW(), db.InsertWorkspaceParams{
		ID: workspaceID, OrgID: workspaceID, Name: t.Name(), Slug: uid.New(uid.TestPrefix),
		CreatedAt: time.Now().UnixMilli(), K8sNamespace: uid.DNS1035(),
	}))
	require.NoError(t, db.Query.InsertKeySpace(t.Context(), database.RW(), db.InsertKeySpaceParams{
		ID: keySpaceID, WorkspaceID: workspaceID, CreatedAtM: time.Now().UnixMilli(),
	}))
	require.NoError(t, db.Query.InsertKey(t.Context(), database.RW(), db.InsertKeyParams{
		ID: keyID, KeySpaceID: keySpaceID, WorkspaceID: workspaceID, Hash: hash.Sha256(keyID),
		Prefix: "", Start: keyID[:4], End: keyID[len(keyID)-4:], Enabled: true,
		CreatedAtM: time.Now().UnixMilli(), RemainingRequests: credits,
		ForWorkspaceID: sql.NullString{String: "", Valid: false}, Name: sql.NullString{String: "", Valid: false},
		IdentityID: sql.NullString{String: "", Valid: false}, Meta: sql.NullString{String: "", Valid: false},
		Expires: sql.NullTime{Time: time.Time{}, Valid: false}, RefillDay: sql.NullInt16{Int16: 0, Valid: false},
		RefillAmount: sql.NullInt64{Int64: 0, Valid: false}, PendingMigrationID: sql.NullString{String: "", Valid: false},
	}))
	return keyID
}

type unavailableCounter struct {
	counter.Counter
}

func (unavailableCounter) DecrementIfExists(context.Context, string, int64) (int64, bool, bool, error) {
	return 0, false, false, errors.New("counter unavailable")
}
