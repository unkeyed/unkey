package deploy

import (
	"encoding/json"
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"text/template"

	"github.com/moby/buildkit/client"
	"github.com/moby/buildkit/session"
	"github.com/moby/buildkit/session/secrets/secretsprovider"
	restate "github.com/restatedev/sdk-go"
	"github.com/tonistiigi/fsutil"

	"github.com/unkeyed/unkey/pkg/logger"
	githubclient "github.com/unkeyed/unkey/svc/ctrl/worker/github"
)

const (
	// railpackFrontendImage is the pinned Railpack BuildKit frontend image.
	// It provides the /railpack binary for plan generation and acts as the
	// gateway frontend for the image build.
	//
	// This is our fork build (github.com/unkeyed/railpack, branch
	// unkey/git-context): upstream's frontend can only read the build context
	// from a session local mount, which would force the worker to download
	// and stream untrusted repository content. The fork adds git context
	// support so the build machine fetches the repository itself. Generated
	// plans are only guaranteed to be understood by the same frontend
	// version, which is why this is a constant rather than configuration —
	// bump it together with code changes, not at deploy time.
	railpackFrontendImage = "ghcr.io/unkeyed/railpack-frontend:v0.27.0-unkey.1"

	// railpackBuilderImage is the image the prepare step runs in. It must be
	// glibc-based: railpack downloads a gnu-libc mise binary at plan time to
	// resolve package versions, which cannot exec on the musl-based frontend
	// image. This is the same builder image the generated plan's own steps
	// run in, so the build machine pulls it regardless. The mise version in
	// the tag is pinned by the railpack release (core/mise/version.txt) —
	// keep it in sync when bumping railpackFrontendImage.
	railpackBuilderImage = "ghcr.io/railwayapp/railpack-builder:mise-2026.6.1"

	// railpackPlanFilename is the build plan filename the Railpack BuildKit
	// frontend expects inside the "dockerfile" local mount.
	railpackPlanFilename = "railpack-plan.json"

	// railpackPlanBytesMax caps the plan file returned from the prepare solve.
	// Plans are small JSON documents; anything larger indicates a bug or abuse.
	railpackPlanBytesMax = 16 << 20 // 16 MiB

	// railpackWorkspacePattern names the temp directories holding the
	// generated prepare Dockerfile and the returned plan. Shared between
	// workspace creation and the stale-workspace sweep.
	railpackWorkspacePattern = "railpack-build-*"
)

// cleanupStaleRailpackWorkspaces removes build workspaces left behind by a
// previous process that crashed before its deferred cleanup could run. The
// worker's /tmp is a pod-lifetime emptyDir in Kubernetes, so crash leaks
// survive container restarts. The files are tiny (a Dockerfile and a plan
// JSON per workspace), but they would otherwise accumulate until the pod is
// replaced. Any matching directory at process start is an orphan by
// definition: no builds are running yet.
func cleanupStaleRailpackWorkspaces() {
	matches, err := filepath.Glob(filepath.Join(os.TempDir(), railpackWorkspacePattern))
	if err != nil {
		logger.Error("unable to scan for stale railpack build workspaces", "error", err)
		return
	}
	for _, dir := range matches {
		if err := os.RemoveAll(dir); err != nil {
			logger.Error("unable to remove stale railpack build workspace", "dir", dir, "error", err)
			continue
		}
		logger.Warn("removed stale railpack build workspace from a previous run", "dir", dir)
	}
}

// shellEnvKeyRegex matches POSIX environment variable names. Stricter than
// validation.IsValidEnvVarKey (which mirrors K8s Secret keys and allows dots
// and hyphens) because Railpack secret names are referenced as shell variables
// in the generated prepare Dockerfile and mounted as process env at build time.
var shellEnvKeyRegex = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)

