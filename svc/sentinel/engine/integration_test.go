package engine_test

import (
	"context"
	"database/sql"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	sentinelv1 "github.com/unkeyed/unkey/gen/proto/sentinel/v1"
	"github.com/unkeyed/unkey/internal/services/keys"
	"github.com/unkeyed/unkey/internal/services/ratelimit"
	"github.com/unkeyed/unkey/internal/services/usagelimiter"
	"github.com/unkeyed/unkey/pkg/cache"
	"github.com/unkeyed/unkey/pkg/clickhouse"
	"github.com/unkeyed/unkey/pkg/clock"
	"github.com/unkeyed/unkey/pkg/counter"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/dockertest"
	"github.com/unkeyed/unkey/pkg/hash"
	"github.com/unkeyed/unkey/pkg/rbac"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/pkg/zen"
	"github.com/unkeyed/unkey/svc/sentinel/engine"
)

// testHarness holds all real services needed for integration tests.
type testHarness struct {
	t          *testing.T
	db         db.Database
	keyService keys.KeyService
	engine     *engine.Engine
	clk        clock.Clock
}

func newTestHarness(t *testing.T) *testHarness {
	t.Helper()

	cluster := dockertest.New(t)
	mysqlCfg := cluster.MySQL()
	redisCfg := cluster.Redis()

	clk := clock.New()

	database, err := db.New(db.Config{
		PrimaryDSN:  mysqlCfg.HostDSN,
		ReadOnlyDSN: "",
	})
	require.NoError(t, err)
	t.Cleanup(func() { _ = database.Close() })

	redisCounter, err := counter.NewRedis(counter.RedisConfig{
		RedisURL: redisCfg.HostURL,
	})
	require.NoError(t, err)
	t.Cleanup(func() { _ = redisCounter.Close() })

	rateLimiter, err := ratelimit.New(ratelimit.Config{
		Clock:   clk,
		Counter: redisCounter,
	})
	require.NoError(t, err)
	t.Cleanup(func() { _ = rateLimiter.Close() })

	usageLimiter, err := usagelimiter.NewCounter(usagelimiter.CounterConfig{
		DB:            database,
		Counter:       redisCounter,
		TTL:           60 * time.Second,
		ReplayWorkers: 2,
	})
	require.NoError(t, err)
	t.Cleanup(func() { _ = usageLimiter.Close() })

	keyCache, err := cache.New[string, db.CachedKeyData](cache.Config[string, db.CachedKeyData]{
		Fresh:    10 * time.Second,
		Stale:    10 * time.Minute,
		MaxSize:  1000,
		Resource: "test_key_cache",
		Clock:    clk,
	})
	require.NoError(t, err)

	keyService, err := keys.New(keys.Config{
		DB:           database,
		RateLimiter:  rateLimiter,
		RBAC:         rbac.New(),
		Clickhouse:   clickhouse.NewNoop(),
		Region:       "test",
		UsageLimiter: usageLimiter,
		KeyCache:     keyCache,
	})
	require.NoError(t, err)

	eng := engine.New(engine.Config{
		KeyService: keyService,
		Clock:      clk,
	})

	return &testHarness{
		t:          t,
		db:         database,
		keyService: keyService,
		engine:     eng,
		clk:        clk,
	}
}

// seedResult holds the IDs/values created during seeding.
type seedResult struct {
	WorkspaceID string
	KeySpaceID  string
	ApiID       string
	KeyID       string
	RawKey      string // the unhashed key value to use in requests
}

