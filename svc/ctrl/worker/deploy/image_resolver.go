package deploy

import (
	"context"

	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/name"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/unkeyed/unkey/pkg/fault"
)

// ImageResolver resolves a mutable OCI image reference to an immutable digest.
type ImageResolver interface {
	Resolve(ctx context.Context, imageReference string) (string, error)
}

type ImageResolverFunc func(ctx context.Context, imageReference string) (string, error)

func (f ImageResolverFunc) Resolve(ctx context.Context, imageReference string) (string, error) {
	return f(ctx, imageReference)
}

type ociImageResolver struct {
	internalRepository string
	internalAuth       authn.Authenticator
	options            []remote.Option
}

// NewImageResolver resolves public images anonymously and reuses the existing
// build-registry credentials for historical Unkey-built images.
func NewImageResolver(config RegistryConfig) (ImageResolver, error) {
	resolver := &ociImageResolver{
		internalRepository: "",
		internalAuth:       authn.Anonymous,
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
	if reference.Context().Name() == r.internalRepository {
		authenticator = r.internalAuth
	}
	options := append([]remote.Option{}, r.options...)
	options = append(options, remote.WithContext(ctx), remote.WithAuth(authenticator))
	descriptor, err := remote.Get(reference, options...)
	if err != nil {
		return "", fault.Wrap(err, fault.Internal("unable to resolve OCI image reference"))
	}

	return reference.Context().Digest(descriptor.Digest.String()).Name(), nil
}
