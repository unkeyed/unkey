package core

import (
	"github.com/unkeyed/unkey/pkg/clickhouse"
	"github.com/unkeyed/unkey/pkg/db"
)

// Env is the shared client container passed to every probe. Clients are
// dialed once at the top of a run and reused across probes. Probes MUST
// NOT open their own connections; add a client here instead.
//
// Field types firm up as probes land. Anything typed as a real package
// type is considered stable; fields still typed as `any` are placeholders
// that will be replaced once a probe needs them (and that probe gets to
// pick the type).
type Env struct {
	// Target identifies where this run is pointed (dev, staging, prod).
	Target Target

	// Region is the AWS region this runner is executing in, or "local"
	// for the dev harness.
	Region string

	// RunID uniquely identifies this preflight run. Propagated as the
	// X-Preflight-Run-Id header on every synthetic request so sentinel
	// logs can be correlated back in ClickHouse.
	RunID string

	// CtrlBaseURL is the base URL of the control-plane HTTP API. Probes
	// construct their own http.Client rather than sharing one because
	// different probes want different transports (h2c vs http/1.1, etc.).
	CtrlBaseURL string

	// CtrlAuthToken is the bearer token the control-plane API accepts.
	// Workspace-scoped in staging/prod so preflight cannot reach real
	// tenants.
	CtrlAuthToken string

	// GitHubWebhookSecret is the HMAC secret the ctrl API verifies
	// webhook payloads against. Used by the tier-1.1 probe to sign its
	// synthetic `push` events.
	GitHubWebhookSecret string

	// PreflightProjectID / PreflightAppID / PreflightEnvironmentSlug
	// identify the dedicated preflight tenant. Probes that call
	// CreateDeployment (tier 1.3) target this project, so a compromised
	// run cannot touch real customer data.
	PreflightProjectID       string
	PreflightAppID           string
	PreflightEnvironmentSlug string

	// PreflightProjectSlug / PreflightAppSlug / PreflightWorkspaceSlug /
	// PreflightApex are the pieces needed to compute per-commit
	// hostnames for tier-1.10 and tier-1+ probes that hit deployed URLs.
	// A per-commit hostname follows
	// <project>-<app>-git-<sha>-<workspace>.<apex>; all four pieces are
	// workspace-scoped config, not secrets.
	PreflightProjectSlug   string
	PreflightAppSlug       string
	PreflightWorkspaceSlug string
	PreflightApex          string

	// GitHubAppID / GitHubInstallationID / GitHubPrivateKeyPEM are the
	// credentials used to authenticate as the dedicated preflight
	// GitHub App. Populated for probes that need to push commits
	// (tier-1.10 git_push) or create deployments (tier-1.2 build_depot).
	//
	// MUST be the preflight-only App; reusing the production App means
	// a probe bug could forge any webhook.
	GitHubAppID          int64
	GitHubInstallationID int64
	GitHubPrivateKeyPEM  string

	// PreflightTestRepo is "owner/repo" for the dedicated preflight
	// repository (e.g. "unkeyed/preflight-test-app"). Probes that push
	// or deploy target this repo exclusively.
	PreflightTestRepo string

	// DB is a direct MySQL handle. Scoped by whichever credential the
	// binary was handed.
	//
	// Harness always populates this fully; probes running against the
	// harness may use it freely for both assertion and diagnostics.
	//
	// Binary runs against staging/prod may populate this with a
	// READ-ONLY credential that has GRANT SELECT only on the
	// deploy-pipeline tables (`deployments`, `deployment_steps`,
	// `deployment_changes`, `instances`, `frontline_routes`). When
	// present, probes MUST use it only for diagnostics on failure,
	// never for primary assertions. Primary assertions flow through
	// the ctrl API so auth / RLS / workspace scoping regressions
	// actually fail the probe.
	//
	// When nil, diagnostic-only reads simply do not happen and probes
	// fall back to whatever artifacts they can gather without it.
	DB db.Database

	// ClickHouse is the analytics client. Populated in every target,
	// including staging and prod, because ClickHouse is the only path
	// that answers "did my sentinel log arrive". Running it through an
	// RPC would defeat the purpose (the RPC would just wrap the same
	// query).
	//
	// Credentials are minted the same way customer SDKs would get
	// them, via the per-workspace ClickHouse user mechanism in
	// svc/ctrl/worker/clickhouseuser. Blast radius of a leaked
	// credential is preflight's own analytics data, which is designed
	// to be customer-readable anyway.
	ClickHouse clickhouse.ClickHouse

	// When a future probe needs a Kubernetes, Thanos, or GitHub client
	// on Env, add it here as a typed field in the same PR. Do not add
	// `any`-typed placeholders: they erase the type contract the
	// Runner and probes depend on and invite incorrect usage.
}

// Target distinguishes the three environments a preflight binary runs
// against. Dev is implemented by the harness package and is only
// reachable via `go test`; the binary supports staging and prod only.
type Target string

const (
	TargetDev     Target = "dev"
	TargetStaging Target = "staging"
	TargetProd    Target = "prod"
)
