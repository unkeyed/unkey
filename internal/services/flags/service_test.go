package flags_test

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/internal/services/flags"
	"github.com/unkeyed/unkey/pkg/clock"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/testutil/containers"
	"github.com/unkeyed/unkey/pkg/uid"
)

func TestService_CachesMetadataAndReturnsIndependentCopies(t *testing.T) {
	database, service, testClock := newService(t)
	workspaceID := uid.New(uid.WorkspacePrefix)
	require.NoError(t, db.Query.UpsertFlag(t.Context(), database.RW(), db.UpsertFlagParams{
		Slug: flags.HelloWorldSlug, Description: "Original description", DefaultEnabled: true,
		AllowOptIn: false, AllowOptOut: true,
	}))
	setOverride(t, database, workspaceID, false)
	values, err := service.Values(t.Context(), workspaceID)
	require.NoError(t, err)
	want := map[string]flags.Flag{flags.HelloWorldSlug: {
		Enabled: false, Description: "Original description", AllowOptIn: false, AllowOptOut: true,
	}}
	require.Equal(t, want, values)
	delete(values, flags.HelloWorldSlug)
	require.NoError(t, db.Query.UpsertFlag(t.Context(), database.RW(), db.UpsertFlagParams{
		Slug: flags.HelloWorldSlug, Description: "Changed description", DefaultEnabled: true,
		AllowOptIn: true, AllowOptOut: false,
	}))
	cached, err := service.Values(t.Context(), workspaceID)
	require.NoError(t, err)
	require.Equal(t, want, cached)
	testClock.Tick(time.Minute)
	refreshed, err := service.Values(t.Context(), workspaceID)
	require.NoError(t, err)
	require.Equal(t, map[string]flags.Flag{flags.HelloWorldSlug: {
		Enabled: false, Description: "Changed description", AllowOptIn: true, AllowOptOut: false,
	}}, refreshed)
}

// TestService_Precedence guarantees that a workspace override, including false,
// beats the database default, and that overrides never leak across workspaces.
func TestService_Precedence(t *testing.T) {
	database, service, _ := newService(t)
	ctx := t.Context()

	for _, tt := range []struct {
		name         string
		defaultValue bool
		override     sql.NullBool
		want         bool
	}{
		{name: "database true beats code false", defaultValue: true, want: true},
		{name: "database false inherited", defaultValue: false, want: false},
		{name: "override true beats database false", defaultValue: false, override: sql.NullBool{Bool: true, Valid: true}, want: true},
		{name: "override false beats database true", defaultValue: true, override: sql.NullBool{Bool: false, Valid: true}, want: false},
	} {
		t.Run(tt.name, func(t *testing.T) {
			workspaceID := uid.New(uid.WorkspacePrefix)
			upsertHelloWorld(t, database, tt.defaultValue)
			if tt.override.Valid {
				setOverride(t, database, workspaceID, tt.override.Bool)
			}
			require.Equal(t, tt.want, service.IsHelloWorldEnabled(ctx, workspaceID))
			require.Equal(t, tt.defaultValue, service.IsHelloWorldEnabled(ctx, uid.New(uid.WorkspacePrefix)))

			values, err := service.Values(ctx, workspaceID)
			require.NoError(t, err)
			require.Equal(t, map[string]flags.Flag{flags.HelloWorldSlug: {
				Enabled: tt.want, Description: t.Name(), AllowOptIn: false, AllowOptOut: false,
			}}, values)
			values[flags.HelloWorldSlug] = flags.Flag{Enabled: !tt.want, Description: "Changed", AllowOptIn: true, AllowOptOut: true}
			require.Equal(t, tt.want, service.IsHelloWorldEnabled(ctx, workspaceID))
		})
	}
}

// TestService_MissingDefinitionUsesCodeDefault guarantees that code can ship
// before its definition exists, even if an override row already names the slug.
func TestService_MissingDefinitionUsesCodeDefault(t *testing.T) {
	database, service, _ := newService(t)
	workspaceID := uid.New(uid.WorkspacePrefix)
	setOverride(t, database, workspaceID, true)
	require.False(t, service.IsHelloWorldEnabled(t.Context(), workspaceID))
}

