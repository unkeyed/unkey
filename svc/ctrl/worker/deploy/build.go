package deploy

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"maps"
	"net/http"
	"regexp"
	"slices"
	"sort"
	"strings"
	"time"

	"buf.build/gen/go/depot/api/connectrpc/go/depot/core/v1/corev1connect"
	corev1 "buf.build/gen/go/depot/api/protocolbuffers/go/depot/core/v1"
	"connectrpc.com/connect"
	"github.com/depot/depot-go/build"
	"github.com/depot/depot-go/machine"
	cliv1 "github.com/depot/depot-go/proto/depot/cli/v1"
	"github.com/docker/cli/cli/config/configfile"
	"github.com/docker/cli/cli/config/types"
	"github.com/moby/buildkit/client"
	"github.com/moby/buildkit/session"
	"github.com/moby/buildkit/session/auth/authprovider"
	"github.com/moby/buildkit/session/secrets/secretsprovider"
	"github.com/opencontainers/go-digest"
	restate "github.com/restatedev/sdk-go"

	"github.com/unkeyed/unkey/pkg/assert"
	"github.com/unkeyed/unkey/pkg/clickhouse/schema"
	"github.com/unkeyed/unkey/pkg/logger"
	"github.com/unkeyed/unkey/pkg/ptr"
	"github.com/unkeyed/unkey/pkg/validation"
	"github.com/unkeyed/unkey/svc/ctrl/internal/db"
	githubclient "github.com/unkeyed/unkey/svc/ctrl/worker/github"
)

const (
	// defaultCacheKeepGB is the maximum cache size in gigabytes for new Depot
	// projects. Depot evicts least-recently-used cache entries when exceeded.
	defaultCacheKeepGB = 25

	// defaultCacheKeepDays is the maximum age in days for cached build layers.
	// Layers older than this are evicted regardless of cache size.
	defaultCacheKeepDays = 7

	// gitAuthTokenSecretID is the BuildKit session secret holding the GitHub
	// installation token for git context fetches. The host suffix scopes the
	// token to github.com; BuildKit's git source looks up the host-suffixed
	// name first. Shared by the Dockerfile and Railpack build paths.
	gitAuthTokenSecretID = "GIT_AUTH_TOKEN.github.com"
)

// knownBuildError maps a BuildKit error pattern to a user-friendly message.
type knownBuildError struct {
	// substr is matched anywhere in the error string.
	substr string
	// message is the clean, actionable text shown to the user.
	message string
}

// knownBuildErrors lists BuildKit error patterns caused by user mistakes
// (bad Dockerfile, missing files, invalid config) paired with friendly messages.
var knownBuildErrors = []knownBuildError{
	// Railpack (Dockerfile-less) builds. Listed first because the
	// "railpack prepare failed" wrapper is more specific than the generic
	// patterns below (e.g. "no such file or directory") that could otherwise
	// shadow it.
	// The message is deliberately tech-neutral: Railpack is an implementation
	// detail we don't surface to customers. "check the root directory" also
	// makes the dashboard's failed-deployment banner show its settings link.
	{substr: "railpack prepare failed", message: "Unkey could not build this app automatically. For a monorepo, set the root directory to your app and a custom build command in settings, or review the build logs for details."},
	// Settings-fixable: dockerfile path / docker context
	{substr: "the dockerfile cannot be empty", message: "The Dockerfile appears to be empty. Please verify the file path in settings."},
	{substr: "failed to read dockerfile", message: "Dockerfile could not be read. Please check that the file path is correct in settings."},
	{substr: "failed to find target", message: "The specified build target stage was not found. Please check the target name in settings."},
	{substr: "failed to compute cache key", message: "A file referenced in the Dockerfile was not found. Please check the root directory in settings."},
	{substr: "no such file or directory", message: "A file or directory referenced in the build was not found. Please check the root directory in settings."},
	// Dockerfile content issues (require editing the Dockerfile)
	{substr: "dockerfile parse error on line", message: "Dockerfile has a syntax error. Please check the Dockerfile for typos."},
	{substr: "no build stage in current context", message: "Dockerfile has no valid build stage. Please add a FROM instruction."},
	{substr: "circular dependency detected on stage", message: "Dockerfile contains a circular dependency between stages. Please review the stage references."},
	{substr: "invalid reference format", message: "A Docker image reference is invalid. Please check your FROM lines."},
	{substr: "no match for platform in manifest", message: "The base image does not support the target platform. Please use a multi-platform image or change the platform."},
	{substr: "no matching manifest", message: "The base image does not support the target platform. Please use a multi-platform image or change the platform."},
	{substr: "there is no variable named", message: "Dockerfile references an undefined variable. Please check your ARG declarations."},
	{substr: "the expression result is null", message: "A Dockerfile expression evaluated to null. Please check your variable references."},
	{substr: "invalid expression", message: "Dockerfile contains an invalid expression. Please check the syntax."},
	{substr: "invalid block definition", message: "Dockerfile contains an invalid block definition. Please check the syntax."},
	{substr: "must be of the form: name=value", message: "A build argument is malformed. Please use the format name=value."},
	{substr: "contains value with non-printable ascii characters", message: "A build argument contains non-printable characters. Please remove them."},
	{substr: "docker exporter does not currently support", message: "The requested export format is not supported. Please check your export configuration."},
	// Generic fallbacks
	{substr: "failed to solve: process", message: "A build command failed. Please check the build logs for details."},
	{substr: "linting failed", message: "Dockerfile linting failed. Please check the Dockerfile for issues."},
}

