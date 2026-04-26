package preflight

import (
	"context"
	"fmt"

	"github.com/unkeyed/unkey/pkg/clickhouse"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/logger"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/preflight/core"
	"github.com/unkeyed/unkey/svc/preflight/runner"
	"github.com/unkeyed/unkey/svc/preflight/runner/diskupload"
)

// Config is the full preflight runtime configuration. Loaded from a
// TOML file by pkg/config; see cmd/preflight/unkey.toml.example for
// the canonical shape.
//
// Secrets are NOT stored in the TOML file directly. Use ${VAR}
// interpolation in the TOML and populate the env vars via an
// ExternalSecret / Kubernetes Secret at runtime; pkg/config runs
// os.ExpandEnv before unmarshalling, so "${PREFLIGHT_CTRL_TOKEN}"
// resolves from the process environment.
type Config struct {
	Target    core.Target `toml:"target" config:"required,nonempty"`
	Region    string      `toml:"region" config:"required,nonempty"`
	SuiteName string      `toml:"suite" config:"default=solo"`

	Ctrl       CtrlConfig       `toml:"ctrl"`
	GitHub     GitHubConfig     `toml:"github"`
	Tenant     TenantConfig     `toml:"preflight_tenant"`
	ClickHouse ClickHouseConfig `toml:"clickhouse"`
	MySQL      MySQLConfig      `toml:"mysql"`
	Artifacts  ArtifactsConfig  `toml:"artifacts"`
}

// ArtifactsConfig controls where failure diagnostics are written.
// Default behaviour writes to ./preflight-artifacts/ in the working
// directory; override DiskRoot for a different path, or set Disabled
// to true to drop artifacts entirely (useful for short-lived sandbox
// runs that have nothing to grep through anyway).
type ArtifactsConfig struct {
	DiskRoot string `toml:"disk_root" config:"default=./preflight-artifacts"`
	Disabled bool   `toml:"disabled"`
}

// CtrlConfig covers the control-plane API. AuthToken is a secret;
// supply via ${PREFLIGHT_CTRL_TOKEN} in the TOML.
type CtrlConfig struct {
	BaseURL   string `toml:"base_url" config:"required,nonempty"`
	AuthToken string `toml:"auth_token" config:"required,nonempty"`
}

// GitHubConfig carries the dedicated preflight GitHub App credentials
// and metadata. WebhookSecret and PrivateKeyPEM are secrets.
type GitHubConfig struct {
	// WebhookSecret is the HMAC secret the ctrl API verifies webhook
	// payloads against. Used by the github_webhook probe to sign
	// synthetic `push` events.
	WebhookSecret string `toml:"webhook_secret"`

	// AppID / InstallationID identify the preflight App + its
	// installation on the test repo. Numeric; not secret but
	// frequently lives alongside the private key.
	AppID          int64 `toml:"app_id"`
	InstallationID int64 `toml:"installation_id"`

	// PrivateKeyPEM is the App's PEM-encoded private key. Always a
	// secret; supply via ${PREFLIGHT_GITHUB_PRIVATE_KEY}.
	PrivateKeyPEM string `toml:"private_key_pem"`

	// TestRepo is "owner/repo" of the preflight test repository.
	// Defaults to unkeyed/preflight-test-app; override only if you
	// actually stand up a different repo.
	TestRepo string `toml:"test_repo" config:"default=unkeyed/preflight-test-app"`
}

// TenantConfig identifies the dedicated preflight workspace/project/
// app/environment the probes target. All non-secret; hardcoded per
// environment in the Helm values.
type TenantConfig struct {
	ProjectID       string `toml:"project_id"`
	AppID           string `toml:"app_id"`
	EnvironmentSlug string `toml:"environment_slug" config:"default=production"`

	// Slugs + apex assemble the per-commit hostname the git_push
	// probe polls. Match what ctrl writes into frontline_routes.
	ProjectSlug   string `toml:"project_slug"`
	AppSlug       string `toml:"app_slug"`
	WorkspaceSlug string `toml:"workspace_slug"`
	Apex          string `toml:"apex"`
}

// ClickHouseConfig: URL is a secret.
type ClickHouseConfig struct {
	URL string `toml:"url"`
}

// MySQLConfig: DSN is a secret AND must be scoped read-only to the
// deploy-pipeline tables (see docs/runbooks/preflight/setup.md).
type MySQLConfig struct {
	DSN string `toml:"dsn"`
}

