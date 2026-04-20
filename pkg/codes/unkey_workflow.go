package codes

// workflowProvider defines errors caused by external providers we depend on.
type workflowProvider struct {
	// DepotBuildFailed indicates the Depot remote build SDK returned an error.
	DepotBuildFailed Code

	// DepotMachineUnavailable indicates Depot could not allocate a builder machine.
	DepotMachineUnavailable Code

	// AcmeRateLimited indicates the ACME provider (Let's Encrypt) rate-limited us.
	AcmeRateLimited Code

	// AcmeUnauthorized indicates the ACME provider rejected our credentials.
	AcmeUnauthorized Code

	// GitHubInstallationToken indicates we failed to obtain a GitHub installation token.
	GitHubInstallationToken Code
}

// workflowApp defines errors caused by the user's own application.
type workflowApp struct {
	// BuildBroken indicates the user's Dockerfile or build context failed to compile.
	BuildBroken Code

	// HealthcheckFailed indicates user app instances never reported healthy.
	HealthcheckFailed Code

	// InstanceNotReady indicates user app instances did not become ready in time.
	InstanceNotReady Code
}

// workflowInfra defines errors caused by our own infrastructure or control plane.
type workflowInfra struct {
	// KraneTimeout indicates a krane operation exceeded its deadline.
	KraneTimeout Code

	// SentinelDeployTimeout indicates a sentinel deployment did not become ready in time.
	SentinelDeployTimeout Code
}

// UnkeyWorkflowErrors defines all workflow-related errors used by the
// pkg/restate/observability layer to classify failures by blame.
type UnkeyWorkflowErrors struct {
	// Provider contains errors caused by external providers.
	Provider workflowProvider

	// App contains errors caused by the user's own application.
	App workflowApp

	// Infra contains errors caused by our own infrastructure.
	Infra workflowInfra
}

// Workflow contains all predefined workflow error codes used by the
// observability layer to categorize failures across Restate workflows.
var Workflow = UnkeyWorkflowErrors{
	Provider: workflowProvider{
		DepotBuildFailed:        Code{SystemDepot, CategoryWorkflowProvider, "depot_build_failed"},
		DepotMachineUnavailable: Code{SystemDepot, CategoryWorkflowProvider, "depot_machine_unavailable"},
		AcmeRateLimited:         Code{SystemAcme, CategoryWorkflowProvider, "acme_rate_limited"},
		AcmeUnauthorized:        Code{SystemAcme, CategoryWorkflowProvider, "acme_unauthorized"},
		GitHubInstallationToken: Code{SystemGitHub, CategoryWorkflowProvider, "github_installation_token"},
	},
	App: workflowApp{
		BuildBroken:       Code{SystemUser, CategoryWorkflowApp, "build_broken"},
		HealthcheckFailed: Code{SystemUser, CategoryWorkflowApp, "healthcheck_failed"},
		InstanceNotReady:  Code{SystemUser, CategoryWorkflowApp, "instance_not_ready"},
	},
	Infra: workflowInfra{
		KraneTimeout:          Code{SystemUnkey, CategoryWorkflowInfra, "krane_timeout"},
		SentinelDeployTimeout: Code{SystemUnkey, CategoryWorkflowInfra, "sentinel_deploy_timeout"},
	},
}
