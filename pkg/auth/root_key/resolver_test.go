package rootkey

import (
	"context"
	"database/sql"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/internal/services/keys"
	keysdb "github.com/unkeyed/unkey/internal/services/keys/db"
	authprincipal "github.com/unkeyed/unkey/pkg/auth/principal"
	"github.com/unkeyed/unkey/pkg/zen"
)

// newSessionWithAuth builds a session with the given Authorization header set so
// resolver tests exercise the path where credentials are actually present.
func newSessionWithAuth(t *testing.T, auth string) *zen.Session {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	if auth != "" {
		req.Header.Set("Authorization", auth)
	}
	sess := &zen.Session{}
	require.NoError(t, sess.Init(httptest.NewRecorder(), req, 0))
	return sess
}

type stubKeyService struct {
	rootKey *keys.KeyVerifier
	err     error
}

func (s *stubKeyService) Get(_ context.Context, _ *zen.Session, _ string) (*keys.KeyVerifier, error) {
	return nil, errors.New("not implemented")
}

func (s *stubKeyService) GetRootKey(_ context.Context, _ *zen.Session) (*keys.KeyVerifier, error) {
	return s.rootKey, s.err
}

func (s *stubKeyService) GetMigrated(_ context.Context, _ *zen.Session, _, _ string) (*keys.KeyVerifier, error) {
	return nil, errors.New("not implemented")
}

func (s *stubKeyService) CreateKey(_ context.Context, _ keys.CreateKeyRequest) (keys.CreateKeyResponse, error) {
	return keys.CreateKeyResponse{}, errors.New("not implemented")
}

func (s *stubKeyService) CreateKeyV1(_ context.Context, _ keys.CreateKeyV1Request) (keys.CreateKeyV1Response, error) {
	return keys.CreateKeyV1Response{}, errors.New("not implemented")
}

// TestResolver_ResolveRootKeyPrincipal verifies a verified root key normalizes
// into the shared principal shape used by the API. The stubbed key service
// returns the post-verification object so this test focuses on the resolver's
// security contract: subject, workspace, source, and permissions all come from
// the verified root key.
func TestResolver_ResolveRootKeyPrincipal(t *testing.T) {
	t.Parallel()

	keyService := &stubKeyService{
		rootKey: &keys.KeyVerifier{
			Key: keysdb.FindKeyForVerificationRow{
				ID:             "key_123",
				KeyAuthID:      "ks_123",
				WorkspaceID:    "ws_owner",
				ForWorkspaceID: sql.NullString{String: "ws_authorized", Valid: true},
				Name:           sql.NullString{String: "Production root key", Valid: true},
			},
			Roles:                 []string{"admin"},
			Permissions:           []string{"api.*.read_key"},
			Status:                keys.StatusValid,
			AuthorizedWorkspaceID: "ws_authorized",
		},
	}
	resolver := NewResolver(keyService)

	p, err := resolver.Resolve(context.Background(), newSessionWithAuth(t, "Bearer unkey_root_key"))

	require.NoError(t, err)
	require.Equal(t, &authprincipal.Principal{
		Version: authprincipal.Version,
		Subject: authprincipal.Subject{
			ID:   "key_123",
			Name: "Production root key",
			Type: authprincipal.SubjectTypeRootKey,
		},
		Type: authprincipal.TypeAPIKey,
		Source: authprincipal.KeySource{
			KeyID:       "key_123",
			KeySpaceID:  "ks_123",
			WorkspaceID: "ws_owner",
			Permissions: []string{"api.*.read_key"},
		},
		AuthorizedWorkspaceID: "ws_authorized",
		Permissions:           []string{"api.*.read_key"},
	}, p)
}

// TestResolver_UsesFallbackRootKeyName guarantees audit data has a stable
// subject name when the verified root key has no configured name.
func TestResolver_UsesFallbackRootKeyName(t *testing.T) {
	t.Parallel()

	keyService := &stubKeyService{
		rootKey: &keys.KeyVerifier{
			Key: keysdb.FindKeyForVerificationRow{
				ID:   "key_unnamed",
				Name: sql.NullString{},
			},
			Status: keys.StatusValid,
		},
	}
	resolver := NewResolver(keyService)

	p, err := resolver.Resolve(context.Background(), newSessionWithAuth(t, "Bearer unkey_root_key"))

	require.NoError(t, err)
	require.NotNil(t, p)
	require.Equal(t, "root key", p.Subject.Name)
}

// TestResolver_PropagatesRootKeyError verifies root-key verification failures
// fail closed instead of yielding to later resolvers. A root-key-shaped bearer
// token that fails verification must not be reinterpreted as another credential
// type.
func TestResolver_PropagatesRootKeyError(t *testing.T) {
	t.Parallel()

	wantErr := errors.New("invalid root key")
	keyService := &stubKeyService{err: wantErr}
	resolver := NewResolver(keyService)

	p, err := resolver.Resolve(context.Background(), newSessionWithAuth(t, "Bearer bad_root_key"))

	require.ErrorIs(t, err, wantErr)
	require.Nil(t, p)
}

// TestResolver_YieldsWhenAuthorizationMissing verifies the resolver does not
// claim requests that have no Authorization header. This lets the auth chain
// surface a generic missing-credentials error rather than leaking the
// root-key-specific message for portal or unauthenticated callers.
func TestResolver_YieldsWhenAuthorizationMissing(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		session func(t *testing.T) *zen.Session
	}{
		{
			name: "nil session",
			session: func(_ *testing.T) *zen.Session {
				return nil
			},
		},
		{
			name: "nil request",
			session: func(_ *testing.T) *zen.Session {
				return &zen.Session{}
			},
		},
		{
			name: "missing header",
			session: func(t *testing.T) *zen.Session {
				return newSessionWithAuth(t, "")
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			keyService := &stubKeyService{err: errors.New("unexpected GetRootKey call")}
			resolver := NewResolver(keyService)

			p, err := resolver.Resolve(context.Background(), test.session(t))

			require.NoError(t, err)
			require.Nil(t, p)
		})
	}
}
