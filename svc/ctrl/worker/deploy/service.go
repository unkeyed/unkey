package deploy

import (
	"k8s.io/client-go/kubernetes"

	hydrav1 "github.com/unkeyed/unkey/gen/proto/hydra/v1"
	"github.com/unkeyed/unkey/gen/rpc/vault"
	"github.com/unkeyed/unkey/pkg/assert"
	"github.com/unkeyed/unkey/pkg/batch"
	"github.com/unkeyed/unkey/pkg/clickhouse"
	"github.com/unkeyed/unkey/pkg/clickhouse/schema"
	"github.com/unkeyed/unkey/svc/ctrl/internal/db"
	githubclient "github.com/unkeyed/unkey/svc/ctrl/worker/github"
)

// BuildBackend identifies which system executes container image builds.
type BuildBackend string

const (
	// BuildBackendDepot runs builds on Depot.dev remote BuildKit machines.
	BuildBackendDepot BuildBackend = "depot"

	// BuildBackendKubernetes runs each build as a one-off BuildKit Job in
	// the cluster the worker itself runs in. Local development only: the
	// build pod runs privileged and offers no isolation beyond the pod
	// boundary, which is acceptable only for builds you already trust.
	BuildBackendKubernetes BuildBackend = "kubernetes"
)

// BuildConfig selects and configures the backend that executes builds.
type BuildConfig struct {
	Backend    BuildBackend
	Depot      DepotConfig
	Kubernetes KubernetesBuildConfig
}

// KubernetesBuildConfig configures the Kubernetes Job build backend.
type KubernetesBuildConfig struct {
	// Namespace is where build Jobs are created. The worker's service
	// account needs permission to manage Jobs and read Pods there.
	Namespace string

	// Image is the BuildKit daemon image each build Job runs.
	Image string
}

// BuildPlatform specifies the target platform for container builds.
type BuildPlatform struct {
	Platform     string
	Architecture string
}

// DepotConfig holds configuration for connecting to the Depot.dev API.
type DepotConfig struct {
	APIUrl        string
	ProjectRegion string
	ProjectPrefix string
}

// RegistryConfig holds credentials for the container registry.
type RegistryConfig struct {
	Repository string
	Username   string
	Password   string

	// Insecure allows plain-HTTP pushes. Only for local registries without
	// TLS; never enable it against a production registry.
	Insecure bool
}

// Workflow orchestrates deployment lifecycle operations.
//
// This workflow manages the complete deployment lifecycle including deploying new versions,
// rolling back to previous versions, and promoting deployments to live. It coordinates
// between container orchestration (Krane), database updates, and domain routing to ensure
// consistent deployment state.
//
// The workflow uses Restate virtual objects keyed by app ID to ensure that only one
// deployment operation runs per app at any time, preventing race conditions during
// concurrent deploy/rollback/promote operations while allowing parallel deploys
// across different apps within the same project.
type Workflow struct {
	hydrav1.UnimplementedDeployServiceServer
	db db.Database

	defaultDomain string
	vault         vault.VaultServiceClient

	github githubclient.GitHubClient

	// Build dependencies
	buildConfig                     BuildConfig
	k8s                             kubernetes.Interface
	registryConfig                  RegistryConfig
	buildPlatform                   BuildPlatform
	clickhouse                      clickhouse.ClickHouse
	buildSteps                      *batch.BatchProcessor[schema.BuildStepV1]
	buildStepLogs                   *batch.BatchProcessor[schema.BuildStepLogV1]
	allowUnauthenticatedDeployments bool
	dashboardURL                    string
}

var _ hydrav1.DeployServiceServer = (*Workflow)(nil)

// Config holds the configuration for creating a deployment workflow.
type Config struct {
	// DB is the main database connection for workspace, project, and deployment data.
	DB db.Database

	// DefaultDomain is the apex domain for generated deployment URLs (e.g., "unkey.app").
	DefaultDomain string

	// Vault provides encryption/decryption services for secrets.
	Vault vault.VaultServiceClient

	// GitHub provides access to GitHub API for downloading tarballs.
	GitHub githubclient.GitHubClient

	// Build selects and configures the build backend. See [BuildConfig].
	Build BuildConfig

	// K8s is the cluster client used by the kubernetes build backend to run
	// build Jobs. Required when Build.Backend is [BuildBackendKubernetes],
	// unused otherwise.
	K8s kubernetes.Interface

	// RegistryConfig provides credentials for the container registry.
	RegistryConfig RegistryConfig

	// BuildPlatform specifies the target platform for all builds.
	BuildPlatform BuildPlatform

	// Clickhouse provides query access for deployment request counts.
	Clickhouse clickhouse.ClickHouse

	// BuildSteps buffers build step events for ClickHouse.
	BuildSteps *batch.BatchProcessor[schema.BuildStepV1]

	// BuildStepLogs buffers build step log events for ClickHouse.
	BuildStepLogs *batch.BatchProcessor[schema.BuildStepLogV1]

	// AllowUnauthenticatedDeployments controls whether builds can skip GitHub authentication.
	// Set to true only for local development with public repositories.
	AllowUnauthenticatedDeployments bool

	// DashboardURL is the base URL of the dashboard for constructing log URLs
	// in GitHub deployment statuses (e.g., "https://app.unkey.com").
	DashboardURL string
}

// New creates a new deployment workflow instance.
func New(cfg Config) (*Workflow, error) {
	if cfg.Build.Backend == BuildBackendKubernetes {
		if err := assert.NotNil(cfg.K8s, "kubernetes build backend requires a k8s client"); err != nil {
			return nil, err
		}
	}

	// Reclaim build workspaces orphaned by a previous crash. Runs before any
	// handler is bound, so no live workspace can match.
	cleanupStaleRailpackWorkspaces()

	return &Workflow{
		UnimplementedDeployServiceServer: hydrav1.UnimplementedDeployServiceServer{},
		db:                               cfg.DB,
		defaultDomain:                    cfg.DefaultDomain,
		vault:                            cfg.Vault,

		github:                          cfg.GitHub,
		buildConfig:                     cfg.Build,
		k8s:                             cfg.K8s,
		registryConfig:                  cfg.RegistryConfig,
		buildPlatform:                   cfg.BuildPlatform,
		clickhouse:                      cfg.Clickhouse,
		buildSteps:                      cfg.BuildSteps,
		buildStepLogs:                   cfg.BuildStepLogs,
		allowUnauthenticatedDeployments: cfg.AllowUnauthenticatedDeployments,
		dashboardURL:                    cfg.DashboardURL,
	}, nil
}
