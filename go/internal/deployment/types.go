package deployment

import (
	"context"
	"time"
)

type DeploymentStatus string

const (
	StatusPending    DeploymentStatus = "pending"
	StatusBuilding   DeploymentStatus = "building"
	StatusDeploying  DeploymentStatus = "deploying"
	StatusRunning    DeploymentStatus = "running"
	StatusFailed     DeploymentStatus = "failed"
	StatusCanceled   DeploymentStatus = "canceled"
	StatusRolledBack DeploymentStatus = "rolled_back"
)

type DeploymentStep string

const (
	StepSourceDownload DeploymentStep = "source_download"
	StepBuildImage     DeploymentStep = "build_image"
	StepPushImage      DeploymentStep = "push_image"
	StepProvision      DeploymentStep = "provision_resources"
	StepDeploy         DeploymentStep = "deploy_application"
	StepHealthCheck    DeploymentStep = "health_check"
	StepTrafficRoute   DeploymentStep = "traffic_routing"
)

type Deployment struct {
	ID           string            `json:"id"`
	CustomerID   string            `json:"customer_id"`
	ProjectID    string            `json:"project_id"`
	Status       DeploymentStatus  `json:"status"`
	CurrentStep  DeploymentStep    `json:"current_step"`
	Source       SourceConfig      `json:"source"`
	Build        BuildConfig       `json:"build"`
	Runtime      RuntimeConfig     `json:"runtime"`
	Resources    ResourceConfig    `json:"resources"`
	Metadata     DeploymentMeta    `json:"metadata"`
	Steps        []StepResult      `json:"steps"`
	CreatedAt    time.Time         `json:"created_at"`
	UpdatedAt    time.Time         `json:"updated_at"`
	CompletedAt  *time.Time        `json:"completed_at,omitempty"`
}

type SourceConfig struct {
	Type       string            `json:"type"` // git, archive, docker
	Repository string            `json:"repository,omitempty"`
	Branch     string            `json:"branch,omitempty"`
	Commit     string            `json:"commit,omitempty"`
	URL        string            `json:"url,omitempty"`
	Headers    map[string]string `json:"headers,omitempty"`
}

type BuildConfig struct {
	Dockerfile   string            `json:"dockerfile,omitempty"`
	BuildContext string            `json:"build_context,omitempty"`
	BuildArgs    map[string]string `json:"build_args,omitempty"`
	Command      []string          `json:"command,omitempty"`
	Environment  map[string]string `json:"environment,omitempty"`
}

type RuntimeConfig struct {
	Port        int               `json:"port"`
	Command     []string          `json:"command,omitempty"`
	Environment map[string]string `json:"environment,omitempty"`
	HealthCheck HealthCheckConfig `json:"health_check"`
}

type HealthCheckConfig struct {
	Path               string        `json:"path"`
	Port               int           `json:"port,omitempty"`
	IntervalSeconds    int           `json:"interval_seconds"`
	TimeoutSeconds     int           `json:"timeout_seconds"`
	HealthyThreshold   int           `json:"healthy_threshold"`
	UnhealthyThreshold int           `json:"unhealthy_threshold"`
	GracePeriod        time.Duration `json:"grace_period"`
}

type ResourceConfig struct {
	CPU    string `json:"cpu"`    // "0.5", "1", "2"
	Memory string `json:"memory"` // "512Mi", "1Gi", "2Gi"
	Disk   string `json:"disk"`   // "1Gi", "10Gi"
	Region string `json:"region"`
}

type DeploymentMeta struct {
	ImageURI     string    `json:"image_uri,omitempty"`
	ServiceName  string    `json:"service_name,omitempty"`
	Domain       string    `json:"domain,omitempty"`
	InternalURL  string    `json:"internal_url,omitempty"`
	ExternalURL  string    `json:"external_url,omitempty"`
	BuildTime    *Duration `json:"build_time,omitempty"`
	DeployTime   *Duration `json:"deploy_time,omitempty"`
	BuildLogs    string    `json:"build_logs,omitempty"`
	DeployLogs   string    `json:"deploy_logs,omitempty"`
}