// seed creates a workspace, key space, API, and key in the database.
func (h *testHarness) seed(ctx context.Context) seedResult {
	h.t.Helper()

	now := time.Now().UnixMilli()
	wsID := uid.New("test_ws")
	orgID := uid.New("test_org")

	err := db.Query.InsertWorkspace(ctx, h.db.RW(), db.InsertWorkspaceParams{
		ID:           wsID,
		OrgID:        orgID,
		Name:         uid.New("test_name"),
		Slug:         uid.New("slug"),
		CreatedAt:    now,
		K8sNamespace: sql.NullString{Valid: true, String: uid.New("ns")},
	})
	require.NoError(h.t, err)

	ksID := uid.New(uid.KeySpacePrefix)
	err = db.Query.InsertKeySpace(ctx, h.db.RW(), db.InsertKeySpaceParams{
		ID:                 ksID,
		WorkspaceID:        wsID,
		CreatedAtM:         now,
		StoreEncryptedKeys: false,
		DefaultPrefix:      sql.NullString{Valid: false},
		DefaultBytes:       sql.NullInt32{Valid: false},
	})
	require.NoError(h.t, err)

	apiID := uid.New("api")
	err = db.Query.InsertApi(ctx, h.db.RW(), db.InsertApiParams{
		ID:          apiID,
		Name:        "test-api",
		WorkspaceID: wsID,
		AuthType:    db.NullApisAuthType{Valid: true, ApisAuthType: db.ApisAuthTypeKey},
		IpWhitelist: sql.NullString{Valid: false},
		KeyAuthID:   sql.NullString{Valid: true, String: ksID},
		CreatedAtM:  now,
	})
	require.NoError(h.t, err)

	rawKey := uid.New("sk_live")
	keyID := uid.New(uid.KeyPrefix)
	err = db.Query.InsertKey(ctx, h.db.RW(), db.InsertKeyParams{
		ID:                 keyID,
		KeySpaceID:         ksID,
		Hash:               hash.Sha256(rawKey),
		Start:              rawKey[:8],
		WorkspaceID:        wsID,
		ForWorkspaceID:     sql.NullString{Valid: false},
		Name:               sql.NullString{String: "test-key", Valid: true},
		IdentityID:         sql.NullString{Valid: false},
		Meta:               sql.NullString{Valid: false},
		Expires:            sql.NullTime{Valid: false},
		CreatedAtM:         now,
		Enabled:            true,
		RemainingRequests:  sql.NullInt32{Valid: false},
		RefillDay:          sql.NullInt16{Valid: false},
		RefillAmount:       sql.NullInt32{Valid: false},
		PendingMigrationID: sql.NullString{Valid: false},
	})
	require.NoError(h.t, err)

	return seedResult{
		WorkspaceID: wsID,
		KeySpaceID:  ksID,
		ApiID:       apiID,
		KeyID:       keyID,
		RawKey:      rawKey,
	}
}

// seedDisabledKey creates a key that is disabled.
func (h *testHarness) seedDisabledKey(ctx context.Context, wsID, ksID string) seedResult {
	h.t.Helper()

	rawKey := uid.New("sk_live")
	keyID := uid.New(uid.KeyPrefix)
	err := db.Query.InsertKey(ctx, h.db.RW(), db.InsertKeyParams{
		ID:                 keyID,
		KeySpaceID:         ksID,
		Hash:               hash.Sha256(rawKey),
		Start:              rawKey[:8],
		WorkspaceID:        wsID,
		ForWorkspaceID:     sql.NullString{Valid: false},
		Name:               sql.NullString{String: "disabled-key", Valid: true},
		IdentityID:         sql.NullString{Valid: false},
		Meta:               sql.NullString{Valid: false},
		Expires:            sql.NullTime{Valid: false},
		CreatedAtM:         time.Now().UnixMilli(),
		Enabled:            false,
		RemainingRequests:  sql.NullInt32{Valid: false},
		RefillDay:          sql.NullInt16{Valid: false},
		RefillAmount:       sql.NullInt32{Valid: false},
		PendingMigrationID: sql.NullString{Valid: false},
	})
	require.NoError(h.t, err)

	return seedResult{
		WorkspaceID: wsID,
		KeySpaceID:  ksID,
		KeyID:       keyID,
		RawKey:      rawKey,
	}
}

// seedKeyWithIdentity creates a key linked to an identity with an external ID.
func (h *testHarness) seedKeyWithIdentity(ctx context.Context, wsID, ksID string) seedResult {
	h.t.Helper()

	now := time.Now().UnixMilli()
	externalID := uid.New("ext")
	identityID := uid.New("id")

	err := db.Query.InsertIdentity(ctx, h.db.RW(), db.InsertIdentityParams{
		ID:          identityID,
		ExternalID:  externalID,
		WorkspaceID: wsID,
		Environment: "",
		CreatedAt:   now,
		Meta:        []byte("{}"),
	})
	require.NoError(h.t, err)

	rawKey := uid.New("sk_live")
	keyID := uid.New(uid.KeyPrefix)
	err = db.Query.InsertKey(ctx, h.db.RW(), db.InsertKeyParams{
		ID:                 keyID,
		KeySpaceID:         ksID,
		Hash:               hash.Sha256(rawKey),
		Start:              rawKey[:8],
		WorkspaceID:        wsID,
		ForWorkspaceID:     sql.NullString{Valid: false},
		Name:               sql.NullString{String: "identity-key", Valid: true},
		IdentityID:         sql.NullString{String: identityID, Valid: true},
		Meta:               sql.NullString{Valid: false},
		Expires:            sql.NullTime{Valid: false},
		CreatedAtM:         now,
		Enabled:            true,
		RemainingRequests:  sql.NullInt32{Valid: false},
		RefillDay:          sql.NullInt16{Valid: false},
		RefillAmount:       sql.NullInt32{Valid: false},
		PendingMigrationID: sql.NullString{Valid: false},
	})
	require.NoError(h.t, err)

	return seedResult{
		WorkspaceID: wsID,
		KeySpaceID:  ksID,
		KeyID:       keyID,
		RawKey:      rawKey,
	}
}

