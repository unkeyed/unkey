package deploy

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"
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

	"github.com/unkeyed/unkey/pkg/clickhouse/schema"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/logger"
	"github.com/unkeyed/unkey/pkg/ptr"
	githubclient "github.com/unkeyed/unkey/svc/ctrl/worker/github"
)

const (
	// defaultCacheKeepGB is the maximum cache size in gigabytes for new Depot
	// projects. Depot evicts least-recently-used cache entries when exceeded.
	defaultCacheKeepGB = 25

	// defaultCacheKeepDays is the maximum age in days for cached build layers.
	// Layers older than this are evicted regardless of cache size.
	defaultCacheKeepDays = 7
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
	{substr: "failed to solve: process", message: "A build command failed. Please check the build logs for details."},
	{substr: "linting failed", message: "Dockerfile linting failed. Please check the Dockerfile for issues."},
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
	InstallationID                int64
	Repository                    string
	CommitSHA                     string
	ContextPath                   string
	DockerfilePath                string
	ProjectID                     string
	AppID                         string
	DeploymentID                  string
	WorkspaceID                   string
	PrNumber                      int64
	EncryptedEnvironmentVariables []byte
	EnvironmentID                 string
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
	architecture := w.buildPlatform.Architecture

	logger.Info("Starting git build process",
		"repository", params.Repository,
		"commit_sha", params.CommitSHA,
		"project_id", params.ProjectID,
		"platform", platform,
		"architecture", architecture)

	depotProjectID, err := restate.Run(ctx, func(runCtx restate.RunContext) (string, error) {
		return w.getOrCreateDepotProject(runCtx, params.ProjectID)
	}, restate.WithName("get or create depot project"))
	if err != nil {
		return nil, fmt.Errorf("failed to get/create depot project: %w", err)
	}

	logger.Info("Creating depot build",
		"depot_project_id", depotProjectID,
		"project_id", params.ProjectID)

	return restate.Run(ctx, func(runCtx restate.RunContext) (*buildResult, error) {
		// Get GitHub installation token for BuildKit to fetch the repo
		var ghToken githubclient.InstallationToken
		if w.allowUnauthenticatedDeployments && params.InstallationID == noInstallationID {
			// Unauthenticated mode - skip GitHub auth for public repos (local dev only)
			logger.Info("Unauthenticated mode: skipping GitHub authentication for public repo",
				"repository", params.Repository)
		} else {
			token, err := w.github.GetInstallationToken(params.InstallationID)
			if err != nil {
				return nil, fmt.Errorf("failed to get GitHub installation token: %w", err)
			}
			ghToken = token
		}

		// Decrypt env vars in-memory so they can be injected as a BuildKit secret.
		envVars, err := w.decryptEnvVars(runCtx, params.EncryptedEnvironmentVariables, params.EnvironmentID)
		if err != nil {
			if errors.Is(err, errInvalidSecretsConfig) {
				return nil, restate.TerminalError(fmt.Errorf("failed to decrypt env vars for build: %w", err))
			}
			return nil, fmt.Errorf("failed to decrypt env vars for build: %w", err)
		}

		depotBuild, err := build.NewBuild(runCtx, &cliv1.CreateBuildRequest{
			Options:   nil,
			ProjectId: depotProjectID,
		}, w.registryConfig.Password)
		if err != nil {
			return nil, fmt.Errorf("failed to create build: %w", err)
		}
		defer func() { depotBuild.Finish(err) }()

		logger.Info("Depot build created",
			"build_id", depotBuild.ID,
			"depot_project_id", depotProjectID,
			"project_id", params.ProjectID)

		logger.Info("Acquiring build machine",
			"build_id", depotBuild.ID,
			"architecture", architecture,
			"project_id", params.ProjectID)

		buildkit, err := machine.Acquire(runCtx, depotBuild.ID, depotBuild.Token, architecture)
		if err != nil {
			return nil, fmt.Errorf("failed to acquire machine: %w", err)
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
			return nil, fmt.Errorf("unable to create build client: %w", err)
		}
		defer func() {
			if closeErr := buildClient.Close(); closeErr != nil {
				logger.Error("unable to close client", "error", closeErr)
			}
		}()

		imageName := fmt.Sprintf("%s:%s-%s", w.registryConfig.Repository, params.ProjectID, params.DeploymentID)

		dockerfilePath := params.DockerfilePath
		if dockerfilePath == "" {
			dockerfilePath = "Dockerfile"
		}

		// Normalize context path: trim whitespace and leading slashes, treat "." as root
		contextPath := strings.TrimSpace(params.ContextPath)
		contextPath = strings.TrimPrefix(contextPath, "/")
		if contextPath == "." {
			contextPath = ""
		}

		// Build git context URL with commit SHA or PR ref.
		// Format: https://github.com/owner/repo.git#<ref>:<subdir>
		// For fork PRs, use refs/pull/<number>/head so BuildKit can fetch
		// the fork's commits from the base repo.
		ref := params.CommitSHA
		if params.PrNumber > 0 {
			ref = fmt.Sprintf("refs/pull/%d/head", params.PrNumber)
		}
		gitContextURL := fmt.Sprintf("https://github.com/%s.git#%s", params.Repository, ref)
		if contextPath != "" {
			gitContextURL = fmt.Sprintf("https://github.com/%s.git#%s:%s", params.Repository, ref, contextPath)
		}

		logger.Info("Starting build execution",
			"image_name", imageName,
			"dockerfile", dockerfilePath,
			"platform", platform,
			"architecture", architecture,
			"build_id", depotBuild.ID,
			"project_id", params.ProjectID,
			"git_context_url", gitContextURL,
		)

		buildStatusCh := make(chan *client.SolveStatus, 100)
		go w.processBuildStatus(buildStatusCh, params.WorkspaceID, params.ProjectID, params.DeploymentID)

		// Choose solver options based on authentication mode
		var solverOptions client.SolveOpt
		if w.allowUnauthenticatedDeployments && params.InstallationID == noInstallationID {
			solverOptions, err = w.buildSolverOptions(platform, gitContextURL, dockerfilePath, imageName, envVars)
		} else {
			solverOptions, err = w.buildGitSolverOptions(platform, gitContextURL, dockerfilePath, imageName, ghToken.Token, envVars)
		}
		if err != nil {
			return nil, restate.TerminalError(fmt.Errorf("failed to build solver options: %w", err))
		}

		_, err = buildClient.Solve(runCtx, nil, solverOptions, buildStatusCh)
		if err != nil {
			// Context cancellations and timeouts are transient — let Restate retry.
			if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
				return nil, fmt.Errorf("build interrupted: %w", err)
			}
			return nil, restate.TerminalError(fmt.Errorf("build failed: %w", err))
		}

		logger.Info("Build completed successfully")

		return &buildResult{
			ImageName:      imageName,
			DepotBuildID:   depotBuild.ID,
			DepotProjectID: depotProjectID,
		}, nil
	}, restate.WithName("build docker image from git"))
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
		//nolint: exhaustruct
		authprovider.NewDockerAuthProvider(authprovider.DockerAuthProviderConfig{
			ConfigFile: &configfile.ConfigFile{
				AuthConfigs: map[string]types.AuthConfig{
					w.registryConfig.Repository: {
						Username: w.registryConfig.Username,
						Password: w.registryConfig.Password,
					},
				},
			},
		}),
	}

	envFile, err := buildEnvFileSecret(envVars)
	if err != nil {
		return client.SolveOpt{}, fmt.Errorf("invalid environment variables: %w", err)
	}
	if envFile != nil {
		sessionAttachables = append(sessionAttachables, secretsprovider.FromMap(map[string][]byte{"env": envFile}))
	}

	frontendAttrs := map[string]string{
		"platform": platform,
		"context":  contextURL,
		"filename": dockerfilePath,
	}
	if h := hashEnvVars(envVars); h != "" {
		frontendAttrs["label:org.unkey.env-hash"] = h
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
		"GIT_AUTH_TOKEN.github.com": []byte(githubToken),
	}
	envFile, err := buildEnvFileSecret(envVars)
	if err != nil {
		return client.SolveOpt{}, fmt.Errorf("invalid environment variables: %w", err)
	}
	if envFile != nil {
		secrets["env"] = envFile
	}

	frontendAttrs := map[string]string{
		"platform": platform,
		"context":  gitContextURL,
		"filename": dockerfilePath,
	}
	if h := hashEnvVars(envVars); h != "" {
		frontendAttrs["label:org.unkey.env-hash"] = h
	}

	return client.SolveOpt{
		Frontend:      "dockerfile.v0",
		FrontendAttrs: frontendAttrs,

		Session: []session.Attachable{
			//nolint: exhaustruct
			authprovider.NewDockerAuthProvider(authprovider.DockerAuthProviderConfig{
				ConfigFile: &configfile.ConfigFile{
					AuthConfigs: map[string]types.AuthConfig{
						w.registryConfig.Repository: {
							Username: w.registryConfig.Username,
							Password: w.registryConfig.Password,
						},
					},
				},
			}),
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
	project, err := db.Query.FindProjectById(ctx, w.db.RO(), unkeyProjectID)
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
	err = db.Query.UpdateProjectDepotID(ctx, w.db.RW(), db.UpdateProjectDepotIDParams{
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
