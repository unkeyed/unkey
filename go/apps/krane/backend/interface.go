package backend

import "context"

type Backend interface {
	CreateDeployment(context.Context, CreateDeploymentRequest) error

	GetDeployment(context.Context, GetDeploymentRequest) (GetDeploymentResponse, error)

	DeleteDeployment(context.Context, DeleteDeploymentRequest) error

	CreateGateway(context.Context, CreateGatewayRequest) error
	GetGateway(context.Context, GetGatewayRequest) (GetGatewayResponse, error)
	DeleteGateway(context.Context, DeleteGatewayRequest) error
}

type CreateDeploymentRequest struct {
	Namespace     string
	WorkspaceID   string
	ProjectID     string
	EnvironmentID string
	DeploymentID  string
	Image         string
	Replicas      int
	CpuMillicores uint32
	MemorySizeMib uint64
}

type GetDeploymentRequest struct {
	Namespace    string
	DeploymentID string
}

type GetDeploymentResponse struct {
	Instances []Instance
}

type DeleteDeploymentRequest struct {
	Namespace    string
	DeploymentID string
}

type Instance struct {
	Id      string
	Address string
	Status  DeploymentStatus
}

type DeploymentStatus string

const (
	DEPLOYMENT_STATUS_UNSPECIFIED DeploymentStatus = "UNSPECIFIED"
	DEPLOYMENT_STATUS_PENDING     DeploymentStatus = "PENDING"
	DEPLOYMENT_STATUS_RUNNING     DeploymentStatus = "RUNNING"
	DEPLOYMENT_STATUS_TERMINATING DeploymentStatus = "TERMINATING"
)

type CreateGatewayRequest struct {
	Namespace     string
	WorkspaceID   string
	ProjectID     string
	EnvironmentID string
	GatewayID     string
	Image         string
	Replicas      int
	CpuMillicores uint32
	MemorySizeMib uint64
}

type GetGatewayRequest struct {
	Namespace string
	GatewayID string
}

type GetGatewayResponse struct {
	Status GatewayStatus
}

type GatewayStatus string

const (
	GATEWAY_STATUS_UNSPECIFIED GatewayStatus = "UNSPECIFIED"
	GATEWAY_STATUS_PENDING     GatewayStatus = "PENDING"
	GATEWAY_STATUS_RUNNING     GatewayStatus = "RUNNING"
	GATEWAY_STATUS_TERMINATING GatewayStatus = "TERMINATING"
)

type DeleteGatewayRequest struct {
	Namespace string
	GatewayID string
}