func newSession(t *testing.T, req *http.Request) *zen.Session {
	t.Helper()
	w := httptest.NewRecorder()
	//nolint:exhaustruct
	sess := &zen.Session{}
	err := sess.Init(w, req, 0)
	require.NoError(t, err)
	return sess
}

// --- KeyAuth integration tests ---

func TestKeyAuth_ValidKey(t *testing.T) {
	h := newTestHarness(t)
	ctx := context.Background()
	s := h.seed(ctx)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer "+s.RawKey)
	sess := newSession(t, req)

	policies := []*sentinelv1.Policy{
		{
			Id:      "auth",
			Enabled: true,
			Config: &sentinelv1.Policy_Keyauth{
				Keyauth: &sentinelv1.KeyAuth{KeySpaceIds: []string{s.KeySpaceID}},
			},
		},
	}

	result, err := h.engine.Evaluate(ctx, sess, req, policies)
	require.NoError(t, err)
	require.NotNil(t, result.Principal)

	// Subject falls back to key ID when no external ID is set
	require.Equal(t, s.KeyID, result.Principal.Subject)
	require.Equal(t, sentinelv1.PrincipalType_PRINCIPAL_TYPE_API_KEY, result.Principal.Type)
	require.Equal(t, s.KeyID, result.Principal.Claims["key_id"])
	require.Equal(t, s.WorkspaceID, result.Principal.Claims["workspace_id"])
}

func TestKeyAuth_ValidKey_WithIdentity(t *testing.T) {
	h := newTestHarness(t)
	ctx := context.Background()
	base := h.seed(ctx)
	s := h.seedKeyWithIdentity(ctx, base.WorkspaceID, base.KeySpaceID)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer "+s.RawKey)
	sess := newSession(t, req)

	policies := []*sentinelv1.Policy{
		{
			Id:      "auth",
			Enabled: true,
			Config: &sentinelv1.Policy_Keyauth{
				Keyauth: &sentinelv1.KeyAuth{KeySpaceIds: []string{s.KeySpaceID}},
			},
		},
	}

	result, err := h.engine.Evaluate(ctx, sess, req, policies)
	require.NoError(t, err)
	require.NotNil(t, result.Principal)

	// Subject should be the external ID from the identity
	require.NotEqual(t, s.KeyID, result.Principal.Subject)
	require.NotEmpty(t, result.Principal.Claims["identity_id"])
	require.NotEmpty(t, result.Principal.Claims["external_id"])
}

func TestKeyAuth_MissingKey_Reject(t *testing.T) {
	h := newTestHarness(t)
	ctx := context.Background()
	s := h.seed(ctx)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	// No Authorization header
	sess := newSession(t, req)

	policies := []*sentinelv1.Policy{
		{
			Id:      "auth",
			Enabled: true,
			Config: &sentinelv1.Policy_Keyauth{
				Keyauth: &sentinelv1.KeyAuth{
					KeySpaceIds: []string{s.KeySpaceID},
				},
			},
		},
	}

	_, err := h.engine.Evaluate(ctx, sess, req, policies)
	require.Error(t, err)
	require.Contains(t, err.Error(), "missing API key")
}

func TestKeyAuth_InvalidKey_NotFound(t *testing.T) {
	h := newTestHarness(t)
	ctx := context.Background()
	s := h.seed(ctx)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer sk_this_key_does_not_exist")
	sess := newSession(t, req)

	//nolint:exhaustruct
	policies := []*sentinelv1.Policy{
		{
			Id:      "auth",
			Enabled: true,
			Config: &sentinelv1.Policy_Keyauth{
				Keyauth: &sentinelv1.KeyAuth{KeySpaceIds: []string{s.KeySpaceID}},
			},
		},
	}

	_, err := h.engine.Evaluate(ctx, sess, req, policies)
	require.Error(t, err)
}

func TestKeyAuth_InvalidKey_Disabled(t *testing.T) {
	h := newTestHarness(t)
	ctx := context.Background()
	base := h.seed(ctx)
	disabled := h.seedDisabledKey(ctx, base.WorkspaceID, base.KeySpaceID)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer "+disabled.RawKey)
	sess := newSession(t, req)

	policies := []*sentinelv1.Policy{
		{
			Id:      "auth",
			Enabled: true,
			Config: &sentinelv1.Policy_Keyauth{
				Keyauth: &sentinelv1.KeyAuth{KeySpaceIds: []string{base.KeySpaceID}},
			},
		},
	}

	_, err := h.engine.Evaluate(ctx, sess, req, policies)
	require.Error(t, err)
}

