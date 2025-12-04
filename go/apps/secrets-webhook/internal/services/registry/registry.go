package registry

import (
	"context"
	"fmt"
	"runtime"

	"github.com/google/go-containerregistry/pkg/authn/k8schain"
	"github.com/google/go-containerregistry/pkg/name"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"

	"github.com/unkeyed/unkey/go/pkg/otel/logging"
)

const defaultOS = "linux"

type ImageConfig struct {
	Entrypoint []string
	Cmd        []string
}

type Registry struct {
	logger    logging.Logger
	clientset kubernetes.Interface
}

func New(logger logging.Logger, clientset kubernetes.Interface) *Registry {
	return &Registry{
		logger:    logger,
		clientset: clientset,
	}
}

func (r *Registry) GetImageConfig(
	ctx context.Context,
	namespace string,
	container *corev1.Container,
	podSpec *corev1.PodSpec,
) (*ImageConfig, error) {
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

	ref, err := name.ParseReference(container.Image)
	if err != nil {
		return nil, fmt.Errorf("failed to parse image reference %q: %w", container.Image, err)
	}

	options := []remote.Option{
		remote.WithAuthFromKeychain(authChain),
		remote.WithContext(ctx),
	}

	descriptor, err := remote.Get(ref, options...)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch image descriptor for %q: %w", container.Image, err)
	}

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

		digest, found := r.findPlatformManifest(manifest.Manifests)
		if !found {
			r.logger.Warn("no matching platform found in image index, using first manifest",
				"image", container.Image,
				"wanted_os", targetOS(),
				"wanted_arch", targetArch(),
			)
			digest = manifest.Manifests[0].Digest
		}

		image, err = index.Image(digest)
		if err != nil {
			return nil, fmt.Errorf("failed to get image from manifest: %w", err)
		}
	} else {
		image, err = descriptor.Image()
		if err != nil {
			return nil, fmt.Errorf("failed to convert descriptor to image: %w", err)
		}
	}

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

func (r *Registry) findPlatformManifest(manifests []v1.Descriptor) (v1.Hash, bool) {
	wantOS := targetOS()
	wantArch := targetArch()

	for _, m := range manifests {
		if m.Platform == nil {
			continue
		}
		if m.Platform.OS == wantOS && m.Platform.Architecture == wantArch {
			return m.Digest, true
		}
	}
	return v1.Hash{}, false
}

func targetOS() string {
	if runtime.GOOS == "darwin" {
		return defaultOS
	}
	return runtime.GOOS
}

// targetArch returns the architecture to look for in multi-arch image manifests.
// We assume the webhook runs on the same architecture as the workloads it mutates.
// If this assumption changes, adjust this function accordingly.
func targetArch() string {
	return runtime.GOARCH
}
