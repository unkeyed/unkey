// Package git_push is the end-to-end customer-flow probe. A commit
// is pushed to the preflight test repo and the per-commit deploy
// hostname is polled until /meta echoes the new SHA. Every leg of
// the pipeline — webhook, build, deploy, routing, sentinel, testapp
// — has to work for this probe to pass.
package git_push

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/unkeyed/unkey/pkg/httpx"
	"github.com/unkeyed/unkey/svc/preflight/core"
	"github.com/unkeyed/unkey/svc/preflight/internal/dbdiag"
	"github.com/unkeyed/unkey/svc/preflight/internal/githubpush"
	"github.com/unkeyed/unkey/svc/preflight/internal/k8sdiag"
	"github.com/unkeyed/unkey/svc/preflight/internal/wait"
	"github.com/unkeyed/unkey/svc/preflight/probes"
)

const (
	// branchPrefix isolates each region's pushes so two regional
	// runners never step on each other's branch HEAD. The branch is
	// reused across runs; GitHub's update-file endpoint handles both
	// create and update transparently.
	branchPrefix = "preflight/"

	// sentinelFile is the path we rewrite on every run. The test
	// repo's build config MUST watch this path (so the push triggers
	// a rebuild); see setup.md.
	sentinelFile = ".preflight-timestamp"

	// pushTimeout covers GitHub installation-token exchange + contents
	// GET/PUT.
	pushTimeout = 30 * time.Second

	// deployTimeout covers webhook -> Restate -> Depot build -> Krane
	// provision -> frontline route -> testapp response. 10 minutes is
	// generous: real deploys usually finish in 2-3 minutes but a cold
	// Depot or a new region can stretch further.
	deployTimeout = 10 * time.Minute

	// metaPollPeriod keeps frontline traffic reasonable while still
	// catching the READY transition promptly.
	metaPollPeriod = 5 * time.Second
)

type Probe struct{}

// Name implements probes.Probe.
func (Probe) Name() string { return "git_push" }

// Run implements probes.Probe.
func (Probe) Run(ctx context.Context, env *core.Env) core.Result {
	if err := validatePrereqs(env); err != nil {
		return core.Fail(err)
	}

	branch := branchPrefix + env.Region
	repo := env.PreflightTestRepo
	content := []byte(fmt.Sprintf("%s %s\n", env.RunID, time.Now().UTC().Format(time.RFC3339Nano)))
	message := "preflight: " + env.RunID

	client, err := githubpush.New(githubpush.Config{
		AppID:          env.GitHubAppID,
		InstallationID: env.GitHubInstallationID,
		PrivateKeyPEM:  env.GitHubPrivateKeyPEM,
		BaseURL:        "", // default api.github.com
		HTTPClient:     nil,
		Now:            nil,
	})
	if err != nil {
		return core.Failf("githubpush client: %w", err)
	}

	phases := make([]core.Phase, 0, 3)

	pushCtx, pushCancel := context.WithTimeout(ctx, pushTimeout)
	defer pushCancel()

	pushStart := time.Now()
	sha, pushErr := client.PushFile(pushCtx, repo, branch, sentinelFile, content, message)
	phases = append(phases, core.Phase{Name: "push", Duration: time.Since(pushStart), Err: pushErr})
	if pushErr != nil {
		return core.Failf("push commit: %w", pushErr).
			WithPhases(phases).
			WithDims(map[string]string{"repo": repo, "branch": branch})
	}

	hostname := perCommitHostname(env, sha)
	deployCtx, deployCancel := context.WithTimeout(ctx, deployTimeout)
	defer deployCancel()

	deployStart := time.Now()
	_, waitErr := wait.Poll(deployCtx, metaPollPeriod, func(ctx context.Context) (struct{}, bool, error) {
		ok, err := metaReportsSHA(ctx, hostname, sha)
		if err != nil {
			return struct{}{}, false, nil
		}
		return struct{}{}, ok, nil
	})

	phases = append(phases, core.Phase{Name: "await_deploy", Duration: time.Since(deployStart), Err: waitErr})
	dims := map[string]string{
		"repo":     repo,
		"branch":   branch,
		"commit":   sha,
		"hostname": hostname,
	}
	if waitErr != nil {
		return core.Failf("deploy did not serve commit %s within %s: %w", shortSHA(sha), deployTimeout, waitErr).
			WithPhases(phases).
			WithDims(dims)
	}

	// Fire a traceable request so request_logs has a deterministic
	// row to read. Transport failure surfaces as a dim; it does not
	// fail git_push because the deploy itself is already verified.
	traceStart := time.Now()
	traceErr := emitTracerRequest(ctx, hostname, env.RunID)
	phases = append(phases, core.Phase{Name: "emit_tracer", Duration: time.Since(traceStart), Err: traceErr})
	if traceErr != nil {
		dims["trace_err"] = traceErr.Error()
	}

	return core.Pass().WithPhases(phases).WithDims(dims)
}