type StepResult struct {
	Step        DeploymentStep `json:"step"`
	Status      StepStatus     `json:"status"`
	StartedAt   time.Time      `json:"started_at"`
	CompletedAt *time.Time     `json:"completed_at,omitempty"`
	Duration    *Duration      `json:"duration,omitempty"`
	Attempts    int            `json:"attempts"`
	Error       *StepError     `json:"error,omitempty"`
	Output      string         `json:"output,omitempty"`
}

type StepStatus string

const (
	StepStatusPending    StepStatus = "pending"
	StepStatusRunning    StepStatus = "running"
	StepStatusCompleted  StepStatus = "completed"
	StepStatusFailed     StepStatus = "failed"
	StepStatusRetrying   StepStatus = "retrying"
	StepStatusSkipped    StepStatus = "skipped"
)

type StepError struct {
	Code       string    `json:"code"`
	Message    string    `json:"message"`
	Details    string    `json:"details,omitempty"`
	Retryable  bool      `json:"retryable"`
	OccurredAt time.Time `json:"occurred_at"`
}

type Duration struct {
	time.Duration
}

func (d *Duration) MarshalJSON() ([]byte, error) {
	return []byte(`"` + d.String() + `"`), nil
}

func (d *Duration) UnmarshalJSON(data []byte) error {
	str := string(data[1 : len(data)-1])
	dur, err := time.ParseDuration(str)
	if err != nil {
		return err
	}
	d.Duration = dur
	return nil
}

type DeploymentOrchestrator interface {
	Deploy(ctx context.Context, deployment *Deployment) error
	Cancel(ctx context.Context, deploymentID string) error
	GetStatus(ctx context.Context, deploymentID string) (*Deployment, error)
	ListDeployments(ctx context.Context, customerID string, opts ListOptions) ([]*Deployment, error)
	Rollback(ctx context.Context, deploymentID string, targetDeploymentID string) error
}

type StepExecutor interface {
	Execute(ctx context.Context, deployment *Deployment) error
	Rollback(ctx context.Context, deployment *Deployment) error
	GetName() DeploymentStep
	IsRetryable(err error) bool
	MaxRetries() int
	RetryDelay() time.Duration
}

type ListOptions struct {
	Limit  int    `json:"limit"`
	Offset int    `json:"offset"`
	Status string `json:"status,omitempty"`
}

type DeploymentEvent struct {
	Type         string                 `json:"type"`
	DeploymentID string                 `json:"deployment_id"`
	CustomerID   string                 `json:"customer_id"`
	ProjectID    string                 `json:"project_id"`
	Step         DeploymentStep         `json:"step,omitempty"`
	Status       DeploymentStatus       `json:"status"`
	Data         map[string]interface{} `json:"data,omitempty"`
	Timestamp    time.Time              `json:"timestamp"`
}

type EventPublisher interface {
	Publish(ctx context.Context, event *DeploymentEvent) error
}

type LogStreamer interface {
	StreamBuildLogs(ctx context.Context, deploymentID string) (<-chan string, error)
	StreamDeployLogs(ctx context.Context, deploymentID string) (<-chan string, error)
	GetLogs(ctx context.Context, deploymentID string, step DeploymentStep) (string, error)
}

type ResourceManager interface {
	ProvisionResources(ctx context.Context, deployment *Deployment) error
	DestroyResources(ctx context.Context, deployment *Deployment) error
	GetResourceStatus(ctx context.Context, deployment *Deployment) (ResourceStatus, error)
}

type ResourceStatus struct {
	CPU    ResourceMetric `json:"cpu"`
	Memory ResourceMetric `json:"memory"`
	Disk   ResourceMetric `json:"disk"`
	Status string         `json:"status"`
}

type ResourceMetric struct {
	Used      string  `json:"used"`
	Total     string  `json:"total"`
	Percentage float64 `json:"percentage"`
}