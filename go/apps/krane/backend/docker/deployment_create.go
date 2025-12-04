package docker

import (
	"context"
	"fmt"

	"connectrpc.com/connect"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/go-connections/nat"
	ctrlv1 "github.com/unkeyed/unkey/go/gen/proto/ctrl/v1"
	kranev1 "github.com/unkeyed/unkey/go/gen/proto/krane/v1"
	vaultv1 "github.com/unkeyed/unkey/go/gen/proto/vault/v1"
	"google.golang.org/protobuf/encoding/protojson"
)

// CreateDeployment creates containers for a deployment with the specified replica count.
//
// Creates multiple containers with shared labels, dynamic port mapping to port 8080,
// and resource limits. Returns DEPLOYMENT_STATUS_PENDING as containers may not be
// immediately ready.
func (d *docker) CreateDeployment(ctx context.Context, req *connect.Request[kranev1.CreateDeploymentRequest]) (*connect.Response[kranev1.CreateDeploymentResponse], error) {
	deployment := req.Msg.GetDeployment()
	d.logger.Info("creating deployment",
		"deployment_id", deployment.GetDeploymentId(),
		"image", deployment.GetImage(),
	)

	// Ensure image exists locally (pull if not present)
	if err := d.ensureImageExists(ctx, deployment.GetImage()); err != nil {
		return nil, connect.NewError(connect.CodeInternal,
			fmt.Errorf("failed to ensure image exists: %w", err))
	}

	// Configure port mapping
	exposedPorts := nat.PortSet{
		"8080/tcp": struct{}{},
	}

	portBindings := nat.PortMap{
		"8080/tcp": []nat.PortBinding{
			{
				HostIP:   "0.0.0.0",
				HostPort: "0", // Docker will assign a random available port
			},
		},
	}

	// Configure resource limits
	cpuNanos := int64(deployment.GetCpuMillicores()) * 1_000_000      // Convert millicores to nanoseconds
	memoryBytes := int64(deployment.GetMemorySizeMib()) * 1024 * 1024 //nolint:gosec // Intentional conversion

	// Build environment variables list
	env := []string{
		// Unkey-provided environment variables
		fmt.Sprintf("UNKEY_DEPLOYMENT_ID=%s", deployment.GetDeploymentId()),
		fmt.Sprintf("UNKEY_ENVIRONMENT_ID=%s", deployment.GetEnvironmentId()),
		fmt.Sprintf("UNKEY_REGION=%s", d.region),
		fmt.Sprintf("UNKEY_ENVIRONMENT_SLUG=%s", deployment.GetEnvironmentSlug()),
		// UNKEY_INSTANCE_ID is set per-container below
	}

	// Decrypt and inject secrets directly as environment variables
	if len(deployment.GetEncryptedSecretsBlob()) > 0 && d.vault != nil {
		decryptedEnvVars, err := d.decryptSecrets(ctx, deployment.GetEnvironmentId(), deployment.GetEncryptedSecretsBlob())
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to decrypt secrets: %w", err))
		}
		for key, value := range decryptedEnvVars {
			env = append(env, fmt.Sprintf("%s=%s", key, value))
		}
	}

	//nolint:exhaustruct // Docker SDK types have many optional fields
	containerConfig := &container.Config{
		Image: deployment.GetImage(),
		Labels: map[string]string{
			"unkey.deployment.id": deployment.GetDeploymentId(),
			"unkey.managed.by":    "krane",
		},
		ExposedPorts: exposedPorts,
		Env:          env,
	}

	//nolint:exhaustruct // Docker SDK types have many optional fields
	hostConfig := &container.HostConfig{
		PortBindings: portBindings,
		RestartPolicy: container.RestartPolicy{
			Name: "unless-stopped",
		},
		Resources: container.Resources{
			NanoCPUs: cpuNanos,
			Memory:   memoryBytes,
		},
	}

	//nolint:exhaustruct // Docker SDK types have many optional fields
	networkConfig := &network.NetworkingConfig{}

	// Create containers for each replica
	for i := range req.Msg.GetDeployment().GetReplicas() {
		instanceID := fmt.Sprintf("%s-%d", deployment.GetDeploymentId(), i)

		// Add instance-specific env var
		instanceEnv := append(env, fmt.Sprintf("UNKEY_INSTANCE_ID=%s", instanceID)) //nolint:gocritic // intentional append to new slice

		//nolint:exhaustruct // Docker SDK types have many optional fields
		instanceConfig := &container.Config{
			Image:        containerConfig.Image,
			Labels:       containerConfig.Labels,
			ExposedPorts: containerConfig.ExposedPorts,
			Env:          instanceEnv,
		}

		//nolint:exhaustruct // Docker SDK types have many optional fields
		resp, err := d.client.ContainerCreate(
			ctx,
			instanceConfig,
			hostConfig,
			networkConfig,
			nil,
			instanceID,
		)
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to create container: %w", err))
		}

		//nolint:exhaustruct // Docker SDK types have many optional fields
		err = d.client.ContainerStart(ctx, resp.ID, container.StartOptions{})
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to start container: %w", err))
		}
	}

	return connect.NewResponse(&kranev1.CreateDeploymentResponse{
		Status: kranev1.DeploymentStatus_DEPLOYMENT_STATUS_PENDING,
	}), nil
}

// decryptSecrets decrypts an encrypted secrets blob and returns the plain env vars.
func (d *docker) decryptSecrets(ctx context.Context, environmentID string, encryptedBlob []byte) (map[string]string, error) {
	// Step 1: Decrypt outer layer to get SecretsConfig JSON
	decryptedBlobResp, err := d.vault.Decrypt(ctx, &vaultv1.DecryptRequest{
		Keyring:   environmentID,
		Encrypted: string(encryptedBlob),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt secrets blob: %w", err)
	}

	// Step 2: Parse the SecretsConfig (values inside are still encrypted)
	var secretsConfig ctrlv1.SecretsConfig
	if err = protojson.Unmarshal([]byte(decryptedBlobResp.GetPlaintext()), &secretsConfig); err != nil {
		return nil, fmt.Errorf("failed to parse secrets config: %w", err)
	}

	// Step 3: Decrypt each value individually
	envVars := make(map[string]string, len(secretsConfig.GetSecrets()))
	for key, encryptedValue := range secretsConfig.GetSecrets() {
		decrypted, decryptErr := d.vault.Decrypt(ctx, &vaultv1.DecryptRequest{
			Keyring:   environmentID,
			Encrypted: encryptedValue,
		})
		if decryptErr != nil {
			return nil, fmt.Errorf("failed to decrypt env var %s: %w", key, decryptErr)
		}
		envVars[key] = decrypted.GetPlaintext()
	}

	return envVars, nil
}