// repoFullNameRegex matches a GitHub "owner/repo" full name. Mirrors the
// dashboard's REPO_FULL_NAME guard (resolve-deploy-ref.ts) so untrusted inputs
// can't smuggle a URL fragment or path traversal into the git context URL.
var repoFullNameRegex = regexp.MustCompile(`^[A-Za-z0-9._-]+/[A-Za-z0-9._-]+$`)

// commitSHARegex matches a hex git object name (7-40 chars). Anything outside
// this set (':', '#', '/', '..') could alter what BuildKit checks out once
// interpolated into the git context URL's ref/subdir fragment.
var commitSHARegex = regexp.MustCompile(`^[0-9a-fA-F]{7,40}$`)

// validateGitBuildParams rejects untrusted git inputs before they reach the
// BuildKit context URL. The dashboard validates these in TS, but the public v2
// API accepts commitSha as a free string and fillFromGitHub can skip the
// GitHub round-trip, so a malicious SHA or repo name would otherwise reach
// buildGitContextURL unchecked. Validating here covers the dashboard, webhook,
// rebuild-from-DB, and API paths at once.
func validateGitBuildParams(params gitBuildParams) error {
	if !repoFullNameRegex.MatchString(params.Repository) {
		return fmt.Errorf("invalid repository %q: must be in owner/repo form", params.Repository)
	}
	if params.ForkRepository != "" && !repoFullNameRegex.MatchString(params.ForkRepository) {
		return fmt.Errorf("invalid fork repository %q: must be in owner/repo form", params.ForkRepository)
	}
	// SHA is unused when a PR ref drives the build (refs/pull/<n>/head), so only
	// validate a non-empty SHA.
	if params.CommitSHA != "" && !commitSHARegex.MatchString(params.CommitSHA) {
		return fmt.Errorf("invalid commit SHA %q: must be a hex git object name", params.CommitSHA)
	}
	return nil
}

// buildResult contains the output of a Docker image build, including the image
// name and identifiers needed to trace builds in Depot.
type buildResult struct {
	ImageName      string
	DepotBuildID   string
	DepotProjectID string
}

// gitBuildParams holds the inputs for building a container image from a Git
// repository, including the exact commit and the build context location.
type gitBuildParams struct {
	InstallationID int64
	Repository     string
	ForkRepository string
	CommitSHA      string
	ContextPath    string
	DockerfilePath string
	// BuildCommand overrides Railpack's auto-detected build command
	// (RAILPACK_BUILD_CMD) so monorepos can scope the build to a single app.
	// Empty means auto-detect. Only consumed by the Railpack build path;
	// Dockerfile builds ignore it.
	BuildCommand                  string
	ProjectID                     string
	AppID                         string
	DeploymentID                  string
	WorkspaceID                   string
	PrNumber                      int64
	EncryptedEnvironmentVariables []byte
	EnvironmentID                 string
}

// gitBuildContext carries the inputs that the shared git-build scaffold
// resolves identically for every build method: Dockerfile and Railpack
// builders receive it and only implement what differentiates them.
type gitBuildContext struct {
	DepotProjectID string

	// GithubToken is empty when the clone needs no credential: unauthenticated
	// mode (local dev) or a fork build of a public repo. See resolveCloneToken.
	GithubToken string

	// EnvVars are the decrypted environment variables for the deployment.
	EnvVars map[string]string

	// GitContextURL is the BuildKit git context the build machine fetches.
	GitContextURL string

	// ImageName is the fully qualified registry tag to push.
	ImageName string
}

