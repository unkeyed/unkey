package deployment

import (
	"context"
	"fmt"

	"connectrpc.com/connect"
	ctrlv1 "github.com/unkeyed/unkey/gen/proto/ctrl/v1"
	"github.com/unkeyed/unkey/pkg/auditlog"
	"github.com/unkeyed/unkey/pkg/deploy/imageref"
	"github.com/unkeyed/unkey/pkg/logger"
	"github.com/unkeyed/unkey/svc/ctrl/internal/db"
)

const (
	// rebuildAuditActorID is the synthetic actor recorded when an
	// operator triggers a rebuild via the OpsService bearer. There is no
	// human user_id available at the RPC boundary — the bearer is the
	// auth boundary — so we tag every rebuild with the same system actor
	// and rely on `triggerReason` (free text on the new deployment row)
	// for incident attribution.
	rebuildAuditActorID   = "unkey-ops"
	rebuildAuditActorName = "Unkey Ops"
)

// Rebuild creates a new deployment cloning the source's project, app, and
// environment, then kicks off the deploy workflow. Explicit provenance decides
// whether to rebuild the pinned Git commit or reuse the resolved Docker digest;
// historical unknown rows retain the legacy Git-SHA inference.
//
// The new deployment inherits the app's *current* runtime settings and env
// vars — config drift since the source applies. That's the desired behavior
// for common use cases (hotfixing env vars, rolling out a runtime-settings
// change without a code change) as well as image-loss recovery.
//
// Guardrail (unless force=true): no newer active sibling on
// (app, environment, branch).
//
// The new deployment is persisted with trigger=unkey and the provided reason.
func (s *Service) Rebuild(ctx context.Context, sourceDeploymentID, reason string, force bool) (string, error) {
	if sourceDeploymentID == "" {
		return "", connect.NewError(connect.CodeInvalidArgument,
			fmt.Errorf("deployment_id is required"))
	}

	src, err := s.db.FindDeploymentById(ctx, sourceDeploymentID)
	if err != nil {
		if db.IsNotFound(err) {
			return "", connect.NewError(connect.CodeNotFound,
				fmt.Errorf("source deployment %q not found", sourceDeploymentID))
		}
		return "", connect.NewError(connect.CodeInternal,
			fmt.Errorf("failed to lookup source deployment: %w", err))
	}

	if !force {
		hasNewer, err := s.db.HasNewerActiveDeployment(ctx, db.HasNewerActiveDeploymentParams{
			AppID:         src.AppID,
			EnvironmentID: src.EnvironmentID,
			GitBranch:     src.GitBranch,
			CreatedAt:     src.CreatedAt,
			DeploymentID:  src.ID,
		})
		if err != nil {
			return "", connect.NewError(connect.CodeInternal,
				fmt.Errorf("failed to check for newer deployment: %w", err))
		}
		if hasNewer {
			return "", connect.NewError(connect.CodeFailedPrecondition,
				fmt.Errorf("a newer active deployment exists for this (app, environment, branch); pass force=true to override"))
		}
	}

	hasSha := src.GitCommitSha.Valid && src.GitCommitSha.String != ""
	hasRepoConn := false
	requiresGit := src.Source == db.DeploymentsSourceGit
	historicalSource := src.Source == db.DeploymentsSourceUnknown || src.Source == ""
	tryGit := requiresGit || (historicalSource && hasSha)
	if tryGit && hasSha {
		if _, repoErr := s.db.FindGithubRepoConnectionByAppId(ctx, src.AppID); repoErr == nil {
			hasRepoConn = true
		} else if !db.IsNotFound(repoErr) {
			return "", connect.NewError(connect.CodeInternal,
				fmt.Errorf("failed to lookup github repo connection: %w", repoErr))
		}
	}
	useGit := tryGit && hasSha && hasRepoConn
	resolvedImage := resolvedDeploymentImage(src)
	hasImage := resolvedImage.Valid && resolvedImage.String != ""

	if requiresGit && !useGit {
		return "", connect.NewError(connect.CodeFailedPrecondition,
			fmt.Errorf("source deployment %q is a Git build but has no usable commit and repository connection", sourceDeploymentID))
	}
	if !requiresGit && !hasImage {
		return "", connect.NewError(connect.CodeFailedPrecondition,
			fmt.Errorf("source deployment %q has no resolved Docker image; nothing to rebuild from", sourceDeploymentID))
	}

	env, err := s.db.FindEnvironmentById(ctx, src.EnvironmentID)
	if err != nil {
		return "", connect.NewError(connect.CodeInternal,
			fmt.Errorf("failed to lookup environment: %w", err))
	}

	depCtx, err := s.loadDeploymentContext(ctx, src.ProjectID, src.AppID, env.Slug)
	if err != nil {
		return "", err
	}

	params := createParams{
		context:     depCtx,
		dockerImage: "",
		gitCommit: &ctrlv1.GitCommitInfo{
			CommitSha:       src.GitCommitSha.String,
			Branch:          src.GitBranch.String,
			CommitMessage:   src.GitCommitMessage.String,
			AuthorHandle:    src.GitCommitAuthorHandle.String,
			AuthorAvatarUrl: src.GitCommitAuthorAvatarUrl.String,
			Timestamp:       src.GitCommitTimestamp.Int64,
			ForkRepository:  src.ForkRepositoryFullName.String,
		},
		command:       nil,
		trigger:       db.DeploymentsTriggerUnkey,
		triggeredBy:   "",
		triggerReason: reason,
	}

	if useGit {
		logger.Info("rebuilding deployment from git",
			"source_deployment_id", sourceDeploymentID,
			"app_id", src.AppID,
			"env_slug", env.Slug,
			"git_commit_sha", src.GitCommitSha.String,
			"reason", reason,
		)
	} else {
		// Reuse the source deployment's immutable image digest.
		// Passing dockerImage explicitly short-circuits createAndDeploy's
		// auto-detect, so we don't accidentally pick up the app's current
		// deployment image (which may be a different one).
		imageReference, imageErr := imageref.NormalizeHistorical(resolvedImage.String)
		if imageErr != nil {
			return "", connect.NewError(connect.CodeFailedPrecondition,
				fmt.Errorf("source deployment %q has an invalid Docker image: %w", sourceDeploymentID, imageErr))
		}
		params.dockerImage = imageReference
		logger.Info("rebuilding deployment by reusing source image",
			"source_deployment_id", sourceDeploymentID,
			"app_id", src.AppID,
			"env_slug", env.Slug,
			"image", resolvedImage.String,
			"reason", reason,
		)
	}

	newID, err := s.createAndDeploy(ctx, params)
	if err != nil {
		return "", err
	}

	// Persist an audit_log entry so customers can see in their audit feed
	// who replaced their deployment and why. Failure to write the audit
	// log must not undo the rebuild — the deployment is already running.
	if auditErr := s.recordRebuildAudit(ctx, src, newID, reason); auditErr != nil {
		logger.Error("failed to write rebuild audit log",
			"source_deployment_id", sourceDeploymentID,
			"new_deployment_id", newID,
			"error", auditErr,
		)
	}

	return newID, nil
}