func TestKeyAuth_WrongKeySpace(t *testing.T) {
	h := newTestHarness(t)
	ctx := context.Background()
	s := h.seed(ctx)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer "+s.RawKey)
	sess := newSession(t, req)

	policies := []*sentinelv1.Policy{
		{
			Id:      "auth",
			Enabled: true,
			Config: &sentinelv1.Policy_Keyauth{
				Keyauth: &sentinelv1.KeyAuth{KeySpaceIds: []string{"ks_wrong_space"}},
			},
		},
	}

	_, err := h.engine.Evaluate(ctx, sess, req, policies)
	require.Error(t, err)
	require.Contains(t, err.Error(), "key does not belong to expected key space")
}

func TestKeyAuth_MultipleKeySpaceIds(t *testing.T) {
	h := newTestHarness(t)
	ctx := context.Background()

	s1 := h.seed(ctx)
	s2 := h.seed(ctx)

	t.Run("key from first keyspace accepted", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.Header.Set("Authorization", "Bearer "+s1.RawKey)
		sess := newSession(t, req)

		policies := []*sentinelv1.Policy{
			{
				Id:      "auth",
				Enabled: true,
				Config: &sentinelv1.Policy_Keyauth{
					Keyauth: &sentinelv1.KeyAuth{KeySpaceIds: []string{s1.KeySpaceID, s2.KeySpaceID}},
				},
			},
		}

		result, err := h.engine.Evaluate(ctx, sess, req, policies)
		require.NoError(t, err)
		require.NotNil(t, result.Principal)
		require.Equal(t, s1.KeyID, result.Principal.Subject)
	})

	t.Run("key from second keyspace accepted", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.Header.Set("Authorization", "Bearer "+s2.RawKey)
		sess := newSession(t, req)

		policies := []*sentinelv1.Policy{
			{
				Id:      "auth",
				Enabled: true,
				Config: &sentinelv1.Policy_Keyauth{
					Keyauth: &sentinelv1.KeyAuth{KeySpaceIds: []string{s1.KeySpaceID, s2.KeySpaceID}},
				},
			},
		}

		result, err := h.engine.Evaluate(ctx, sess, req, policies)
		require.NoError(t, err)
		require.NotNil(t, result.Principal)
		require.Equal(t, s2.KeyID, result.Principal.Subject)
	})

	t.Run("key not in any listed keyspace rejected", func(t *testing.T) {
		s3 := h.seed(ctx)

		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.Header.Set("Authorization", "Bearer "+s3.RawKey)
		sess := newSession(t, req)

		policies := []*sentinelv1.Policy{
			{
				Id:      "auth",
				Enabled: true,
				Config: &sentinelv1.Policy_Keyauth{
					Keyauth: &sentinelv1.KeyAuth{KeySpaceIds: []string{s1.KeySpaceID, s2.KeySpaceID}},
				},
			},
		}

		_, err := h.engine.Evaluate(ctx, sess, req, policies)
		require.Error(t, err)
		require.Contains(t, err.Error(), "key does not belong to expected key space")
	})
}

// --- Engine Evaluate integration tests ---

func TestEvaluate_DisabledPoliciesSkipped(t *testing.T) {
	h := newTestHarness(t)
	ctx := context.Background()
	s := h.seed(ctx)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer "+s.RawKey)
	sess := newSession(t, req)

	policies := []*sentinelv1.Policy{
		{
			Id:      "disabled",
			Enabled: false,
			Config: &sentinelv1.Policy_Keyauth{
				Keyauth: &sentinelv1.KeyAuth{KeySpaceIds: []string{s.KeySpaceID}},
			},
		},
	}

	result, err := h.engine.Evaluate(ctx, sess, req, policies)
	require.NoError(t, err)
	require.Nil(t, result.Principal)
}

func TestEvaluate_MatchFiltering(t *testing.T) {
	h := newTestHarness(t)
	ctx := context.Background()
	s := h.seed(ctx)

	// Request to /health doesn't match /api prefix
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	req.Header.Set("Authorization", "Bearer "+s.RawKey)
	sess := newSession(t, req)

	policies := []*sentinelv1.Policy{
		{
			Id:      "api-auth",
			Enabled: true,
			Match: []*sentinelv1.MatchExpr{
				{Expr: &sentinelv1.MatchExpr_Path{Path: &sentinelv1.PathMatch{
					Path: &sentinelv1.StringMatch{Match: &sentinelv1.StringMatch_Prefix{Prefix: "/api"}},
				}}},
			},
			Config: &sentinelv1.Policy_Keyauth{
				Keyauth: &sentinelv1.KeyAuth{KeySpaceIds: []string{s.KeySpaceID}},
			},
		},
	}

	result, err := h.engine.Evaluate(ctx, sess, req, policies)
	require.NoError(t, err)
	require.Nil(t, result.Principal)
}