// runGitBuild wraps the lifecycle shared by every git-based image build:
// validating the untrusted git params, resolving the Depot project and the
// least-privilege clone credential, decrypting env vars, and invoking the
// method-specific buildFn inside a single durable Run with the standard
// retry bounds.
//
// buildFn executes entirely within one Run attempt on one process. It must
// keep all process-local state (temp files, BuildKit connections) inside its
// own invocation, because a retry may resume on a different pod.
func (w *Workflow) runGitBuild(
	ctx restate.Context,
	runName string,
	params gitBuildParams,
	buildFn func(runCtx restate.RunContext, bctx gitBuildContext) (*buildResult, error),
) (*buildResult, error) {
	if err := validateGitBuildParams(params); err != nil {
		return nil, restate.TerminalError(fmt.Errorf("invalid git build params: %w", err))
	}

	depotProjectID, err := restate.Run(ctx, func(runCtx restate.RunContext) (string, error) {
		return w.getOrCreateDepotProject(runCtx, params.ProjectID)
	}, restate.WithName("get or create depot project"), restate.WithMaxRetryAttempts(runMaxAttempts))
	if err != nil {
		return nil, fmt.Errorf("failed to get/create depot project: %w", err)
	}

	return restate.Run(ctx, func(runCtx restate.RunContext) (*buildResult, error) {
		// A fork build runs build instructions we do not trust. PrNumber > 0
		// is a live fork PR (clones the base via refs/pull/<n>/head);
		// ForkRepository != "" is a dashboard deploy of a fork ref by SHA
		// (clones the fork directly).
		isForkBuild := params.PrNumber > 0 || params.ForkRepository != ""

		// tokenless means no GIT_AUTH_TOKEN secret is registered, so
		// fork-controlled build instructions cannot mount a GitHub credential.
		// Env vars are still injected as the env secret on both paths.
		ghToken, tokenless, err := w.resolveCloneToken(params, isForkBuild)
		if err != nil {
			return nil, err
		}
		githubToken := ""
		if !tokenless {
			githubToken = ghToken.Token
		}

		// Decrypt env vars in-memory so they can be injected as BuildKit secrets.
		// Treat all decryption failures as terminal: bearer-token / keyring
		// config errors never self-heal, and genuine vault outages are better
		// surfaced to the user fast than burned inside a retry loop.
		envVars, err := w.decryptEnvVars(runCtx, params.EncryptedEnvironmentVariables, params.EnvironmentID)
		if err != nil {
			return nil, restate.TerminalError(fmt.Errorf("failed to decrypt env vars for build: %w", err))
		}

		if err := validateShellEnvKeys(envVars); err != nil {
			return nil, restate.TerminalError(err)
		}

		return buildFn(runCtx, gitBuildContext{
			DepotProjectID: depotProjectID,
			GithubToken:    githubToken,
			EnvVars:        envVars,
			GitContextURL:  buildGitContextURL(params),
			ImageName:      fmt.Sprintf("%s:%s-%s", w.registryConfig.Repository, params.ProjectID, params.DeploymentID),
		})
	}, restate.WithName(runName),
		// Bound retries both by count (for transient Depot/BuildKit blips) and
		// by wall-clock (a single build attempt is long-running, so 5 attempts
		// × worst-case backoff could otherwise exceed any reasonable ceiling).
		// Whichever bound fires first wins.
		restate.WithMaxRetryAttempts(runMaxAttempts),
		restate.WithMaxRetryDuration(buildImageRetryCeiling))
}

// buildDockerImageFromGit builds a container image from a GitHub repository using Depot.
//
// The method retrieves or creates a Depot project for the Unkey project,
// acquires a remote build machine, and executes the build. BuildKit fetches
// the repository directly from GitHub using the provided installation token.
// Build progress is streamed to ClickHouse for observability.
func (w *Workflow) buildDockerImageFromGit(
	ctx restate.Context,
	params gitBuildParams,
) (*buildResult, error) {
	platform := w.buildPlatform.Platform

	logger.Info("Starting git build process",
		"repository", params.Repository,
		"commit_sha", params.CommitSHA,
		"project_id", params.ProjectID,
		"platform", platform,
		"architecture", w.buildPlatform.Architecture)

	return w.runGitBuild(ctx, "build docker image from git", params, func(runCtx restate.RunContext, bctx gitBuildContext) (*buildResult, error) {
		// An empty dockerfile path must never reach this builder: buildImage
		// routes those deployments to Railpack. Passing an empty "filename"
		// to BuildKit would silently fall back to "Dockerfile", masking a
		// routing bug, so assert instead.
		dockerfilePath := params.DockerfilePath
		if assertErr := assert.NotEmpty(dockerfilePath, "dockerfile path must be set for dockerfile builds"); assertErr != nil {
			return nil, restate.TerminalError(assertErr)
		}

		logger.Info("Starting build execution",
			"image_name", bctx.ImageName,
			"dockerfile", dockerfilePath,
			"platform", platform,
			"project_id", params.ProjectID,
			"git_context_url", bctx.GitContextURL,
		)

		// Without a token (unauthenticated mode, or a fork build of a public
		// repo) the git fetch needs no credentials, so skip the auth secret
		// entirely.
		var solverOptions client.SolveOpt
		var err error
		if bctx.GithubToken == "" {
			solverOptions, err = w.buildSolverOptions(platform, bctx.GitContextURL, dockerfilePath, bctx.ImageName, bctx.EnvVars)
		} else {
			solverOptions, err = w.buildGitSolverOptions(platform, bctx.GitContextURL, dockerfilePath, bctx.ImageName, bctx.GithubToken, bctx.EnvVars)
		}
		if err != nil {
			return nil, restate.TerminalError(fmt.Errorf("failed to build solver options: %w", err))
		}

		return w.solveOnDepotMachine(runCtx, bctx.DepotProjectID, bctx.ImageName, params, solverOptions)
	})
}