// TestService_RefreshesOverridesAndRestoresInheritance guarantees that override
// changes reach callers after the fresh window and that deleting an override
// restores the database default.
func TestService_RefreshesOverridesAndRestoresInheritance(t *testing.T) {
	database, service, testClock := newService(t)
	ctx := t.Context()
	workspaceID := uid.New(uid.WorkspacePrefix)
	upsertHelloWorld(t, database, true)
	require.True(t, service.IsHelloWorldEnabled(ctx, workspaceID))

	setOverride(t, database, workspaceID, false)
	testClock.Tick(9 * time.Second)
	require.True(t, service.IsHelloWorldEnabled(ctx, workspaceID))
	testClock.Tick(time.Second)
	require.True(t, service.IsHelloWorldEnabled(ctx, workspaceID))
	require.Eventually(t, func() bool { return !service.IsHelloWorldEnabled(ctx, workspaceID) }, time.Second, time.Millisecond)

	require.NoError(t, db.Query.DeleteWorkspaceFlagOverride(ctx, database.RW(), db.DeleteWorkspaceFlagOverrideParams{
		WorkspaceID: workspaceID,
		FlagSlug:    flags.HelloWorldSlug,
	}))
	testClock.Tick(time.Minute)
	require.True(t, service.IsHelloWorldEnabled(ctx, workspaceID))
}

// TestService_LoadFailureDoesNotCacheFallback guarantees that one failed load
// does not pin the code default for the rest of the cache window.
func TestService_LoadFailureDoesNotCacheFallback(t *testing.T) {
	database, service, _ := newService(t)
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	workspaceID := uid.New(uid.WorkspacePrefix)
	upsertHelloWorld(t, database, true)

	require.False(t, service.IsHelloWorldEnabled(ctx, workspaceID))
	require.True(t, service.IsHelloWorldEnabled(t.Context(), workspaceID))
}

// TestService_ExpiresDuringDatabaseOutage guarantees that a database outage
// cannot keep serving cached values past the one-minute maximum age, and that
// Values reports the outage instead of code defaults.
func TestService_ExpiresDuringDatabaseOutage(t *testing.T) {
	database, service, testClock := newService(t)
	ctx := t.Context()
	workspaceID := uid.New(uid.WorkspacePrefix)
	upsertHelloWorld(t, database, true)
	require.True(t, service.IsHelloWorldEnabled(ctx, workspaceID))
	require.NoError(t, database.Close())

	testClock.Tick(59 * time.Second)
	require.True(t, service.IsHelloWorldEnabled(ctx, workspaceID))
	testClock.Tick(time.Second)
	require.False(t, service.IsHelloWorldEnabled(ctx, workspaceID))
	_, err := service.Values(ctx, workspaceID)
	require.Error(t, err)
}

func TestNoop_UsesCodeDefaultWithoutDatabase(t *testing.T) {
	service := flags.NewNoop()
	t.Cleanup(service.Close)
	require.False(t, service.IsHelloWorldEnabled(t.Context(), uid.New(uid.WorkspacePrefix)))
}

type helloWorldService interface {
	flags.Service
	IsHelloWorldEnabled(ctx context.Context, workspaceID string) bool
}

func newService(t *testing.T) (db.Database, helloWorldService, *clock.TestClock) {
	t.Helper()
	database, err := db.New(db.Config{PrimaryDSN: containers.MySQLIsolated(t).DSN})
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, database.Close()) })
	testClock := clock.NewTestClock()
	service, err := flags.New(flags.Config{DB: db.ToMySQL(database), Clock: testClock})
	require.NoError(t, err)
	t.Cleanup(service.Close)
	return database, service, testClock
}

func upsertHelloWorld(t *testing.T, database db.Database, defaultValue bool) {
	t.Helper()
	require.NoError(t, db.Query.UpsertFlag(t.Context(), database.RW(), db.UpsertFlagParams{
		Slug:           flags.HelloWorldSlug,
		Description:    t.Name(),
		DefaultEnabled: defaultValue,
		AllowOptIn:     false,
		AllowOptOut:    false,
	}))
}

func setOverride(t *testing.T, database db.Database, workspaceID string, value bool) {
	t.Helper()
	require.NoError(t, db.Query.UpsertWorkspaceFlagOverride(t.Context(), database.RW(), db.UpsertWorkspaceFlagOverrideParams{
		WorkspaceID: workspaceID,
		FlagSlug:    flags.HelloWorldSlug,
		Enabled:     value,
	}))
}
