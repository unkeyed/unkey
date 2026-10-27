package githubstatus

import (
	"database/sql"
	"errors"
	"time"

	restate "github.com/restatedev/sdk-go"
	"github.com/unkeyed/unkey/pkg/notification"
	"github.com/unkeyed/unkey/pkg/notification/deployment"
	"github.com/unkeyed/unkey/svc/ctrl/internal/db"
)

type reportRecord struct {
	Fingerprint string       `json:"fingerprint"`
	Result      ReportResult `json:"result"`
}

type creationResult struct {
	ID     int64   `json:"id"`
	Status outcome `json:"status"`
}

// ReportDeployment requires the deployment ID as its object key. Duplicate events
// return the recorded result; stale changes and disconnected repositories are skipped.
func (s *Service) ReportDeployment(ctx restate.ObjectContext, event notification.Event) (ReportResult, error) {
	result := ReportResult{Commit: skipped, Deployment: skipped, Comment: skipped}
	fingerprint, err := event.Fingerprint()
	if err != nil {
		return result, restate.ToTerminalError(err)
	}
	payload, err := deployment.Decode(event)
	if err != nil || restate.Key(ctx) != payload.DeploymentID {
		return result, restate.ToTerminalError(errors.New("invalid GitHub deployment report"))
	}
	previous, err := restate.Get[*reportRecord](ctx, "event:"+event.ID())
	if err != nil {
		return result, err
	}
	if previous != nil {
		if previous.Fingerprint != fingerprint {
			return result, restate.ToTerminalError(errors.New("deployment event identity reused"))
		}
		return previous.Result, nil
	}
	last, err := restate.Get[deployment.Change](ctx, "last_change")
	if err != nil {
		return result, err
	}
	if last.Terminal() || last.Revision() >= payload.Change.Revision() {
		return result, nil
	}
	target, err := restate.Run(ctx, func(runCtx restate.RunContext) (reportContext, error) {
		return s.resolveReport(runCtx, event, payload)
	}, restate.WithName("resolve GitHub connection"))
	if err != nil || target.InstallationID == 0 {
		return result, err
	}
	original, err := restate.Get[*reportContext](ctx, "target")
	if err != nil {
		return result, err
	}
	if original != nil && (original.InstallationID != target.InstallationID || original.RepositoryID != target.RepositoryID) {
		return result, nil
	}
	if original == nil {
		restate.Set(ctx, "target", target)
	}

	if target.CommitSHA != "" {
		result.Commit, err = restate.Run(ctx, func(restate.RunContext) (outcome, error) {
			err := s.github.CreateCommitStatus(target.InstallationID, target.Repository, target.CommitSHA,
				gitHubCommitState(gitHubState(payload.Change)), payload.LogURL, payload.Change.Description(),
				"Unkey / "+environmentLabel(payload))
			return operationOutcome(err, event.ID(), "commit_status"), nil
		}, restate.WithName("report GitHub commit status"))
		if err != nil {
			return result, err
		}
	}
	created, err := s.ensureDeployment(ctx, target, event, payload)
	if err != nil {
		return result, err
	}
	result.Deployment = created.Status
	if created.ID > 0 {
		result.Deployment, err = restate.Run(ctx, func(restate.RunContext) (outcome, error) {
			err := s.github.CreateDeploymentStatus(target.InstallationID, target.Repository, created.ID,
				gitHubState(payload.Change), payload.EnvironmentURL, payload.LogURL, payload.Change.Description())
			return operationOutcome(err, event.ID(), "deployment_status"), nil
		}, restate.WithName("report GitHub deployment status"))
		if err != nil {
			return result, err
		}
	}
	result.Comment, err = s.reportComment(ctx, target, event)
	if err != nil {
		return result, err
	}
	restate.Set(ctx, "event:"+event.ID(), reportRecord{Fingerprint: fingerprint, Result: result})
	restate.Set(ctx, "last_change", payload.Change)
	return result, nil
}

func (s *Service) ensureDeployment(ctx restate.ObjectContext, target reportContext, event notification.Event, payload deployment.Payload) (creationResult, error) {
	previous, err := restate.Get[*creationResult](ctx, "creation")
	if err != nil {
		return creationResult{}, err
	}
	if previous != nil {
		return *previous, nil
	}
	created, err := restate.Run(ctx, func(restate.RunContext) (creationResult, error) {
		if target.GitHubDeploymentID > 0 {
			return creationResult{ID: target.GitHubDeploymentID, Status: accepted}, nil
		}
		if target.CommitSHA == "" {
			return creationResult{ID: 0, Status: skipped}, nil
		}
		id, err := s.github.CreateDeployment(target.InstallationID, target.Repository, target.CommitSHA, environmentLabel(payload), "Deploying...", target.Production)
		if err == nil && id <= 0 {
			err = errors.New("invalid GitHub deployment id")
		}
		return creationResult{ID: id, Status: operationOutcome(err, event.ID(), "create_deployment")}, nil
	}, restate.WithName("create GitHub deployment"))
	if err != nil {
		return created, err
	}
	restate.Set(ctx, "creation", created)
	if created.ID > 0 {
		err = restate.RunVoid(ctx, func(runCtx restate.RunContext) error {
			return s.db.UpdateDeploymentGithubDeploymentId(runCtx, db.UpdateDeploymentGithubDeploymentIdParams{
				ID: payload.DeploymentID, GithubDeploymentID: sql.NullInt64{Int64: created.ID, Valid: true},
				UpdatedAt: sql.NullInt64{Int64: event.OccurredAt, Valid: true},
			})
		}, restate.WithName("persist GitHub deployment id"))
	}
	return created, err
}

func (s *Service) reportComment(ctx restate.ObjectContext, target reportContext, event notification.Event) (outcome, error) {
	prNumber := target.PullRequest
	if prNumber == 0 && target.Branch != "" {
		var err error
		prNumber, err = restate.Run(ctx, func(restate.RunContext) (int, error) {
			return s.github.FindPullRequestForBranch(target.InstallationID, target.Repository, target.Branch)
		}, restate.WithName("find deployment PR"), restate.WithMaxRetryDuration(5*time.Second))
		if err != nil {
			return operationOutcome(err, event.ID(), "find_pull_request"), nil
		}
	}
	if prNumber == 0 {
		return skipped, nil
	}
	target.PullRequest = prNumber
	return restate.Object[outcome](ctx, commentServiceName, commentKey(target), "Update").Request(commentRequest{Target: target, Event: event})
}