// validateShellEnvKeys rejects env var names that are not valid environment
// variable names. Creation already enforces this at the API boundary
// (envVarKeySchema in the dashboard, [validation.IsValidEnvVarKey] in Go);
// this is the paired assertion before use, catching variables that were
// created before the rule was tightened.
func validateShellEnvKeys(envVars map[string]string) error {
	for _, key := range slices.Sorted(maps.Keys(envVars)) {
		if !validation.IsValidEnvVarKey(key) {
			return fmt.Errorf("environment variable %q cannot be used during builds: %s", key, validation.ErrMsgInvalidEnvVarKey)
		}
	}
	return nil
}

// buildGitContextURL builds the BuildKit git context URL for a deployment:
// https://github.com/owner/repo.git#<ref>[:<subdir>]. BuildKit fetches the
// repository directly on the build machine, so no repository content ever
// passes through the worker. Shared by the Dockerfile and Railpack paths so
// both fetch identical sources.
//
// For a fork PR (PrNumber > 0) the ref is refs/pull/<n>/head fetched from the
// BASE repo: GitHub exposes that ref only on the base, never on the fork.
// Only a concrete fork commit SHA (a dashboard deploy of a fork ref, where
// PrNumber is 0) is fetched directly from ForkRepository. The context path is
// normalized: leading slashes are stripped and "." means the repository root.
func buildGitContextURL(params gitBuildParams) string {
	contextPath := strings.TrimSpace(params.ContextPath)
	contextPath = strings.TrimPrefix(contextPath, "/")
	if contextPath == "." {
		contextPath = ""
	}

	ref := params.CommitSHA
	if params.PrNumber > 0 {
		ref = fmt.Sprintf("refs/pull/%d/head", params.PrNumber)
	}
	buildRepo := cloneRepoFor(params.Repository, params.ForkRepository, params.PrNumber)

	if contextPath != "" {
		return fmt.Sprintf("https://github.com/%s.git#%s:%s", buildRepo, ref, contextPath)
	}
	return fmt.Sprintf("https://github.com/%s.git#%s", buildRepo, ref)
}

// withDepotBuildkit creates a Depot build, acquires a remote BuildKit
// machine, connects, and invokes fn with the connected client. The Depot
// build is finalized and the machine released regardless of fn's outcome.
// Returns the Depot build ID alongside fn's error.
func (w *Workflow) withDepotBuildkit(
	runCtx context.Context,
	depotProjectID string,
	params gitBuildParams,
	fn func(buildClient *client.Client) error,
) (_ string, err error) {
	depotBuild, err := build.NewBuild(runCtx, &cliv1.CreateBuildRequest{
		Options:   nil,
		ProjectId: depotProjectID,
	}, w.registryConfig.Password)
	if err != nil {
		return "", fmt.Errorf("failed to create build: %w", err)
	}
	defer func() { depotBuild.Finish(err) }()

	logger.Info("Depot build created",
		"build_id", depotBuild.ID,
		"depot_project_id", depotProjectID,
		"project_id", params.ProjectID)

	logger.Info("Acquiring build machine",
		"build_id", depotBuild.ID,
		"architecture", w.buildPlatform.Architecture,
		"project_id", params.ProjectID)

	buildkit, err := machine.Acquire(runCtx, depotBuild.ID, depotBuild.Token, w.buildPlatform.Architecture)
	if err != nil {
		return "", fmt.Errorf("failed to acquire machine: %w", err)
	}
	defer func() {
		if releaseErr := buildkit.Release(); releaseErr != nil {
			logger.Error("unable to release buildkit", "error", releaseErr)
		}
	}()

	logger.Info("Build machine acquired, connecting to buildkit",
		"build_id", depotBuild.ID,
		"project_id", params.ProjectID)

	buildClient, err := buildkit.Connect(runCtx)
	if err != nil {
		return "", fmt.Errorf("unable to create build client: %w", err)
	}
	defer func() {
		if closeErr := buildClient.Close(); closeErr != nil {
			logger.Error("unable to close client", "error", closeErr)
		}
	}()

	err = fn(buildClient)
	return depotBuild.ID, err
}

