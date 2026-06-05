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
	calls   int
}

func (s *stubKeyService) Get(_ context.Context, _ *zen.Session, _ string) (*keys.KeyVerifier, func(), error) {
	return nil, func() {}, errors.New("not implemented")
}

func (s *stubKeyService) GetRootKey(_ context.Context, _ *zen.Session) (*keys.KeyVerifier, func(), error) {
	s.calls++
	return s.rootKey, func() {}, s.err
}

func (s *stubKeyService) GetMigrated(_ context.Context, _ *zen.Session, _, _ string) (*keys.KeyVerifier, func(), error) {
	return nil, func() {}, errors.New("not implemented")
}

func (s *stubKeyService) CreateKey(_ context.Context, _ keys.CreateKeyRequest) (keys.CreateKeyResponse, error) {
	return keys.CreateKeyResponse{}, errors.New("not implemented")
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
				ID:        "key_123",
				KeyAuthID: "ks_123",
				Name:      sql.NullString{String: "Production root key", Valid: true},
			},
			Permissions:           []string{"api.*.read_key"},
			AuthorizedWorkspaceID: "ws_123",
		},
	}
	resolver := NewResolver(keyService)

	p, err := resolver.Resolve(context.Background(), newSessionWithAuth(t, "Bearer unkey_root_key"))

	require.NoError(t, err)
	require.Equal(t, 1, keyService.calls)
	require.Equal(t, authprincipal.Version, p.Version)
	require.Equal(t, authprincipal.TypeAPIKey, p.Type)
	require.Equal(t, authprincipal.SubjectTypeRootKey, p.Subject.Type)
	require.Equal(t, "key_123", p.Subject.ID)
	require.Equal(t, "Production root key", p.Subject.Name)
	require.Equal(t, "ws_123", p.WorkspaceID)
	require.Equal(t, []string{"api.*.read_key"}, p.Permissions)
	source, ok := p.Source.(authprincipal.KeySource)
	require.True(t, ok)
	require.Equal(t, "key_123", source.KeyID)
	require.Equal(t, "ks_123", source.KeySpaceID)
	require.Equal(t, []string{"api.*.read_key"}, source.Permissions)
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
	require.Equal(t, 1, keyService.calls)
}

// TestResolver_YieldsWhenAuthorizationMissing verifies the resolver does not
// claim requests that have no Authorization header. This lets the auth chain
// surface a generic missing-credentials error rather than leaking the
// root-key-specific message for portal or unauthenticated callers.
func TestResolver_YieldsWhenAuthorizationMissing(t *testing.T) {
	t.Parallel()

	keyService := &stubKeyService{}
	resolver := NewResolver(keyService)

	p, err := resolver.Resolve(context.Background(), newSessionWithAuth(t, ""))

	require.NoError(t, err)
	require.Nil(t, p)
	require.Equal(t, 0, keyService.calls)
}
