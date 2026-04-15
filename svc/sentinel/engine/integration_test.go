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
	keysdb "github.com/unkeyed/unkey/internal/services/keys/db"
	"github.com/unkeyed/unkey/internal/services/ratelimit"
	"github.com/unkeyed/unkey/internal/services/usagelimiter"
	"github.com/unkeyed/unkey/pkg/cache"
	"github.com/unkeyed/unkey/pkg/clock"
	"github.com/unkeyed/unkey/pkg/codes"
	"github.com/unkeyed/unkey/pkg/counter"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/dockertest"
	"github.com/unkeyed/unkey/pkg/fault"
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

	mysqlCfg := dockertest.MySQL(t)
	redisURL := dockertest.Redis(t)

	clk := clock.New()

	database, err := db.New(db.Config{
		PrimaryDSN:  mysqlCfg.DSN,
		ReadOnlyDSN: "",
	})
	require.NoError(t, err)
	t.Cleanup(func() { _ = database.Close() })

	redisCounter, err := counter.NewRedis(counter.RedisConfig{
		RedisURL: redisURL,
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
		FindKeyCredits: func(ctx context.Context, keyID string) (int32, bool, error) {
			limit, err := db.WithRetryContext(ctx, func() (sql.NullInt32, error) {
				return db.Query.FindKeyCredits(ctx, database.RO(), keyID)
			})
			if err != nil {
				return 0, false, err
			}
			return limit.Int32, limit.Valid, nil
		},
		DecrementKeyCredits: func(ctx context.Context, keyID string, cost int32) error {
			return db.Query.UpdateKeyCreditsDecrement(ctx, database.RW(), db.UpdateKeyCreditsDecrementParams{
				ID:      keyID,
				Credits: sql.NullInt32{Int32: cost, Valid: true},
			})
		},
		Counter:       redisCounter,
		TTL:           60 * time.Second,
		ReplayWorkers: 2,
	})
	require.NoError(t, err)
	t.Cleanup(func() { _ = usageLimiter.Close() })

	keyCache, err := cache.New[string, keysdb.CachedKeyData](cache.Config[string, keysdb.CachedKeyData]{
		Fresh:    10 * time.Second,
		Stale:    10 * time.Minute,
		MaxSize:  1000,
		Resource: "test_key_cache",
		Clock:    clk,
	})
	require.NoError(t, err)

	keyService, err := keys.New(keys.Config{
		DB:               db.ToMySQL(database),
		RateLimiter:      rateLimiter,
		RBAC:             rbac.New(),
		KeyVerifications: nil,
		Region:           "test",
		UsageLimiter:     usageLimiter,
		KeyCache:         keyCache,
		QuotaCache:       nil,
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
	require.Equal(t, engine.PrincipalVersion, result.Principal.Version)
	require.Equal(t, "v1", result.Principal.Version)
	require.Equal(t, s.KeyID, result.Principal.Subject)
	require.Equal(t, engine.PrincipalTypeAPIKey, result.Principal.Type)
	require.Nil(t, result.Principal.Identity)

	key := result.Principal.Source.Key
	require.NotNil(t, key)
	require.Equal(t, s.KeyID, key.KeyID)
	require.Equal(t, s.KeySpaceID, key.KeySpaceID)
	require.NotNil(t, key.Meta)
	require.Empty(t, key.Roles)
	require.Empty(t, key.Permissions)
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
	identity := result.Principal.Identity
	require.NotNil(t, identity)
	require.NotEmpty(t, identity.ExternalID)
	require.Equal(t, identity.ExternalID, result.Principal.Subject)
	require.NotNil(t, identity.Meta)
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

// --- Firewall integration tests ---

func TestFirewall_DenyByPath(t *testing.T) {
	h := newTestHarness(t)
	ctx := context.Background()

	req := httptest.NewRequest(http.MethodGet, "/xxx", nil)
	sess := newSession(t, req)

	policies := []*sentinelv1.Policy{
		{
			Id:      "block-xxx",
			Name:    "Block /xxx",
			Enabled: true,
			Match: []*sentinelv1.MatchExpr{
				{Expr: &sentinelv1.MatchExpr_Path{Path: &sentinelv1.PathMatch{
					Path: &sentinelv1.StringMatch{Match: &sentinelv1.StringMatch_Prefix{Prefix: "/xxx"}},
				}}},
			},
			Config: &sentinelv1.Policy_Firewall{
				Firewall: &sentinelv1.Firewall{Action: sentinelv1.Action_ACTION_DENY},
			},
		},
	}

	_, err := h.engine.Evaluate(ctx, sess, req, policies)
	require.Error(t, err)
	urn, ok := fault.GetCode(err)
	require.True(t, ok)
	require.Equal(t, codes.Sentinel.Firewall.Denied.URN(), urn)
}

func TestFirewall_DenyByPath_NonMatchPasses(t *testing.T) {
	h := newTestHarness(t)
	ctx := context.Background()

	req := httptest.NewRequest(http.MethodGet, "/healthy", nil)
	sess := newSession(t, req)

	policies := []*sentinelv1.Policy{
		{
			Id:      "block-xxx",
			Enabled: true,
			Match: []*sentinelv1.MatchExpr{
				{Expr: &sentinelv1.MatchExpr_Path{Path: &sentinelv1.PathMatch{
					Path: &sentinelv1.StringMatch{Match: &sentinelv1.StringMatch_Prefix{Prefix: "/xxx"}},
				}}},
			},
			Config: &sentinelv1.Policy_Firewall{
				Firewall: &sentinelv1.Firewall{Action: sentinelv1.Action_ACTION_DENY},
			},
		},
	}

	_, err := h.engine.Evaluate(ctx, sess, req, policies)
	require.NoError(t, err)
}

func TestFirewall_DenyRunsBeforeKeyAuth(t *testing.T) {
	h := newTestHarness(t)
	ctx := context.Background()
	s := h.seed(ctx)

	// Invalid bearer token — if keyauth ran, it would return its own error.
	// Firewall DENY is placed first and should short-circuit before keyauth.
	req := httptest.NewRequest(http.MethodGet, "/xxx", nil)
	req.Header.Set("Authorization", "Bearer not-a-real-key")
	sess := newSession(t, req)

	policies := []*sentinelv1.Policy{
		{
			Id:      "block-xxx",
			Enabled: true,
			Match: []*sentinelv1.MatchExpr{
				{Expr: &sentinelv1.MatchExpr_Path{Path: &sentinelv1.PathMatch{
					Path: &sentinelv1.StringMatch{Match: &sentinelv1.StringMatch_Prefix{Prefix: "/xxx"}},
				}}},
			},
			Config: &sentinelv1.Policy_Firewall{
				Firewall: &sentinelv1.Firewall{Action: sentinelv1.Action_ACTION_DENY},
			},
		},
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
	urn, ok := fault.GetCode(err)
	require.True(t, ok)
	require.Equal(t, codes.Sentinel.Firewall.Denied.URN(), urn)
}