// solveWithStatus executes one solve on the given client, streaming build
// status to ClickHouse so steps and logs show up in the dashboard.
//
// Returns a terminal error for build failures that cannot be retried; context
// cancellations, timeouts, and transient registry errors are returned as
// retryable errors.
func (w *Workflow) solveWithStatus(
	runCtx context.Context,
	buildClient *client.Client,
	params gitBuildParams,
	solverOptions client.SolveOpt,
) error {
	buildStatusCh := make(chan *client.SolveStatus, 100)
	go w.processBuildStatus(buildStatusCh, params.WorkspaceID, params.ProjectID, params.DeploymentID)

	_, err := buildClient.Solve(runCtx, nil, solverOptions, buildStatusCh)
	if err != nil {
		// Context cancellations and timeouts are transient — let Restate retry.
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			return fmt.Errorf("build interrupted: %w", err)
		}
		if isTransientSolveError(err) {
			return fmt.Errorf("build hit a transient registry error: %w", err)
		}
		return restate.TerminalError(fmt.Errorf("build failed: %w", err))
	}
	return nil
}

// transientSolveErrorSubstrings are status phrases registries return on
// server-side failures, e.g. a 502 from ghcr.io while the build machine pulls
// an image layer. Matched conservatively: user build failures surface as
// process exit errors and never contain these phrases.
var transientSolveErrorSubstrings = []string{
	"500 Internal Server Error",
	"502 Bad Gateway",
	"503 Service Unavailable",
	"504 Gateway Timeout",
}

// isTransientSolveError reports whether a solve failure looks like a
// transient registry blip worth retrying — the build itself never ran, so a
// retry is safe and likely to succeed.
func isTransientSolveError(err error) bool {
	msg := err.Error()
	for _, substr := range transientSolveErrorSubstrings {
		if strings.Contains(msg, substr) {
			return true
		}
	}
	return false
}

// solveOnDepotMachine runs a single solve on a freshly acquired Depot
// machine. Used by the Dockerfile build path; the Railpack path composes
// [Workflow.withDepotBuildkit] and [Workflow.solveWithStatus] directly since
// it performs two solves on one machine.
func (w *Workflow) solveOnDepotMachine(
	runCtx context.Context,
	depotProjectID string,
	imageName string,
	params gitBuildParams,
	solverOptions client.SolveOpt,
) (*buildResult, error) {
	depotBuildID, err := w.withDepotBuildkit(runCtx, depotProjectID, params, func(buildClient *client.Client) error {
		return w.solveWithStatus(runCtx, buildClient, params, solverOptions)
	})
	if err != nil {
		return nil, err
	}

	logger.Info("Build completed successfully")

	return &buildResult{
		ImageName:      imageName,
		DepotBuildID:   depotBuildID,
		DepotProjectID: depotProjectID,
	}, nil
}

// registryAuthProvider returns a session attachable that authenticates image
// pushes to the configured container registry.
func (w *Workflow) registryAuthProvider() session.Attachable {
	//nolint: exhaustruct
	return authprovider.NewDockerAuthProvider(authprovider.DockerAuthProviderConfig{
		ConfigFile: &configfile.ConfigFile{
			AuthConfigs: map[string]types.AuthConfig{
				w.registryConfig.Repository: {
					Username: w.registryConfig.Username,
					Password: w.registryConfig.Password,
				},
			},
		},
	})
}

// cloneRepoFor returns the repository BuildKit actually clones. A live fork PR
// (prNumber > 0) clones the BASE repo via refs/pull/<n>/head; a fork ref
// deployed by concrete SHA (prNumber == 0) clones the fork itself. Token
// scoping and visibility probing must target this repo, not blindly the base.
func cloneRepoFor(repository, forkRepository string, prNumber int64) string {
	if prNumber == 0 && forkRepository != "" {
		return forkRepository
	}
	return repository
}