// buildRailpackImageFromGit builds a container image from a GitHub repository
// without a Dockerfile.
//
// Repository content never touches the worker: both solves use a git context
// URL, so the build machine fetches the source itself — the same trust model
// as Dockerfile builds. The flow acquires one Depot machine and performs two
// solves on it:
//
//  1. Plan generation: a worker-generated Dockerfile runs `railpack prepare`
//     inside the pinned Railpack builder image against the git context. Only
//     the resulting plan JSON is exported back to the worker.
//  2. Image build: the Railpack BuildKit frontend (gateway.v0, our fork with
//     git context support) consumes the plan and the same git context, and
//     pushes the image to the registry.
//
// The worker's local footprint is two tiny files in a temp dir: the generated
// prepare Dockerfile and the returned plan.
func (w *Workflow) buildRailpackImageFromGit(
	ctx restate.Context,
	params gitBuildParams,
) (*buildResult, error) {
	logger.Info("Starting railpack build process",
		"repository", params.Repository,
		"commit_sha", params.CommitSHA,
		"project_id", params.ProjectID,
		"platform", w.buildPlatform.Platform,
	)

	depotProjectID, err := restate.Run(ctx, func(runCtx restate.RunContext) (string, error) {
		return w.getOrCreateDepotProject(runCtx, params.ProjectID)
	}, restate.WithName("get or create depot project"), restate.WithMaxRetryAttempts(runMaxAttempts))
	if err != nil {
		return nil, fmt.Errorf("failed to get/create depot project: %w", err)
	}

	return restate.Run(ctx, func(runCtx restate.RunContext) (*buildResult, error) {
		// Get a GitHub installation token so the build machine can fetch the
		// repository. Mirrors the Dockerfile path: in unauthenticated mode
		// (local dev, public repos) the git fetch runs without credentials.
		var ghToken githubclient.InstallationToken
		if w.allowUnauthenticatedDeployments && params.InstallationID == noInstallationID {
			logger.Info("Unauthenticated mode: skipping GitHub authentication for public repo",
				"repository", params.Repository)
		} else {
			token, err := w.github.GetInstallationToken(params.InstallationID)
			if err != nil {
				return nil, fmt.Errorf("failed to get GitHub installation token: %w", err)
			}
			ghToken = token
		}

		// Decrypt env vars in-memory. They are needed twice: at plan time so
		// Railpack registers them as build secrets (and honors RAILPACK_*
		// configuration variables), and at build time as BuildKit secrets.
		// Decryption failures are terminal — see buildDockerImageFromGit.
		envVars, err := w.decryptEnvVars(runCtx, params.EncryptedEnvironmentVariables, params.EnvironmentID)
		if err != nil {
			return nil, restate.TerminalError(fmt.Errorf("failed to decrypt env vars for build: %w", err))
		}

		envKeys, err := sortedShellEnvKeys(envVars)
		if err != nil {
			return nil, restate.TerminalError(err)
		}

		// The workspace only holds the generated prepare Dockerfile and the
		// returned plan JSON — a few KB, cleaned up per attempt.
		workDir, err := os.MkdirTemp("", railpackWorkspacePattern)
		if err != nil {
			return nil, fmt.Errorf("failed to create build workspace: %w", err)
		}
		defer func() {
			if removeErr := os.RemoveAll(workDir); removeErr != nil {
				logger.Error("unable to clean up railpack build workspace", "error", removeErr)
			}
		}()

		planDir := filepath.Join(workDir, "plan")
		prepareDir := filepath.Join(workDir, "prepare")
		for _, dir := range []string{planDir, prepareDir} {
			if err := os.MkdirAll(dir, 0o755); err != nil {
				return nil, fmt.Errorf("failed to create build workspace: %w", err)
			}
		}

		prepareDockerfile, err := buildRailpackPrepareDockerfile(railpackFrontendImage, railpackBuilderImage, envKeys, hashEnvVars(envVars))
		if err != nil {
			return nil, restate.TerminalError(err)
		}
		if err := os.WriteFile(filepath.Join(prepareDir, "Dockerfile"), []byte(prepareDockerfile), 0o644); err != nil {
			return nil, fmt.Errorf("failed to write prepare dockerfile: %w", err)
		}

		gitContextURL := buildGitContextURL(params)
		imageName := fmt.Sprintf("%s:%s-%s", w.registryConfig.Repository, params.ProjectID, params.DeploymentID)

		logger.Info("Starting railpack build execution",
			"image_name", imageName,
			"platform", w.buildPlatform.Platform,
			"project_id", params.ProjectID,
			"git_context_url", gitContextURL,
			"frontend_image", railpackFrontendImage,
		)

		// One machine, two solves: plan generation, then the image build.
		depotBuildID, err := w.withDepotBuildkit(runCtx, depotProjectID, params, func(buildClient *client.Client) error {
			prepareOptions, optErr := w.buildRailpackPrepareSolverOptions(gitContextURL, prepareDir, planDir, ghToken.Token, envVars)
			if optErr != nil {
				return restate.TerminalError(fmt.Errorf("failed to build prepare solver options: %w", optErr))
			}
			if solveErr := w.solveWithStatus(runCtx, buildClient, params, prepareOptions); solveErr != nil {
				return fmt.Errorf("railpack prepare failed: %w", solveErr)
			}
			if planErr := validateRailpackPlan(filepath.Join(planDir, railpackPlanFilename)); planErr != nil {
				return restate.TerminalError(fmt.Errorf("railpack prepare failed: %w", planErr))
			}

			buildOptions, optErr := w.buildRailpackSolverOptions(gitContextURL, planDir, imageName, params.ProjectID, ghToken.Token, envVars)
			if optErr != nil {
				return restate.TerminalError(fmt.Errorf("failed to build solver options: %w", optErr))
			}
			return w.solveWithStatus(runCtx, buildClient, params, buildOptions)
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
	}, restate.WithName("build railpack image from git"),
		// Same retry bounds as the Dockerfile path: count for transient blips,
		// wall-clock because a single attempt is long-running.
		restate.WithMaxRetryAttempts(runMaxAttempts),
		restate.WithMaxRetryDuration(buildImageRetryCeiling))
}

// sortedShellEnvKeys validates that every env var name is a POSIX shell
// variable name and returns them sorted. Railpack secret names become shell
// variables in the generated prepare Dockerfile and process env at build
// time, so other names cannot work and must be rejected rather than dropped.
func sortedShellEnvKeys(envVars map[string]string) ([]string, error) {
	keys := slices.Sorted(maps.Keys(envVars))
	for _, key := range keys {
		if !shellEnvKeyRegex.MatchString(key) {
			return nil, fmt.Errorf(
				"environment variable %q is not usable with railpack builds: names must contain only letters, digits, and underscores, and must not start with a digit",
				key,
			)
		}
	}
	return keys, nil
}

// railpackPrepareDockerfileTemplate renders the Dockerfile for the plan
// generation solve.
//
// The syntax directive is required: the secret env mounts need Dockerfile
// frontend >= 1.10, independent of the build machine's BuildKit version.
//
// Secret mounts are not part of BuildKit's RUN cache key, so the hash of the
// secret values is embedded in the command text — it forces plan regeneration
// when values change, since they can influence the generated plan.
var railpackPrepareDockerfileTemplate = template.Must(template.New("railpack-prepare").Parse(
	`# syntax=docker/dockerfile:1
# Generated by the Unkey deploy worker. Runs Railpack plan generation on
# the build machine so repository content is never processed by the
# control plane. The builder stage is glibc-based because railpack downloads
# a gnu-libc mise binary at plan time; the cache mount persists that download
# and mise's metadata across builds.
FROM {{ .FrontendImage }} AS railpack
FROM {{ .BuilderImage }} AS prepare
COPY --from=railpack /railpack /usr/local/bin/railpack
RUN --mount=type=bind,target=/workspace,readonly \
    --mount=type=cache,target=/tmp/railpack
{{- range .EnvKeys }} \
    --mount=type=secret,id={{ . }},env={{ . }}
{{- end }} \
    {{ if .SecretsHash }}RAILPACK_SECRETS_HASH={{ .SecretsHash }} {{ end -}}
    /usr/local/bin/railpack prepare /workspace --plan-out /railpack-plan.json
{{- range .EnvKeys }} --env {{ . }}="${{ . }}"{{ end }}

FROM scratch
COPY --from=prepare /railpack-plan.json /
`))

// buildRailpackPrepareDockerfile generates the Dockerfile for the plan
// generation solve. It copies the railpack binary out of the frontend image
// (which keeps it at /railpack), runs `railpack prepare` inside the builder
// image against the build context, and exposes only the plan JSON in a
// scratch stage so the local exporter returns nothing else.
//
// Only validated env var NAMES are embedded in the Dockerfile; values flow
// through BuildKit session secrets and are expanded by the shell at run time.
func buildRailpackPrepareDockerfile(frontendImage, builderImage string, envKeys []string, secretsHash string) (string, error) {
	var b strings.Builder
	err := railpackPrepareDockerfileTemplate.Execute(&b, struct {
		FrontendImage string
		BuilderImage  string
		EnvKeys       []string
		SecretsHash   string
	}{
		FrontendImage: frontendImage,
		BuilderImage:  builderImage,
		EnvKeys:       envKeys,
		SecretsHash:   secretsHash,
	})
	if err != nil {
		return "", fmt.Errorf("failed to render prepare dockerfile: %w", err)
	}
	return b.String(), nil
}

// buildRailpackPrepareSolverOptions constructs the solver configuration for
// the plan generation solve: a Dockerfile build over the git context whose
// only output is the plan JSON, exported back to planDir on the worker.
// The dockerfile.v0 frontend fetches the git context on the build machine.
func (w *Workflow) buildRailpackPrepareSolverOptions(
	gitContextURL, prepareDir, planDir, githubToken string,
	envVars map[string]string,
) (client.SolveOpt, error) {
	prepareFS, err := fsutil.NewFS(prepareDir)
	if err != nil {
		return client.SolveOpt{}, fmt.Errorf("failed to create dockerfile mount: %w", err)
	}

	var sessionAttachables []session.Attachable
	if secrets := railpackSecrets(githubToken, envVars); len(secrets) > 0 {
		sessionAttachables = append(sessionAttachables, secretsprovider.FromMap(secrets))
	}

	//nolint: exhaustruct
	return client.SolveOpt{
		Frontend: "dockerfile.v0",
		FrontendAttrs: map[string]string{
			"platform": w.buildPlatform.Platform,
			"context":  gitContextURL,
			"filename": "Dockerfile",
			// With a git context the dockerfile frontend reads the Dockerfile
			// from the repository by default — which has none, that's the
			// point. dockerfilekey forces it to read our generated Dockerfile
			// from the session's "dockerfile" local mount instead.
			"dockerfilekey": "dockerfile",
		},
		LocalMounts: map[string]fsutil.FS{
			"dockerfile": prepareFS,
		},
		Session: sessionAttachables,
		Exports: []client.ExportEntry{
			{
				Type:      client.ExporterLocal,
				OutputDir: planDir,
			},
		},
	}, nil
}

// buildRailpackSolverOptions constructs the BuildKit solver configuration for
// the image build solve: the gateway frontend runs the pinned Railpack
// frontend image, which reads the plan from the "dockerfile" local mount and
// fetches the application source from the git context on the build machine.
func (w *Workflow) buildRailpackSolverOptions(
	gitContextURL, planDir, imageName, projectID, githubToken string,
	envVars map[string]string,
) (client.SolveOpt, error) {
	planFS, err := fsutil.NewFS(planDir)
	if err != nil {
		return client.SolveOpt{}, fmt.Errorf("failed to create plan mount: %w", err)
	}

	frontendAttrs := map[string]string{
		"source":   railpackFrontendImage,
		"platform": w.buildPlatform.Platform,
		"context":  gitContextURL,
		"filename": railpackPlanFilename,
		// cache-key prefixes BuildKit mount-cache IDs. Depot already scopes
		// caches per Unkey project, but the explicit key guards against cache
		// sharing if builds are ever consolidated onto shared machines.
		"build-arg:cache-key": projectID,
	}
	if len(envVars) > 0 {
		// The frontend mounts a file derived from this hash so layers consuming
		// secrets are invalidated when secret values change.
		frontendAttrs["build-arg:secrets-hash"] = hashEnvVars(envVars)
	}

	sessionAttachables := []session.Attachable{
		w.registryAuthProvider(),
	}
	if secrets := railpackSecrets(githubToken, envVars); len(secrets) > 0 {
		sessionAttachables = append(sessionAttachables, secretsprovider.FromMap(secrets))
	}

	//nolint: exhaustruct
	return client.SolveOpt{
		Frontend:      "gateway.v0",
		FrontendAttrs: frontendAttrs,
		LocalMounts: map[string]fsutil.FS{
			"dockerfile": planFS,
		},
		Session: sessionAttachables,
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

// railpackSecrets assembles the BuildKit session secrets for a Railpack
// solve: the git auth token for the context fetch (when authenticated) and
// each env var by name. Railpack registers env var names as build secrets in
// the plan, and both solves mount the values from the session by name.
func railpackSecrets(githubToken string, envVars map[string]string) map[string][]byte {
	secrets := make(map[string][]byte, len(envVars)+1)
	if githubToken != "" {
		secrets[gitAuthTokenSecretID] = []byte(githubToken)
	}
	for k, v := range envVars {
		secrets[k] = []byte(v)
	}
	return secrets
}

// validateRailpackPlan sanity-checks the plan file exported from the prepare
// solve before it is handed to the Railpack frontend.
func validateRailpackPlan(planPath string) error {
	info, err := os.Stat(planPath)
	if err != nil {
		return fmt.Errorf("build plan was not produced: %w", err)
	}
	if info.Size() == 0 {
		return fmt.Errorf("build plan is empty")
	}
	if info.Size() > railpackPlanBytesMax {
		return fmt.Errorf("build plan is unexpectedly large: %d bytes", info.Size())
	}

	raw, err := os.ReadFile(planPath)
	if err != nil {
		return fmt.Errorf("build plan could not be read: %w", err)
	}
	if !json.Valid(raw) {
		return fmt.Errorf("build plan is not valid JSON")
	}
	return nil
}
