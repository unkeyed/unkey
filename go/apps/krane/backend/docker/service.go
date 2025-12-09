package docker

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"

	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/api/types/registry"
	"github.com/docker/docker/client"
	"github.com/unkeyed/unkey/go/gen/proto/krane/v1/kranev1connect"
	"github.com/unkeyed/unkey/go/pkg/otel/logging"
	"github.com/unkeyed/unkey/go/pkg/vault"
)

// docker implements kranev1connect.DeploymentServiceHandler using Docker Engine API.
//
// This backend is designed for single-node deployments where container images
// are built locally using the Docker daemon and deployed on the same host.
// It does not support multi-node clusters or remote image registries.
type docker struct {
	logger       logging.Logger
	client       *client.Client
	vault        *vault.Service
	registryAuth string // base64 encoded auth for pulls
	region       string

	kranev1connect.UnimplementedDeploymentServiceHandler
	kranev1connect.UnimplementedGatewayServiceHandler
}

var _ kranev1connect.DeploymentServiceHandler = (*docker)(nil)
var _ kranev1connect.GatewayServiceHandler = (*docker)(nil)

// Config holds configuration for the Docker backend
type Config struct {
	SocketPath       string
	RegistryURL      string
	RegistryUsername string
	RegistryPassword string
	Region           string
	Vault            *vault.Service
}

// New creates a Docker backend instance and validates daemon connectivity.
//
// The socketPath parameter specifies the Docker daemon socket location.
// Common values are "/var/run/docker.sock" on Linux and macOS.
//
// If registry credentials are provided, authenticates with the registry on startup.
//
// Returns an error if the Docker daemon is unreachable or the socket
// path is invalid. The socket must be accessible with appropriate permissions.
func New(logger logging.Logger, cfg Config) (*docker, error) {
	// Create Docker client with configurable socket path
	// socketPath must not include protocol (e.g., "var/run/docker.sock")
	dockerClient, err := client.NewClientWithOpts(
		client.FromEnv,
		client.WithAPIVersionNegotiation(),
		client.WithHost(fmt.Sprintf("unix:///%s", cfg.SocketPath)),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create Docker client: %w", err)
	}

	// Test connection to Docker daemon
	ctx := context.Background()
	_, err = dockerClient.Ping(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to ping Docker daemon at %s (ensure Docker socket is accessible): %w", cfg.SocketPath, err)
	}

	logger.Info("Docker client initialized successfully", "socket", cfg.SocketPath)

	d := &docker{
		registryAuth:                          "",
		region:                                cfg.Region,
		vault:                                 cfg.Vault,
		UnimplementedDeploymentServiceHandler: kranev1connect.UnimplementedDeploymentServiceHandler{},
		UnimplementedGatewayServiceHandler:    kranev1connect.UnimplementedGatewayServiceHandler{},
		logger:                                logger,
		client:                                dockerClient,
	}

	// Encode registry credentials if provided.
	// These will be passed to ImagePull for authentication.
	if cfg.RegistryURL != "" && cfg.RegistryUsername != "" && cfg.RegistryPassword != "" {
		//nolint: exhaustruct
		authConfig := registry.AuthConfig{
			Username:      cfg.RegistryUsername,
			Password:      cfg.RegistryPassword,
			ServerAddress: cfg.RegistryURL,
		}

		authJSON, err := json.Marshal(authConfig)
		if err != nil {
			return nil, fmt.Errorf("failed to encode registry auth: %w", err)
		}
		d.registryAuth = base64.URLEncoding.EncodeToString(authJSON)

		logger.Info("Registry credentials configured",
			"registry", cfg.RegistryURL,
			"username", cfg.RegistryUsername)
	} else {
		logger.Info("No registry credentials configured - will only use local images",
			"registry_url", cfg.RegistryURL,
			"registry_username", cfg.RegistryUsername,
			"registry_password_set", cfg.RegistryPassword != "")
	}

	return d, nil
}

// ensureImageExists checks if an image exists locally, and pulls it if not.
// This implements a "pull if not present" policy similar to Docker's default behavior.
func (d *docker) ensureImageExists(ctx context.Context, imageName string) error {
	// Check if image exists locally
	_, err := d.client.ImageInspect(ctx, imageName)
	if err == nil {
		// Image exists locally
		d.logger.Info("using existing local image", "image", imageName)
		return nil
	}

	// Image doesn't exist, pull it
	hasAuth := d.registryAuth != ""
	d.logger.Info("image not found locally, pulling",
		"image", imageName,
		"using_registry_auth", hasAuth)

	// Pass registry auth if we have it
	// nolint: exhaustruct
	pullOpts := image.PullOptions{
		RegistryAuth: d.registryAuth,
	}

	pullResp, err := d.client.ImagePull(ctx, imageName, pullOpts)
	if err != nil {
		return fmt.Errorf("failed to pull image: %w", err)
	}
	defer pullResp.Close()

	// Wait for pull to complete - discard output but check for errors
	_, err = io.Copy(io.Discard, pullResp)
	if err != nil {
		return fmt.Errorf("failed to complete image pull: %w", err)
	}

	d.logger.Info("image pulled successfully", "image", imageName)
	return nil
}