// repoOwner returns the owner segment of an "owner/repo" full name.
func repoOwner(fullName string) string {
	owner, _, _ := strings.Cut(fullName, "/")
	return owner
}

// buildEnvFileSecret serializes env vars into a .env-formatted byte slice
// for injection as a BuildKit secret. Returns (nil, nil) if there are no env
// vars. Returns an error if any value contains newline or carriage-return
// characters, which would corrupt the .env line format.
func buildEnvFileSecret(envVars map[string]string) ([]byte, error) {
	if len(envVars) == 0 {
		return nil, nil
	}

	var badKeys []string
	for k, v := range envVars {
		if strings.ContainsAny(v, "\n\r") {
			badKeys = append(badKeys, k)
		}
	}
	if len(badKeys) != 0 {
		sort.Strings(badKeys)
		return nil, fmt.Errorf("environment variables contain newlines which cannot be represented in .env format: %s", strings.Join(badKeys, ", "))
	}

	var buf strings.Builder
	for k, v := range envVars {
		buf.WriteString(k)
		buf.WriteByte('=')
		buf.WriteString(v)
		buf.WriteByte('\n')
	}
	return []byte(buf.String()), nil
}

// hashEnvVars returns a hex-encoded SHA-256 hash of the sorted key=value pairs.
// The hash is stable across runs and safe to embed in image metadata without
// exposing secret values. Returns an empty string if there are no env vars.
func hashEnvVars(envVars map[string]string) string {
	if len(envVars) == 0 {
		return ""
	}
	pairs := make([]string, 0, len(envVars))
	for k, v := range envVars {
		pairs = append(pairs, k+"="+v)
	}
	sort.Strings(pairs)
	h := sha256.Sum256([]byte(strings.Join(pairs, "\n")))
	return hex.EncodeToString(h[:])
}

// buildSolverOptions constructs the BuildKit solver configuration for URL-based
// contexts, including registry auth and image export settings. Use
// [Workflow.buildGitSolverOptions] when the context requires GitHub credentials.
func (w *Workflow) buildSolverOptions(
	platform, contextURL, dockerfilePath, imageName string,
	envVars map[string]string,
) (client.SolveOpt, error) {
	sessionAttachables := []session.Attachable{
		w.registryAuthProvider(),
	}

	envFile, err := buildEnvFileSecret(envVars)
	if err != nil {
		return client.SolveOpt{}, fmt.Errorf("invalid environment variables: %w", err)
	}

	frontendAttrs := map[string]string{
		"platform": platform,
		"context":  contextURL,
		"filename": dockerfilePath,
	}
	if envFile != nil {
		// Publish the same content under the stable "env" id (legacy) and the
		// env-hash as a second id. Dockerfiles that reference
		// id=${UNKEY_SECRETS_ID} see the mount declaration change when env
		// content changes, which is what invalidates BuildKit's RUN cache
		// key. id=env stays available so unmigrated Dockerfiles keep building
		// during the rollout.
		h := hashEnvVars(envVars)
		sessionAttachables = append(sessionAttachables, secretsprovider.FromMap(map[string][]byte{
			"env": envFile,
			h:     envFile,
		}))
		frontendAttrs["label:org.unkey.env-hash"] = h
		frontendAttrs["build-arg:UNKEY_SECRETS_ID"] = h
	}

	return client.SolveOpt{
		Frontend:      "dockerfile.v0",
		FrontendAttrs: frontendAttrs,
		Session:       sessionAttachables,
		//nolint: exhaustruct
		Exports: []client.ExportEntry{
			{
				Type: "image",
				Attrs: map[string]string{
					"name":           imageName,
					"oci-mediatypes": "true",
					"push":           "true",
				},
			},
		},
	}, nil
}

// buildGitSolverOptions constructs the buildkit solver configuration for a git context build.
// It includes GitHub token authentication via the secrets provider.
func (w *Workflow) buildGitSolverOptions(
	platform, gitContextURL, dockerfilePath, imageName, githubToken string,
	envVars map[string]string,
) (client.SolveOpt, error) {
	secrets := map[string][]byte{
		gitAuthTokenSecretID: []byte(githubToken),
	}
	envFile, err := buildEnvFileSecret(envVars)
	if err != nil {
		return client.SolveOpt{}, fmt.Errorf("invalid environment variables: %w", err)
	}

	frontendAttrs := map[string]string{
		"platform": platform,
		"context":  gitContextURL,
		"filename": dockerfilePath,
	}
	if envFile != nil {
		// See buildSolverOptions for the dual-id rationale: legacy "env" keeps
		// unmigrated Dockerfiles working, the hash-id lets the mount
		// declaration vary with env content so BuildKit invalidates the
		// secret-consuming RUN when a variable changes.
		h := hashEnvVars(envVars)
		secrets["env"] = envFile
		secrets[h] = envFile
		frontendAttrs["label:org.unkey.env-hash"] = h
		frontendAttrs["build-arg:UNKEY_SECRETS_ID"] = h
	}

	return client.SolveOpt{
		Frontend:      "dockerfile.v0",
		FrontendAttrs: frontendAttrs,

		Session: []session.Attachable{
			w.registryAuthProvider(),
			secretsprovider.FromMap(secrets),
		},
		//nolint: exhaustruct
		Exports: []client.ExportEntry{
			{
				Type: "image",
				Attrs: map[string]string{
					"name":           imageName,
					"oci-mediatypes": "true",
					"push":           "true",
				},
			},
		},
	}, nil
}

