// Package krane_contract asserts what a pod exposes through /meta
// matches what a customer would expect Krane to have injected. Runs
// against the branch-sticky hostname that git_push just updated, so
// the target is deterministic from config alone (no MySQL lookup in
// the hot path).
//
// This probe is deliberately read-only over the customer-observable
// surface: hit /meta, check the UNKEY_* contract. Diagnose is the
// only place it touches MySQL, and that is best-effort bundle
// enrichment, not a primary assertion.
package krane_contract

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/unkeyed/unkey/pkg/httpx"
	"github.com/unkeyed/unkey/svc/preflight/core"
	"github.com/unkeyed/unkey/svc/preflight/internal/dbdiag"
	"github.com/unkeyed/unkey/svc/preflight/probes"
)

// branchPrefix MUST stay in sync with svc/preflight/probes/realinfra/
// git_push/git_push.go: that probe pushes to preflight/<region>, and
// we hit the branch-sticky hostname for the same branch to verify.
const branchPrefix = "preflight/"

type Probe struct{}

func (Probe) Name() string { return "krane_contract" }

type metaResponse struct {
	UnkeyDeploymentID      string `json:"unkey_deployment_id"`
	UnkeyGitCommitSHA      string `json:"unkey_git_commit_sha"`
	UnkeyEnvironmentSlug   string `json:"unkey_environment_slug"`
	UnkeyRegion            string `json:"unkey_region"`
	UnkeyInstanceID        string `json:"unkey_instance_id"`
	UnkeyEphemeralDiskPath string `json:"unkey_ephemeral_disk_path"`
	Port                   string `json:"port"`
	Protocol               string `json:"protocol"`
}

func (Probe) Run(ctx context.Context, env *core.Env) core.Result {
	if err := validatePrereqs(env); err != nil {
		return core.Fail(err)
	}

	hostname := branchHostname(env)
	dims := map[string]string{"hostname": hostname}

	fetchStart := time.Now()
	meta, err := httpx.Get[metaResponse](ctx, "https://"+hostname+"/meta")
	phases := []core.Phase{{Name: "fetch_meta", Duration: time.Since(fetchStart), Err: err}}
	if err != nil {
		return core.Failf("GET /meta on %s: %w", hostname, err).WithPhases(phases).WithDims(dims)
	}

	// Record the deployment ID we observed so Diagnose can key off it
	// without re-resolving anything.
	dims["deployment_id"] = meta.UnkeyDeploymentID

	if fail := assertContract(meta, env.Region); fail != "" {
		return core.Failf("%s", fail).WithPhases(phases).WithDims(dims)
	}

	return core.Pass().WithPhases(phases).WithDims(dims)
}

// Diagnose reaches into MySQL for deployment + step history. Primary
// assertions already ran against the customer surface; this only
// enriches the failure bundle.
func (Probe) Diagnose(ctx context.Context, env *core.Env, failure core.Result) []core.Artifact {
	return dbdiag.New(env.DB).CaptureByID(ctx, failure.Dims["deployment_id"])
}

func validatePrereqs(env *core.Env) error {
	switch {
	case env.PreflightProjectSlug == "", env.PreflightAppSlug == "",
		env.PreflightWorkspaceSlug == "", env.PreflightApex == "":
		return errors.New("preflight slug fields are incomplete")
	case env.Region == "":
		return errors.New("env.Region is empty")
	}

	return nil
}

// branchHostname builds the branch-sticky hostname ctrl writes for
// pushes to preflight/<region>. MUST mirror svc/ctrl/worker/deploy/
// domains.go: prefix omits "default" app slug, branch name is
// sluggified.
func branchHostname(env *core.Env) string {
	prefix := env.PreflightProjectSlug
	if env.PreflightAppSlug != "default" {
		prefix = env.PreflightProjectSlug + "-" + env.PreflightAppSlug
	}

	branch := sluggifyBranch(branchPrefix + env.Region)
	return fmt.Sprintf("%s-git-%s-%s.%s",
		prefix,
		branch,
		env.PreflightWorkspaceSlug,
		env.PreflightApex,
	)
}

// sluggifyBranch mirrors svc/ctrl/worker/deploy/domains.go:sluggify
// closely enough for our inputs (preflight/<region>, all ASCII).
// Replicated rather than imported because the probe MUST NOT depend
// on the control plane.
func sluggifyBranch(s string) string {
	s = strings.TrimSpace(strings.ToLower(s))
	var b strings.Builder
	b.Grow(len(s))
	lastWasSep := true

	for _, r := range s {
		alnum := (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9')
		switch {
		case alnum:
			b.WriteRune(r)
			lastWasSep = false
		case !lastWasSep:
			b.WriteByte('-')
			lastWasSep = true
		}
	}

	out := strings.TrimRight(b.String(), "-")
	if len(out) > 80 {
		out = strings.TrimRight(out[:80], "-")
	}

	return out
}

func assertContract(meta metaResponse, wantRegion string) string {
	switch {
	case meta.UnkeyDeploymentID == "":
		return "UNKEY_DEPLOYMENT_ID is empty; Krane should inject the deployment ID"
	case meta.UnkeyInstanceID == "":
		return "UNKEY_INSTANCE_ID is empty; Krane should inject a per-pod unique ID"
	case meta.UnkeyRegion != wantRegion:
		return fmt.Sprintf("UNKEY_REGION=%q; want %q (probe runner region)", meta.UnkeyRegion, wantRegion)
	case meta.UnkeyEnvironmentSlug == "":
		return "UNKEY_ENVIRONMENT_SLUG is empty"
	case meta.UnkeyGitCommitSHA == "":
		return "UNKEY_GIT_COMMIT_SHA is empty; expected git-source deployment from git_push"
	}

	return ""
}

func init() { probes.Register(Probe{}) }
