package github

import (
	"database/sql"
	"fmt"
	"time"

	restate "github.com/restatedev/sdk-go"
	hydrav1 "github.com/unkeyed/unkey/gen/proto/hydra/v1"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/ctrl/pkg/repofetch"
)

const (
	// jobPollInterval is how often to check job status.
	jobPollInterval = 5 * time.Second

	// jobTimeout is the maximum time to wait for a fetch job.
	jobTimeout = 10 * time.Minute
)

// HandlePush executes the GitHub push to deployment workflow.
//
// This durable workflow orchestrates the complete flow from a GitHub push event
// to a deployment. The workflow is idempotent and can safely resume from any step
// after a crash.
//
// Steps:
// 1. Generate a GitHub installation access token
// 2. Generate an S3 presigned upload URL
// 3. Spawn a Kubernetes job to download tarball and upload to S3 (gVisor isolated)
// 4. Poll for job completion
// 5. Find the project and environment to deploy to
// 6. Create a deployment record and trigger the deployment workflow
//
// The fetch job runs with gVisor isolation in the "builds" namespace for security,
// preventing untrusted repository content from affecting the control plane.
//
// The workflow is keyed by project ID to ensure only one GitHub-triggered deployment
// runs per project at any time.
func (w *Workflow) HandlePush(ctx restate.WorkflowSharedContext, req *hydrav1.HandlePushRequest) (*hydrav1.HandlePushResponse, error) {
	w.logger.Info("GitHub push workflow started",
		"installation_id", req.GetInstallationId(),
		"repository", req.GetRepositoryFullName(),
		"commit_sha", req.GetCommitSha(),
		"project_id", req.GetProjectId(),
	)

	// Generate deployment ID early so we can use it for job naming (idempotency)
	deploymentID := uid.New(uid.DeploymentPrefix)
	buildContextPath := fmt.Sprintf("%s/%s.tar.gz", req.GetProjectId(), uid.New("build"))

	// Get GitHub installation token (short-lived, ~1 hour)
	githubToken, err := restate.Run(ctx, func(runCtx restate.RunContext) (string, error) {
		token, tokenErr := w.github.GetInstallationToken(req.GetInstallationId())
		if tokenErr != nil {
			return "", tokenErr
		}
		return token.Token, nil
	}, restate.WithName("get github installation token"))
	if err != nil {
		return nil, fmt.Errorf("failed to get installation token: %w", err)
	}

	// Generate S3 presigned upload URL (short-lived, 15 min)
	uploadURL, err := restate.Run(ctx, func(runCtx restate.RunContext) (string, error) {
		return w.buildStorage.GenerateUploadURL(runCtx, buildContextPath, 15*time.Minute)
	}, restate.WithName("generate s3 upload url"))
	if err != nil {
		return nil, fmt.Errorf("failed to generate upload URL: %w", err)
	}

	// Spawn the fetch job (runs in gVisor-isolated container)
	jobName, err := restate.Run(ctx, func(runCtx restate.RunContext) (string, error) {
		return w.fetchClient.SpawnFetchJob(runCtx, repofetch.FetchParams{
			DeploymentID: deploymentID,
			ProjectID:    req.GetProjectId(),
			Repo:         req.GetRepositoryFullName(),
			SHA:          req.GetCommitSha(),
			GitHubToken:  githubToken,
			UploadURL:    uploadURL,
		})
	}, restate.WithName("spawn fetch job"))
	if err != nil {
		return nil, fmt.Errorf("failed to spawn fetch job: %w", err)
	}

	w.logger.Info("Fetch job spawned",
		"job_name", jobName,
		"deployment_id", deploymentID,
	)

	// Poll for job completion
	err = restate.RunVoid(ctx, func(runCtx restate.RunContext) error {
		return w.waitForJobCompletion(runCtx, jobName)
	}, restate.WithName("wait for fetch job"))
	if err != nil {
		return nil, fmt.Errorf("fetch job failed: %w", err)
	}

	w.logger.Info("Fetch job completed",
		"job_name", jobName,
		"build_context_path", buildContextPath,
	)

	// Find project and environment
	project, err := restate.Run(ctx, func(runCtx restate.RunContext) (db.FindProjectByIdRow, error) {
		return db.Query.FindProjectById(runCtx, w.db.RO(), req.GetProjectId())
	}, restate.WithName("find project"))
	if err != nil {
		return nil, fmt.Errorf("failed to find project: %w", err)
	}

	env, err := restate.Run(ctx, func(runCtx restate.RunContext) (db.Environment, error) {
		return db.Query.FindEnvironmentByProjectIdAndSlug(runCtx, w.db.RO(), db.FindEnvironmentByProjectIdAndSlugParams{
			WorkspaceID: project.WorkspaceID,
			ProjectID:   project.ID,
			Slug:        "production",
		})
	}, restate.WithName("find environment"))
	if err != nil {
		return nil, fmt.Errorf("failed to find production environment: %w", err)
	}

	branch := "main"
	if req.GetGitCommit() != nil && req.GetGitCommit().GetBranch() != "" {
		branch = req.GetGitCommit().GetBranch()
	}

	now := time.Now().UnixMilli()

	err = restate.RunVoid(ctx, func(runCtx restate.RunContext) error {
		return db.Query.InsertDeployment(runCtx, w.db.RW(), db.InsertDeploymentParams{
			ID:                            deploymentID,
			K8sName:                       uid.DNS1035(12),
			WorkspaceID:                   project.WorkspaceID,
			ProjectID:                     project.ID,
			EnvironmentID:                 env.ID,
			SentinelConfig:                env.SentinelConfig,
			EncryptedEnvironmentVariables: []byte{},
			Command:                       []byte("[]"),
			Status:                        db.DeploymentsStatusPending,
			CreatedAt:                     now,
			UpdatedAt:                     sql.NullInt64{Valid: false},
			GitCommitSha:                  toNullString(req.GetCommitSha()),
			GitBranch:                     toNullString(branch),
			GitCommitMessage:              toNullString(getCommitMessage(req.GetGitCommit())),
			GitCommitAuthorHandle:         toNullString(getAuthorHandle(req.GetGitCommit())),
			GitCommitAuthorAvatarUrl:      toNullString(getAuthorAvatarURL(req.GetGitCommit())),
			GitCommitTimestamp:            toNullInt64(getTimestamp(req.GetGitCommit())),
			OpenapiSpec:                   sql.NullString{Valid: false},
			CpuMillicores:                 256,
			MemoryMib:                     256,
		})
	}, restate.WithName("insert deployment record"))
	if err != nil {
		return nil, fmt.Errorf("failed to insert deployment: %w", err)
	}

	w.logger.Info("Created deployment record",
		"deployment_id", deploymentID,
		"project_id", project.ID,
	)

	dockerfilePath := "Dockerfile"
	deployReq := &hydrav1.DeployRequest{
		DeploymentId:     deploymentID,
		BuildContextPath: &buildContextPath,
		DockerfilePath:   &dockerfilePath,
	}

	_ = hydrav1.NewDeploymentServiceClient(ctx, project.ID).Deploy().Send(deployReq)

	w.logger.Info("GitHub push workflow completed",
		"deployment_id", deploymentID,
		"project_id", project.ID,
		"repository", req.GetRepositoryFullName(),
	)

	return &hydrav1.HandlePushResponse{
		DeploymentId: deploymentID,
	}, nil
}