// getOrCreateDepotProject retrieves the Depot project ID for an Unkey project,
// creating one if it doesn't exist.
func (w *Workflow) getOrCreateDepotProject(ctx context.Context, unkeyProjectID string) (string, error) {
	project, err := w.db.FindProjectById(ctx, unkeyProjectID)
	if err != nil {
		return "", fmt.Errorf("failed to query project: %w", err)
	}

	projectName := fmt.Sprintf("%s-%s", w.depotConfig.ProjectPrefix, unkeyProjectID)
	if project.DepotProjectID.Valid && project.DepotProjectID.String != "" {
		logger.Info(
			"Returning existing depot project",
			"depot_project_id", project.DepotProjectID,
			"project_id", unkeyProjectID,
			"project_name", projectName,
		)
		return project.DepotProjectID.String, nil
	}

	httpClient := &http.Client{}
	authInterceptor := connect.UnaryInterceptorFunc(func(next connect.UnaryFunc) connect.UnaryFunc {
		return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
			req.Header().Set("Authorization", "Bearer "+w.registryConfig.Password)
			return next(ctx, req)
		}
	})

	projectClient := corev1connect.NewProjectServiceClient(httpClient, w.depotConfig.APIUrl, connect.WithInterceptors(authInterceptor))
	//nolint: exhaustruct // optional fields
	createResp, err := projectClient.CreateProject(ctx, connect.NewRequest(&corev1.CreateProjectRequest{
		Name:     projectName,
		RegionId: w.depotConfig.ProjectRegion,
		//nolint: exhaustruct // missing fields is deprecated
		CachePolicy: &corev1.CachePolicy{
			KeepGb:   defaultCacheKeepGB,
			KeepDays: defaultCacheKeepDays,
		},
	}))
	if err != nil {
		return "", fmt.Errorf("failed to create project: %w", err)
	}
	depotProjectID := createResp.Msg.GetProject().GetProjectId()

	now := time.Now().UnixMilli()
	err = w.db.UpdateProjectDepotID(ctx, db.UpdateProjectDepotIDParams{
		DepotProjectID: sql.NullString{
			String: depotProjectID,
			Valid:  true,
		},
		UpdatedAt: sql.NullInt64{Int64: now, Valid: true},
		ID:        unkeyProjectID,
	})
	if err != nil {
		return "", fmt.Errorf("failed to update depot_project_id: %w", err)
	}

	logger.Info("Created new Depot project",
		"depot_project_id", depotProjectID,
		"project_id", unkeyProjectID,
		"project_name", projectName)

	return depotProjectID, nil
}

// processBuildStatus consumes build status events from buildkit and writes
// telemetry to ClickHouse.
func (w *Workflow) processBuildStatus(
	statusCh <-chan *client.SolveStatus,
	workspaceID, projectID, deploymentID string,
) {
	completed := map[digest.Digest]bool{}
	verticesWithLogs := map[digest.Digest]bool{}

	for status := range statusCh {
		for _, log := range status.Logs {
			verticesWithLogs[log.Vertex] = true
		}

		for _, vertex := range status.Vertexes {
			if vertex == nil {
				logger.Warn("vertex is nil")
				continue
			}
			if vertex.Completed != nil && !completed[vertex.Digest] {
				completed[vertex.Digest] = true

				w.buildSteps.Buffer(schema.BuildStepV1{
					Error:        vertex.Error,
					StartedAt:    ptr.SafeDeref(vertex.Started).UnixMilli(),
					CompletedAt:  ptr.SafeDeref(vertex.Completed).UnixMilli(),
					WorkspaceID:  workspaceID,
					ProjectID:    projectID,
					DeploymentID: deploymentID,
					StepID:       vertex.Digest.String(),
					Name:         vertex.Name,
					Cached:       vertex.Cached,
					HasLogs:      verticesWithLogs[vertex.Digest],
				})
			}
		}

		for _, log := range status.Logs {
			w.buildStepLogs.Buffer(schema.BuildStepLogV1{
				WorkspaceID:  workspaceID,
				ProjectID:    projectID,
				DeploymentID: deploymentID,
				StepID:       log.Vertex.String(),
				Time:         log.Timestamp.UnixMilli(),
				Message:      string(log.Data),
			})
		}
	}
}

