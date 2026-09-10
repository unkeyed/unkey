package deploy

import (
	"context"
	"net/http"

	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/name"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/unkeyed/unkey/pkg/fault"
)

type ociImageResolver struct {
	internalRepository string
	internalAuth       authn.Authenticator
	publicTransport    http.RoundTripper
	internalTransport  http.RoundTripper
	options            []remote.Option
}

// NewImageResolver resolves public images anonymously and reuses the existing
// build-registry credentials for historical Unkey-built images.
func NewImageResolver(config RegistryConfig) (ImageResolver, error) {
	resolver := &ociImageResolver{
		internalRepository: "",
		internalAuth:       authn.Anonymous,
		publicTransport:    newRegistryTransport(""),
		internalTransport:  nil,
		options:            nil,
	}
	if config.Repository == "" {
		return resolver, nil
	}

	repository, err := name.NewRepository(config.Repository, name.WeakValidation)
	if err != nil {
		return nil, fault.Wrap(err, fault.Internal("invalid internal registry repository"))
	}
	resolver.internalRepository = repository.Name()
	resolver.internalTransport = newRegistryTransport(repository.RegistryStr())
	if config.Username != "" || config.Password != "" {
		resolver.internalAuth = authn.FromConfig(authn.AuthConfig{
			Username:      config.Username,
			Password:      config.Password,
			Auth:          "",
			IdentityToken: "",
			RegistryToken: "",
		})
	}
	return resolver, nil
}

func (r *ociImageResolver) Resolve(ctx context.Context, imageReference string) (string, error) {
	reference, err := name.ParseReference(imageReference, name.WeakValidation)
	if err != nil {
		return "", fault.Wrap(err, fault.Internal("invalid OCI image reference"))
	}

	authenticator := authn.Anonymous
	transport := r.publicTransport
	if reference.Context().Name() == r.internalRepository {
		authenticator = r.internalAuth
		transport = r.internalTransport
	}
	options := make([]remote.Option, 0, len(r.options)+3)
	if transport != nil {
		options = append(options, remote.WithTransport(transport))
	}
	options = append(options, r.options...)
	options = append(options, remote.WithContext(ctx), remote.WithAuth(authenticator))
	descriptor, err := remote.Get(reference, options...)
	if err != nil {
		return "", fault.Wrap(err, fault.Internal("unable to resolve OCI image reference"))
	}

	return reference.Context().Digest(descriptor.Digest.String()).Name(), nil
}
