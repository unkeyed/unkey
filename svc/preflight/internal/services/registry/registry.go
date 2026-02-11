package registry

import (
	"context"
	"fmt"
	"runtime"
	"strings"

	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/authn/k8schain"
	"github.com/google/go-containerregistry/pkg/name"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"

	"github.com/unkeyed/unkey/pkg/logger"
	"github.com/unkeyed/unkey/svc/preflight/internal/services/registry/credentials"
)

const defaultOS = "linux"

type ImageConfig struct {
	Entrypoint []string
	Cmd        []string
}

type Config struct {
	Clientset          kubernetes.Interface
	Credentials        *credentials.Manager
	InsecureRegistries []string
	RegistryAliases    []string
}

type Registry struct {
	clientset          kubernetes.Interface
	credentials        *credentials.Manager
	insecureRegistries map[string]bool
	registryAliases    map[string]string
}

func New(cfg Config) *Registry {
	insecureRegistries := make(map[string]bool)
	for _, reg := range cfg.InsecureRegistries {
		insecureRegistries[reg] = true
		logger.Info("configured insecure registry", "registry", reg)
	}

	registryAliases := make(map[string]string)
	for _, alias := range cfg.RegistryAliases {
		parts := strings.SplitN(alias, "=", 2)
		if len(parts) == 2 {
			from, to := parts[0], parts[1]
			registryAliases[from] = to
			logger.Info("configured registry alias", "from", from, "to", to)
		}
	}

	return &Registry{
		clientset:          cfg.Clientset,
		credentials:        cfg.Credentials,
		insecureRegistries: insecureRegistries,
		registryAliases:    registryAliases,
	}
}

func (r *Registry) GetImageConfig(
	ctx context.Context,
	namespace string,
	container *corev1.Container,
	podSpec *corev1.PodSpec,
	buildID string,
) (*ImageConfig, error) {
	// Parse the image reference
	ref, err := name.ParseReference(container.Image)
	if err != nil {
		return nil, fmt.Errorf("failed to parse image reference %q: %w", container.Image, err)
	}

	// Apply registry alias translation if configured (handles insecure flag internally)
	ref, err = r.translateReference(ref)
	if err != nil {
		return nil, fmt.Errorf("failed to translate registry alias for %q: %w", container.Image, err)
	}

	// Try to get credentials from our credentials manager first (for private registries like Depot)
	var options []remote.Option
	if r.credentials != nil && r.credentials.Matches(container.Image) {
		dockerConfig, credErr := r.credentials.GetCredentials(ctx, container.Image, buildID)
		if credErr != nil {
			return nil, fmt.Errorf("failed to get credentials for %q: %w", container.Image, credErr)
		}
		if dockerConfig != nil {
			// Use credentials from our manager
			for registry, auth := range dockerConfig.Auths {
				logger.Debug("using credentials from manager",
					"image", container.Image,
					"registry", registry,
				)
				options = append(options, remote.WithAuth(&authn.Basic{
					Username: auth.Username,
					Password: auth.Password,
				}))
				break // Only need one auth
			}
		}
	}

	// Fall back to k8schain for other registries
	if len(options) == 0 {
		//nolint:exhaustruct // k8schain has many optional fields
		chainOpts := k8schain.Options{
			Namespace:          namespace,
			ServiceAccountName: podSpec.ServiceAccountName,
		}
		for _, secret := range podSpec.ImagePullSecrets {
			chainOpts.ImagePullSecrets = append(chainOpts.ImagePullSecrets, secret.Name)
		}

		authChain, chainErr := k8schain.New(ctx, r.clientset, chainOpts)
		if chainErr != nil {
			return nil, fmt.Errorf("failed to create k8s auth chain: %w", chainErr)
		}
		options = append(options, remote.WithAuthFromKeychain(authChain))
	}

	options = append(options, remote.WithContext(ctx))

	descriptor, err := remote.Get(ref, options...)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch image descriptor for %q: %w", container.Image, err)
	}

	var img v1.Image
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
			logger.Warn("no matching platform found in image index, using first manifest",
				"image", container.Image,
				"wanted_os", targetOS(),
				"wanted_arch", targetArch(),
			)
			digest = manifest.Manifests[0].Digest
		}

		img, err = index.Image(digest)
		if err != nil {
			return nil, fmt.Errorf("failed to get image from manifest: %w", err)
		}
	} else {
		img, err = descriptor.Image()
		if err != nil {
			return nil, fmt.Errorf("failed to convert descriptor to image: %w", err)
		}
	}

	configFile, err := img.ConfigFile()
	if err != nil {
		return nil, fmt.Errorf("failed to get image config: %w", err)
	}

	logger.Debug("fetched image config",
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

	return v1.Hash{}, false //nolint:exhaustruct // zero value for not-found case
}

func targetOS() string {
	return defaultOS
}

// targetArch returns the architecture to look for in multi-arch image manifests.
// We assume the webhook runs on the same architecture as the workloads it mutates.
// If this assumption changes, adjust this function accordingly.
func targetArch() string {
	return runtime.GOARCH
}

// translateReference applies registry alias if configured, returning a new reference
// with the translated registry. For example, localhost:5000/demo_api:tag becomes
// registry.kube-system.svc.cluster.local/demo_api:tag.
// Also applies the Insecure option if the target registry is in the insecure list.
func (r *Registry) translateReference(ref name.Reference) (name.Reference, error) {
	currentRegistry := ref.Context().RegistryStr()
	targetRegistry := currentRegistry

	// Check if we have an alias for this registry
	if newRegistry, hasAlias := r.registryAliases[currentRegistry]; hasAlias {
		targetRegistry = newRegistry
		logger.Debug("translating registry alias", "from", currentRegistry, "to", targetRegistry)
	}

	// Determine if target registry should use HTTP (insecure)
	var repoOpts []name.Option
	if r.insecureRegistries[targetRegistry] {
		repoOpts = append(repoOpts, name.Insecure)
		logger.Debug("using insecure (HTTP) connection for registry", "registry", targetRegistry)
	}

	// If no alias and no insecure flag needed, return original
	if targetRegistry == currentRegistry && len(repoOpts) == 0 {
		return ref, nil
	}

	// Construct new repository with target registry
	newRepo, err := name.NewRepository(targetRegistry+"/"+ref.Context().RepositoryStr(), repoOpts...)
	if err != nil {
		return nil, fmt.Errorf("failed to create repository: %w", err)
	}

	// Preserve tag or digest from original reference
	switch v := ref.(type) {
	case name.Tag:
		return newRepo.Tag(v.TagStr()), nil
	case name.Digest:
		return newRepo.Digest(v.DigestStr()), nil
	default:
		return ref, nil
	}
}
