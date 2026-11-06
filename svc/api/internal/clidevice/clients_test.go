package clidevice

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/uid"
)

type recordingDocuments struct {
	documents fakeClientMetadata
	fetched   []string
}

func (r *recordingDocuments) Fetch(ctx context.Context, clientID string) (ClientMetadata, error) {
	r.fetched = append(r.fetched, clientID)
	return r.documents.Fetch(ctx, clientID)
}

// TestDirectoryDispatchesOnTheClientIDShape guarantees that a registered
// client id never triggers a network fetch, and that a URL client id is never
// checked against the registrar, so neither source can vouch for the other's
// clients.
func TestDirectoryDispatchesOnTheClientIDShape(t *testing.T) {
	registrar := newTestRegistrar(t)
	registered, err := registrar.Register(registrationFor("Cursor", "http://localhost:8787/callback"), registeredAt)
	require.NoError(t, err)
	_, document := newOAuthClient()
	documents := &recordingDocuments{documents: fakeClientMetadata{document.ClientID: document}, fetched: nil}
	directory := Directory{Documents: documents, Registrar: registrar}

	found, err := directory.Lookup(t.Context(), registered.ID)
	require.NoError(t, err)
	require.Equal(t, registered, found)
	require.Empty(t, documents.fetched)

	found, err = directory.Lookup(t.Context(), document.ClientID)
	require.NoError(t, err)
	require.Equal(t, OAuthClient{ID: document.ClientID, Name: document.ClientName, RedirectURIs: document.RedirectURIs}, found)
	require.Equal(t, []string{document.ClientID}, documents.fetched)

	for _, clientID := range []string{
		"http://client.example.com/oauth/client.json",
		uid.New(uid.TestPrefix),
		registeredClientPrefix + uid.New(uid.TestPrefix),
		"",
	} {
		_, err := directory.Lookup(t.Context(), clientID)
		requireOAuthError(t, err, OAuthInvalidRequest)
	}
	require.Len(t, documents.fetched, 1)
}

func TestClientHostNamesWhoVouchesForTheClient(t *testing.T) {
	require.Equal(t, "claude.ai", clientHost("https://claude.ai/oauth/claude-code-client-metadata", "http://localhost:53123/callback"))
	require.Equal(t, "localhost:53123", clientHost(registeredClientPrefix+"payload.mac", "http://localhost:53123/callback"))
	require.Empty(t, clientHost(registeredClientPrefix+"payload.mac", ""))
}
