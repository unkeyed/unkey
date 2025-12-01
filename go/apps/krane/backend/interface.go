package backend

import "context"

// Backend defines the interface for idempotent container orchestration.
//
// All methods are designed to be fully idempotent, meaning they can be called
// multiple times with the same parameters without causing errors or creating
// duplicate resources. This is critical for reliability in distributed systems.
//
// Implementations include:
//   - Docker backend for single-node deployments
//   - Kubernetes backend for multi-node deployments
type Backend interface {
	// ApplyDeployment creates or updates a deployment in an idempotent manner.
	//
	// This method will:
	//   - Create the deployment if it doesn't exist
	//   - Reuse existing resources if they match the requirements
	//   - Not return an error if the deployment already exists
	//
	// The operation is safe to retry on any error. Network failures, timeouts,
	// and partial failures can all be recovered by retrying this method.
	ApplyDeployment(context.Context, ApplyDeploymentRequest) error

	// GetDeployment retrieves the current status and instances of a deployment.
	//
	// Returns instance details including addresses for service discovery.
	// Returns an error if the deployment doesn't exist.
	GetDeployment(context.Context, GetDeploymentRequest) (GetDeploymentResponse, error)

	// DeleteDeployment removes a deployment and all associated resources.
	//
	// This operation is idempotent and will succeed even if:
	//   - The deployment doesn't exist
	//   - Some resources are already deleted
	//   - Previous deletion attempts failed partially
	//
	// Safe to retry on any error without side effects.
	DeleteDeployment(context.Context, DeleteDeploymentRequest) error

	// ApplyGateway creates or updates a gateway in an idempotent manner.
	//
	// Similar to ApplyDeployment, this method ensures idempotent gateway
	// creation with safe retry semantics.
	ApplyGateway(context.Context, ApplyGatewayRequest) error

	// GetGateway retrieves the current status of a gateway.
	//
	// Returns the gateway status. Returns an error if the gateway doesn't exist.
	GetGateway(context.Context, GetGatewayRequest) (GetGatewayResponse, error)

	// DeleteGateway removes a gateway and all associated resources.
	//
	// This operation is idempotent, similar to DeleteDeployment.
	// Safe to retry on any error.
	DeleteGateway(context.Context, DeleteGatewayRequest) error
}

// ApplyDeploymentRequest contains parameters for creating or updating a deployment.
//
// All fields are used for both creation and updates. The idempotent nature
// means the same request can be sent multiple times safely.
type ApplyDeploymentRequest struct {
	// Namespace for Kubernetes isolation (optional for Docker backend).
	Namespace string
	// WorkspaceID identifies the workspace owning this deployment.
	WorkspaceID string
	// ProjectID identifies the project within the workspace.
	ProjectID string
	// EnvironmentID identifies the environment (dev, staging, prod).
	EnvironmentID string
	// DeploymentID is the unique identifier for this deployment.
	// Used as the idempotency key - same ID always refers to same deployment.
	DeploymentID string
	// Image is the container image to deploy (with tag).
	Image string
	// Replicas is the number of instances to run.
	Replicas int
	// CpuMillicores is CPU allocation in millicores (1000 = 1 CPU).
	CpuMillicores uint32
	// MemorySizeMib is memory allocation in mebibytes.
	MemorySizeMib uint64
}

// GetDeploymentRequest identifies a deployment to query.
type GetDeploymentRequest struct {
	// Namespace where the deployment exists.
	Namespace string
	// DeploymentID to query.
	DeploymentID string
}

// GetDeploymentResponse contains deployment status and instance information.
type GetDeploymentResponse struct {
	// Instances lists all running instances of the deployment.
	// May be empty if deployment is pending or failed.
	Instances []Instance
}

// DeleteDeploymentRequest identifies a deployment to delete.
type DeleteDeploymentRequest struct {
	// Namespace where the deployment exists.
	Namespace string
	// DeploymentID to delete. Deletion is idempotent.
	DeploymentID string
}

// Instance represents a single running instance of a deployment.
type Instance struct {
	// Id is the unique identifier for this instance (container/pod ID).
	Id string
	// Address is the network address for accessing this instance.
	// Format varies by backend (e.g., hostname:port or IP:port).
	Address string
	// Status indicates the current state of this instance.
	Status DeploymentStatus
}

// DeploymentStatus represents the lifecycle state of a deployment instance.
type DeploymentStatus string

const (
	// DEPLOYMENT_STATUS_UNSPECIFIED indicates unknown or invalid status.
	DEPLOYMENT_STATUS_UNSPECIFIED DeploymentStatus = "UNSPECIFIED"
	// DEPLOYMENT_STATUS_PENDING indicates the instance is being created.
	DEPLOYMENT_STATUS_PENDING DeploymentStatus = "PENDING"
	// DEPLOYMENT_STATUS_RUNNING indicates the instance is healthy and serving.
	DEPLOYMENT_STATUS_RUNNING DeploymentStatus = "RUNNING"
	// DEPLOYMENT_STATUS_TERMINATING indicates the instance is shutting down.
	DEPLOYMENT_STATUS_TERMINATING DeploymentStatus = "TERMINATING"
)

// ApplyGatewayRequest contains parameters for creating or updating a gateway.
//
// Gateways are similar to deployments but typically handle ingress traffic
// and routing to backend services.
type ApplyGatewayRequest struct {
	// Namespace for Kubernetes isolation (optional for Docker backend).
	Namespace string
	// WorkspaceID identifies the workspace owning this gateway.
	WorkspaceID string
	// ProjectID identifies the project within the workspace.
	ProjectID string
	// EnvironmentID identifies the environment (dev, staging, prod).
	EnvironmentID string
	// GatewayID is the unique identifier for this gateway.
	// Used as the idempotency key - same ID always refers to same gateway.
	GatewayID string
	// Image is the gateway container image to deploy (with tag).
	Image string
	// Replicas is the number of gateway instances for load balancing.
	Replicas int
	// CpuMillicores is CPU allocation in millicores (1000 = 1 CPU).
	CpuMillicores uint32
	// MemorySizeMib is memory allocation in mebibytes.
	MemorySizeMib uint64
}

// GetGatewayRequest identifies a gateway to query.
type GetGatewayRequest struct {
	// Namespace where the gateway exists.
	Namespace string
	// GatewayID to query.
	GatewayID string
}

// GetGatewayResponse contains the gateway's current status.
type GetGatewayResponse struct {
	// Status indicates the overall gateway state.
	Status GatewayStatus
}

// GatewayStatus represents the lifecycle state of a gateway.
type GatewayStatus string

const (
	// GATEWAY_STATUS_UNSPECIFIED indicates unknown or invalid status.
	GATEWAY_STATUS_UNSPECIFIED GatewayStatus = "UNSPECIFIED"
	// GATEWAY_STATUS_PENDING indicates the gateway is being created.
	GATEWAY_STATUS_PENDING GatewayStatus = "PENDING"
	// GATEWAY_STATUS_RUNNING indicates the gateway is healthy and routing traffic.
	GATEWAY_STATUS_RUNNING GatewayStatus = "RUNNING"
	// GATEWAY_STATUS_TERMINATING indicates the gateway is shutting down.
	GATEWAY_STATUS_TERMINATING GatewayStatus = "TERMINATING"
)

// DeleteGatewayRequest identifies a gateway to delete.
type DeleteGatewayRequest struct {
	// Namespace where the gateway exists.
	Namespace string
	// GatewayID to delete. Deletion is idempotent.
	GatewayID string
}
