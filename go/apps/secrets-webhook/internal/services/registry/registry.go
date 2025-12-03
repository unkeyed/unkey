package registry

import (
	"context"
	"fmt"

	"github.com/google/go-containerregistry/pkg/authn/k8schain"
	"github.com/google/go-containerregistry/pkg/name"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"

	"github.com/unkeyed/unkey/go/pkg/otel/logging"
)

// ImageConfig contains the entrypoint and command from a container image.
type ImageConfig struct {
	Entrypoint []string
	Cmd        []string
}

// Registry fetches image configuration from container registries.
type Registry struct {
	logger    logging.Logger
	clientset kubernetes.Interface
}

// New creates a new Registry.
func New(logger logging.Logger, clientset kubernetes.Interface) *Registry {
	return &Registry{
		logger:    logger,
		clientset: clientset,
	}
}

// GetImageConfig fetches the entrypoint and command from a container image.
// It uses K8s authentication chain to handle image pull secrets.
func (r *Registry) GetImageConfig(
	ctx context.Context,
	namespace string,
	container *corev1.Container,
	podSpec *corev1.PodSpec,
) (*ImageConfig, error) {
	// Build K8s auth chain options
	chainOpts := k8schain.Options{
		Namespace:          namespace,
		ServiceAccountName: podSpec.ServiceAccountName,
	}
	for _, secret := range podSpec.ImagePullSecrets {
		chainOpts.ImagePullSecrets = append(chainOpts.ImagePullSecrets, secret.Name)
	}

	authChain, err := k8schain.New(ctx, r.clientset, chainOpts)
	if err != nil {
		return nil, fmt.Errorf("failed to create k8s auth chain: %w", err)
	}

	// Parse image reference
	ref, err := name.ParseReference(container.Image)
	if err != nil {
		return nil, fmt.Errorf("failed to parse image reference %q: %w", container.Image, err)
	}

	// Fetch image descriptor
	options := []remote.Option{
		remote.WithAuthFromKeychain(authChain),
		remote.WithContext(ctx),
	}

	descriptor, err := remote.Get(ref, options...)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch image descriptor for %q: %w", container.Image, err)
	}

	// Handle multi-arch images (index) vs single images
	var image v1.Image
	if descriptor.MediaType.IsIndex() {
		index, indexErr := descriptor.ImageIndex()
		if indexErr != nil {
			return nil, fmt.Errorf("failed to get image index: %w", indexErr)
		}

		manifest, manifestErr := index.IndexManifest()
		if manifestErr != nil {
			return nil, fmt.Errorf("failed to get index manifest: %w", manifestErr)
		}

		if len(manifest.Manifests) == 0 {
			return nil, fmt.Errorf("no manifests found in image index for %q", container.Image)
		}

		// Use first available image (usually linux/amd64)
		image, err = index.Image(manifest.Manifests[0].Digest)
		if err != nil {
			return nil, fmt.Errorf("failed to get image from manifest: %w", err)
		}
	} else {
		image, err = descriptor.Image()
		if err != nil {
			return nil, fmt.Errorf("failed to convert descriptor to image: %w", err)
		}
	}

	// Get image config
	configFile, err := image.ConfigFile()
	if err != nil {
		return nil, fmt.Errorf("failed to get image config: %w", err)
	}

	r.logger.Debug("fetched image config",
		"image", container.Image,
		"entrypoint", configFile.Config.Entrypoint,
		"cmd", configFile.Config.Cmd,
	)

	return &ImageConfig{
		Entrypoint: configFile.Config.Entrypoint,
		Cmd:        configFile.Config.Cmd,
	}, nil
}
