package deploy

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/name"
	"github.com/google/go-containerregistry/pkg/registry"
	"github.com/google/go-containerregistry/pkg/v1/empty"
	"github.com/google/go-containerregistry/pkg/v1/mutate"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/stretchr/testify/require"
)

func TestPublicImageResolverPinsImageIndex(t *testing.T) {
	server := httptest.NewTLSServer(registry.New())
	t.Cleanup(server.Close)

	transport := server.Client().Transport
	tag, err := name.NewTag(server.Listener.Addr().String() + "/acme/api:stable")
	require.NoError(t, err)

	index := mutate.AppendManifests(empty.Index, mutate.IndexAddendum{Add: empty.Image})
	require.NoError(t, remote.WriteIndex(tag, index, remote.WithTransport(transport)))
	wantDigest, err := index.Digest()
	require.NoError(t, err)

	resolver := &ociImageResolver{
		internalRepository: "",
		internalAuth:       authn.Anonymous,
		options:            []remote.Option{remote.WithTransport(transport)},
	}
	resolved, err := resolver.Resolve(context.Background(), tag.Name())
	require.NoError(t, err)
	require.Equal(t, tag.Context().Digest(wantDigest.String()).Name(), resolved)
}

func TestImageResolverScopesInternalRegistryCredentialsToConfiguredRepository(t *testing.T) {
	const (
		username = "unkey"
		password = "secret"
	)
	registryHandler := registry.New()
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestUsername, requestPassword, ok := r.BasicAuth()
		if !ok || requestUsername != username || requestPassword != password {
			w.Header().Set("WWW-Authenticate", `Basic realm="test registry"`)
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		registryHandler.ServeHTTP(w, r)
	}))
	t.Cleanup(server.Close)

	host := server.Listener.Addr().String()
	authenticator := authn.FromConfig(authn.AuthConfig{
		Username:      username,
		Password:      password,
		Auth:          "",
		IdentityToken: "",
		RegistryToken: "",
	})
	remoteOptions := []remote.Option{
		remote.WithTransport(server.Client().Transport),
		remote.WithAuth(authenticator),
	}
	internalTag, err := name.NewTag(host + "/unkey/builds:v1")
	require.NoError(t, err)
	require.NoError(t, remote.Write(internalTag, empty.Image, remoteOptions...))
	otherTag, err := name.NewTag(host + "/customer/app:v1")
	require.NoError(t, err)
	require.NoError(t, remote.Write(otherTag, empty.Image, remoteOptions...))

	resolver, err := NewImageResolver(RegistryConfig{
		Repository: host + "/unkey/builds",
		Username:   username,
		Password:   password,
	})
	require.NoError(t, err)
	ociResolver, ok := resolver.(*ociImageResolver)
	require.True(t, ok)
	ociResolver.options = []remote.Option{remote.WithTransport(server.Client().Transport)}

	resolved, err := resolver.Resolve(context.Background(), internalTag.Name())
	require.NoError(t, err)
	wantDigest, err := empty.Image.Digest()
	require.NoError(t, err)
	require.Equal(t, internalTag.Context().Digest(wantDigest.String()).Name(), resolved)

	_, err = resolver.Resolve(context.Background(), otherTag.Name())
	require.Error(t, err, "credentials must not be sent to another repository on the same registry")
}