// resolveCloneToken decides which credential BuildKit gets for cloning the
// source, least-privilege in three tiers:
//
//   - unauthenticated public deploy: no token at all (tokenless).
//   - fork build (untrusted Dockerfile): a public clone target needs no token;
//     a private one gets a read-only token scoped to that single repo, so an
//     exfiltrated token reads only what the PR author already can.
//   - trusted build: a read-only token, also scoped to the single repo being
//     cloned. A Dockerfile pulling private cross-repo deps or submodules will
//     see a clone 404; widen the scope only if that surfaces in practice.
//
// Probing and scoping both target the repo BuildKit actually clones: the base
// for a live fork PR (refs/pull/<n>/head only exists there), the fork itself
// for a fork ref deployed by SHA. A private fork outside the base's
// installation is unreachable by any token this installation can mint, so it
// fails fast with a terminal error instead of a confusing git failure
// mid-build. An inconclusive visibility probe never falls back to the
// tokenless path.
func (w *Workflow) resolveCloneToken(params gitBuildParams, isForkBuild bool) (githubclient.InstallationToken, bool, error) {
	var noToken githubclient.InstallationToken

	if w.allowUnauthenticatedDeployments && params.InstallationID == noInstallationID {
		logger.Info("Unauthenticated mode: skipping GitHub authentication for public repo",
			"repository", params.Repository)
		return noToken, true, nil
	}

	scopeRepo := cloneRepoFor(params.Repository, params.ForkRepository, params.PrNumber)
	if isForkBuild {
		// Same owner as the base means the fork lives in the same GitHub App
		// installation (installations cover one account), so a token can be
		// scoped to it. A different owner is in another installation we cannot
		// mint a token for at all. Only meaningful for the PrNumber == 0
		// fork-ref-by-SHA case: a live PR clones the base (scopeRepo ==
		// params.Repository), so this is always false there.
		//
		// EqualFold because GitHub owners are case-insensitive: ForkRepository
		// arrives verbatim from dashboard user typing while params.Repository
		// carries GitHub's canonical casing, so a casing drift (Acme vs acme)
		// must not be misread as a different owner.
		differentOwner := !strings.EqualFold(repoOwner(scopeRepo), repoOwner(params.Repository))

		public, err := w.github.IsRepoPublic(scopeRepo)

		// Public fork clones anonymously: no token to mint or expose.
		if err == nil && public {
			return noToken, true, nil
		}

		// Different-owner fork lives in another installation; no token we mint
		// reaches it, so fail now rather than hand out a useless one.
		if differentOwner {
			if err != nil {
				// Visibility unknown (e.g. rate limit). Retryable, quota self-heals.
				return noToken, false, fmt.Errorf("could not determine visibility of fork repository %s: %w", scopeRepo, err)
			}
			// Confirmed private: terminal, retrying never helps.
			return noToken, false, restate.TerminalError(fmt.Errorf(
				"cannot access private fork repository %s: it is outside this GitHub App installation", scopeRepo,
			))
		}

		// Same-owner fork, private or visibility-unknown: fall through to mint a
		// scoped read-only token below.
		if err != nil {
			logger.Warn("repo visibility check failed; using scoped read-only token",
				"repository", scopeRepo, "error", err)
		}
	}

	// Permission names and access levels are defined by GitHub:
	// https://docs.github.com/en/rest/authentication/permissions-required-for-github-apps
	// "contents":"read" grants clone/read access only, no write of any kind.
	token, err := w.github.GetScopedInstallationToken(params.InstallationID, scopeRepo, map[string]string{"contents": "read"})
	if err != nil {
		return noToken, false, fmt.Errorf("failed to get GitHub installation token: %w", err)
	}
	return token, false, nil
}

// extractUserBuildError checks whether err matches a known user-caused build
// failure and returns a clean, actionable message. For unrecognized errors it
// returns a generic fallback.
func extractUserBuildError(err error) string {
	msg := strings.ToLower(err.Error())
	for _, known := range knownBuildErrors {
		if strings.Contains(msg, known.substr) {
			return known.message
		}
	}
	return "Build failed. Please check the build logs for details."
}