// emitTracerRequest hits /preflight-<runID> so sentinel writes a row
// for request_logs to find. A 4xx/5xx from the app is still a logged
// request, so only transport failures are errors here.
func emitTracerRequest(ctx context.Context, hostname, runID string) error {
	url := fmt.Sprintf("https://%s/preflight-%s", hostname, runID)
	_, err := httpx.Get[httpx.Empty](ctx, url)
	if err == nil {
		return nil
	}
	var se *httpx.StatusError
	if errors.As(err, &se) {
		return nil
	}
	return err
}

func (Probe) Diagnose(ctx context.Context, env *core.Env, failure core.Result) []core.Artifact {
	dbArtifacts, deploymentID := dbdiag.New(env.DB).CaptureBySHA(ctx, failure.Dims["commit"])

	if deploymentID == "" {
		return dbArtifacts
	}

	kube, err := k8sdiag.New()
	if err != nil {
		return append(dbArtifacts, core.Artifact{
			Name:        "k8sdiag.txt",
			ContentType: "text/plain",
			Body:        []byte(fmt.Sprintf("k8sdiag unavailable: %v\n", err)),
		})
	}
	return append(dbArtifacts, kube.CaptureDeployment(ctx, deploymentID)...)
}

func validatePrereqs(env *core.Env) error {
	switch {
	case env.GitHubAppID == 0:
		return errors.New("env.GitHubAppID is zero; set PREFLIGHT_GITHUB_APP_ID")
	case env.GitHubInstallationID == 0:
		return errors.New("env.GitHubInstallationID is zero; set PREFLIGHT_GITHUB_INSTALLATION_ID")
	case env.GitHubPrivateKeyPEM == "":
		return errors.New("env.GitHubPrivateKeyPEM is empty; set PREFLIGHT_GITHUB_PRIVATE_KEY")
	case env.PreflightTestRepo == "":
		return errors.New("env.PreflightTestRepo is empty")
	case env.PreflightProjectSlug == "", env.PreflightAppSlug == "",
		env.PreflightWorkspaceSlug == "", env.PreflightApex == "":
		return errors.New("preflight slug fields (project/app/workspace/apex) are incomplete; set PREFLIGHT_PROJECT_SLUG, PREFLIGHT_APP_SLUG, PREFLIGHT_WORKSPACE_SLUG, PREFLIGHT_APEX")
	case env.Region == "":
		return errors.New("env.Region is empty")
	}

	return nil
}

// perCommitHostname MUST mirror svc/ctrl/worker/deploy/domains.go
// exactly. The prefix omits the app slug when it is "default", SHAs
// are truncated to 7 chars, git pushes get no random suffix. Any
// drift from ctrl's construction makes this probe poll a hostname
// that does not exist and time out.
func perCommitHostname(env *core.Env, sha string) string {
	prefix := env.PreflightProjectSlug
	if env.PreflightAppSlug != "default" {
		prefix = env.PreflightProjectSlug + "-" + env.PreflightAppSlug
	}
	return fmt.Sprintf("%s-git-%s-%s.%s",
		prefix,
		shortSHA(sha),
		env.PreflightWorkspaceSlug,
		env.PreflightApex,
	)
}

// metaReportsSHA swallows every error because pre-ready 4xx/5xx and
// DNS hiccups are expected during the deploy window; the outer
// deployTimeout is the one that enforces the SLA.
func metaReportsSHA(ctx context.Context, hostname, wantSHA string) (bool, error) {
	type meta struct {
		UnkeyGitCommitSHA string `json:"unkey_git_commit_sha"`
	}

	m, err := httpx.Get[meta](ctx, "https://"+hostname+"/meta")
	if err != nil {
		return false, nil //nolint:nilerr // transient during deploy window; deployTimeout owns the SLA
	}

	return m.UnkeyGitCommitSHA == wantSHA, nil
}

// shortSHA truncates to 7 to match ctrl's per-commit hostname
// construction; drift here breaks the hostname, not just error text.
func shortSHA(sha string) string {
	if len(sha) > 7 {
		return sha[:7]
	}

	return sha
}

func init() { probes.Register(Probe{}) }
