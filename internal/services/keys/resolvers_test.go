package keys

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
	keysdb "github.com/unkeyed/unkey/internal/services/keys/db"
	authprincipal "github.com/unkeyed/unkey/pkg/auth/principal"
	"github.com/unkeyed/unkey/pkg/zen"
)

type stubKeyService struct {
	rootKey *KeyVerifier
	err     error
	calls   int
}

func (s *stubKeyService) Get(_ context.Context, _ *zen.Session, _ string) (*KeyVerifier, func(), error) {
	return nil, func() {}, errors.New("not implemented")
}

func (s *stubKeyService) GetRootKey(_ context.Context, _ *zen.Session) (*KeyVerifier, func(), error) {
	s.calls++
	return s.rootKey, func() {}, s.err
}

func (s *stubKeyService) GetMigrated(_ context.Context, _ *zen.Session, _, _ string) (*KeyVerifier, func(), error) {
	return nil, func() {}, errors.New("not implemented")
}

func (s *stubKeyService) CreateKey(_ context.Context, _ CreateKeyRequest) (CreateKeyResponse, error) {
	return CreateKeyResponse{}, errors.New("not implemented")
}

// TestRootKeyResolver_ResolveRootKeyPrincipal verifies a verified root key
// normalizes into the shared principal shape used by the API. The stubbed key
// service returns the post-verification object so this test focuses on the
// resolver's security contract: subject, workspace, source, and permissions all
// come from the verified root key.
func TestRootKeyResolver_ResolveRootKeyPrincipal(t *testing.T) {
	t.Parallel()

	keyService := &stubKeyService{
		rootKey: &KeyVerifier{
			Key: keysdb.FindKeyForVerificationRow{
				ID:        "key_123",
				KeyAuthID: "ks_123",
				Name:      sql.NullString{String: "Production root key", Valid: true},
			},
			Permissions:           []string{"api.*.read_key"},
			AuthorizedWorkspaceID: "ws_123",
		},
	}
	resolver := NewRootKeyResolver(keyService)

	p, err := resolver.Resolve(context.Background(), &zen.Session{})

	require.NoError(t, err)
	require.Equal(t, 1, keyService.calls)
	require.Equal(t, authprincipal.Version, p.Version)
	require.Equal(t, authprincipal.TypeAPIKey, p.Type)
	require.Equal(t, authprincipal.SubjectTypeRootKey, p.Subject.Type)
	require.Equal(t, "key_123", p.Subject.ID)
	require.Equal(t, "Production root key", p.Subject.Name)
	require.Equal(t, "ws_123", p.WorkspaceID)
	require.Equal(t, []string{"api.*.read_key"}, p.Permissions)
	require.NotNil(t, p.Source.Key)
	require.Equal(t, "key_123", p.Source.Key.KeyID)
	require.Equal(t, "ks_123", p.Source.Key.KeySpaceID)
	require.Equal(t, []string{"api.*.read_key"}, p.Source.Key.Permissions)
	require.Nil(t, p.Source.JWT)
	require.Nil(t, p.Source.PortalSession)
}

// TestRootKeyResolver_PropagatesRootKeyError verifies root-key verification
// failures fail closed instead of yielding to later resolvers. A root-key-shaped
// bearer token that fails verification must not be reinterpreted as another
// credential type.
func TestRootKeyResolver_PropagatesRootKeyError(t *testing.T) {
	t.Parallel()

	wantErr := errors.New("invalid root key")
	keyService := &stubKeyService{err: wantErr}
	resolver := NewRootKeyResolver(keyService)

	p, err := resolver.Resolve(context.Background(), &zen.Session{})

	require.ErrorIs(t, err, wantErr)
	require.Nil(t, p)
	require.Equal(t, 1, keyService.calls)
}