// waitForJobCompletion polls the job status until it completes or times out.
func (w *Workflow) waitForJobCompletion(ctx restate.RunContext, jobName string) error {
	deadline := time.Now().Add(jobTimeout)

	for {
		if time.Now().After(deadline) {
			return fmt.Errorf("fetch job timed out after %v", jobTimeout)
		}

		status, err := w.fetchClient.GetJobStatus(ctx, jobName)
		if err != nil {
			return fmt.Errorf("failed to get job status: %w", err)
		}

		switch status {
		case repofetch.JobStatusSucceeded:
			return nil
		case repofetch.JobStatusFailed:
			return fmt.Errorf("fetch job failed")
		case repofetch.JobStatusPending, repofetch.JobStatusRunning:
			time.Sleep(jobPollInterval)
		case repofetch.JobStatusUnknown:
			return fmt.Errorf("fetch job status unknown")
		}
	}
}

func toNullString(s string) sql.NullString {
	if s == "" {
		return sql.NullString{Valid: false}
	}
	return sql.NullString{String: s, Valid: true}
}

func toNullInt64(v int64) sql.NullInt64 {
	if v == 0 {
		return sql.NullInt64{Valid: false}
	}
	return sql.NullInt64{Int64: v, Valid: true}
}

func getCommitMessage(gc *hydrav1.GitCommitInfo) string {
	if gc == nil {
		return ""
	}
	return gc.GetCommitMessage()
}

func getAuthorHandle(gc *hydrav1.GitCommitInfo) string {
	if gc == nil {
		return ""
	}
	return gc.GetAuthorHandle()
}

func getAuthorAvatarURL(gc *hydrav1.GitCommitInfo) string {
	if gc == nil {
		return ""
	}
	return gc.GetAuthorAvatarUrl()
}

func getTimestamp(gc *hydrav1.GitCommitInfo) int64 {
	if gc == nil {
		return 0
	}
	return gc.GetTimestamp()
}