// recordRebuildAudit writes an audit log entry tagged `deployment.rebuild`
// via the AuditLogService, with the source and new deployment as targets.
func (s *Service) recordRebuildAudit(
	ctx context.Context,
	src db.Deployment,
	newDeploymentID, reason string,
) error {
	display := fmt.Sprintf("Unkey rebuilt deployment %s as %s", src.ID, newDeploymentID)
	if reason != "" {
		display = fmt.Sprintf("%s (reason: %s)", display, reason)
	}

	return s.auditlogs.Insert(ctx, nil, []auditlog.AuditLog{
		{
			Event:       auditlog.DeploymentRebuildEvent,
			WorkspaceID: src.WorkspaceID,
			Display:     display,
			ActorID:     rebuildAuditActorID,
			ActorType:   auditlog.SystemActor,
			ActorName:   rebuildAuditActorName,
			ActorMeta: map[string]any{
				"reason": reason,
			},
			RemoteIP:      "",
			UserAgent:     "",
			CorrelationID: "",
			Resources: []auditlog.AuditLogResource{
				{
					Type:        auditlog.DeploymentResourceType,
					ID:          src.ID,
					Name:        "",
					DisplayName: src.ID,
					Meta:        map[string]any{"role": "source"},
				},
				{
					Type:        auditlog.DeploymentResourceType,
					ID:          newDeploymentID,
					Name:        "",
					DisplayName: newDeploymentID,
					Meta:        map[string]any{"role": "new"},
				},
			},
		},
	})
}
