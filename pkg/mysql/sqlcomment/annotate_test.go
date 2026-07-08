package sqlcomment

import (
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestAnnotate_sqlcOperation(t *testing.T) {
	t.Parallel()

	static := ForService("frontline", "eu-central-1")
	query := `-- name: FindKeyForVerification :one
select id from keys where hash = ?`

	got := Annotate(query, static, "ro", Dynamic{})
	require.Contains(t, got, "select id from keys where hash = ?")
	require.NotContains(t, got, "-- name:")
	require.Contains(t, got, "operation='FindKeyForVerification'")
	require.Contains(t, got, "service='frontline'")
	require.Contains(t, got, "region='eu-central-1'")
	require.Contains(t, got, "mode='ro'")
}

func TestAnnotate_dynamicTags(t *testing.T) {
	t.Parallel()

	static := ForService("api", "us-east-1")
	got := Annotate("select 1", static, "rw", Dynamic{
		Route:  "trpc.keys.create",
		Source: "app",
	})
	require.Contains(t, got, "route='trpc.keys.create'")
	require.Contains(t, got, "source='app'")
	require.NotContains(t, got, "environment=")
}

func TestAnnotate_disabledWithoutService(t *testing.T) {
	t.Parallel()

	query := "select 1"
	require.Equal(t, query, Annotate(query, Static{}, "rw", Dynamic{}))
}

func TestAnnotate_escapesQuotes(t *testing.T) {
	t.Parallel()

	static := ForService("api", "us-east-1")
	got := Annotate("select 1", static, "rw", Dynamic{Route: "a'b"})
	require.Contains(t, got, `route='a%27b'`)
}

func TestAnnotate_urlEncodesRoute(t *testing.T) {
	t.Parallel()

	static := ForService("api", "us-east-1")
	got := Annotate("select 1", static, "rw", Dynamic{Route: "POST /v2/keys.verifyKey"})
	require.Contains(t, got, `route='POST%20%2Fv2%2Fkeys.verifyKey'`)
}

func TestWithDynamic_roundTrip(t *testing.T) {
	t.Parallel()

	ctx := WithDynamic(context.Background(), Dynamic{Source: "worker"})
	got := DynamicFromContext(ctx)
	require.Equal(t, "worker", got.Source)
}

func TestAnnotate_preservesMultilineSQL(t *testing.T) {
	t.Parallel()

	static := ForService("ctrl-worker", "us-east-1")
	query := `-- name: GlobalCountersImported :many
select workspace_id
from ratelimit_global_counters
where expires_at > ?`

	got := Annotate(query, static, "ro", Dynamic{})
	require.True(t, strings.HasPrefix(got, "select workspace_id"))
	require.Contains(t, got, "operation='GlobalCountersImported'")
}

func TestAnnotate_cachesDynamicTags(t *testing.T) {
	t.Parallel()

	static := ForService("api", "us-east-1")
	query := "select 1"
	dynamic := Dynamic{Route: "POST /v2/keys.verifyKey", Source: "http"}

	first := Annotate(query, static, "ro", dynamic)
	second := Annotate(query, static, "ro", dynamic)
	require.Equal(t, first, second)
	require.Contains(t, first, `route='POST%20%2Fv2%2Fkeys.verifyKey'`)
	require.Contains(t, first, `source='http'`)
}