// Run executes a single preflight suite against the target described
// by cfg and returns nil only when every probe in the suite passes.
//
// Dev target is explicitly rejected here: dev is a test-only path
// implemented by the svc/preflight/harness package.
func Run(ctx context.Context, cfg Config) error {
	switch cfg.Target {
	case core.TargetStaging, core.TargetProd:
		// Valid binary targets.
	case core.TargetDev:
		return fmt.Errorf(
			"preflight: the 'dev' target is test-only; run " +
				"`go test -run TestDev ./svc/preflight/harness/...` instead",
		)
	default:
		return fmt.Errorf("preflight: unknown target %q (expected staging|prod)", cfg.Target)
	}

	runID := uid.New("pflt")

	logger.Info("preflight: run starting",
		"target", string(cfg.Target),
		"region", cfg.Region,
		"suite", cfg.SuiteName,
		"run_id", runID,
	)

	// ClickHouse is optional at the binary level: only wire it when a
	// URL is provided. Probes that need CH but find it nil return a
	// loud error via the clickhouse_connectivity prereq probe.
	var chClient clickhouse.ClickHouse
	if cfg.ClickHouse.URL != "" {
		c, err := clickhouse.New(clickhouse.Config{URL: cfg.ClickHouse.URL})
		if err != nil {
			return fmt.Errorf("preflight: clickhouse connect: %w", err)
		}
		chClient = c
	}

	// MySQL is diagnostics-only in binary runs. The DSN MUST be the
	// read-only scoped credential described in core.Env.DB; that scope
	// is enforced at the DB layer via GRANT statements (see setup.md),
	// not at the Go layer. When no DSN is set the Diagnoser path simply
	// does not run.
	var dbHandle db.Database
	if cfg.MySQL.DSN != "" {
		h, err := db.New(db.Config{PrimaryDSN: cfg.MySQL.DSN, ReadOnlyDSN: ""})
		if err != nil {
			return fmt.Errorf("preflight: mysql connect: %w", err)
		}
		dbHandle = h
	}

	env := &core.Env{
		Target:                   cfg.Target,
		Region:                   cfg.Region,
		RunID:                    runID,
		CtrlBaseURL:              cfg.Ctrl.BaseURL,
		CtrlAuthToken:            cfg.Ctrl.AuthToken,
		GitHubWebhookSecret:      cfg.GitHub.WebhookSecret,
		PreflightProjectID:       cfg.Tenant.ProjectID,
		PreflightAppID:           cfg.Tenant.AppID,
		PreflightEnvironmentSlug: cfg.Tenant.EnvironmentSlug,
		PreflightProjectSlug:     cfg.Tenant.ProjectSlug,
		PreflightAppSlug:         cfg.Tenant.AppSlug,
		PreflightWorkspaceSlug:   cfg.Tenant.WorkspaceSlug,
		PreflightApex:            cfg.Tenant.Apex,
		GitHubAppID:              cfg.GitHub.AppID,
		GitHubInstallationID:     cfg.GitHub.InstallationID,
		GitHubPrivateKeyPEM:      cfg.GitHub.PrivateKeyPEM,
		PreflightTestRepo:        cfg.GitHub.TestRepo,
		DB:                       dbHandle,
		ClickHouse:               chClient,
	}

	// Artifact upload defaults to a local-disk sink so dev runs leave
	// something behind to grep. Production swaps in an S3-backed
	// uploader behind the same interface.
	var artifacts runner.ArtifactUploader
	if !cfg.Artifacts.Disabled {
		artifacts = diskupload.New(cfg.Artifacts.DiskRoot)
	}
	r := runner.New(cfg.SuiteName, cfg.Region, artifacts, nil)

	suite, err := selectSuite(cfg.SuiteName)
	if err != nil {
		return err
	}

	results := RunSuite(ctx, r, env, suite)
	if failed := countFailed(results); failed > 0 {
		return fmt.Errorf("preflight: %d probe(s) failed", failed)
	}
	return nil
}

func selectSuite(name string) (Suite, error) {
	switch name {
	case "solo", "":
		return DefaultSolo(), nil
	case "solo-dev":
		// Excludes realinfra/* probes. Useful when running against
		// the local minikube stack without a GitHub App + ngrok
		// tunnel wired up. TestDev uses this suite internally.
		return DevSuite(), nil
	default:
		return Suite{}, fmt.Errorf("preflight: unknown suite %q", name)
	}
}

func countFailed(results []core.Result) int {
	var n int
	for _, r := range results {
		if !r.OK {
			n++
		}
	}

	return n
}
