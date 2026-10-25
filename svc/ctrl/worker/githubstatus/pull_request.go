package githubstatus

import (
	"errors"
	"fmt"
	"strings"
	"time"

	restate "github.com/restatedev/sdk-go"
	"github.com/unkeyed/unkey/pkg/notification"
	"github.com/unkeyed/unkey/pkg/notification/deployment"
)

const commentServiceName = "notification.v1.GitHubPullRequestService"

type commentRequest struct {
	Target reportContext      `json:"target"`
	Event  notification.Event `json:"event"`
}

type commentRowState struct {
	DeploymentOrder uint64            `json:"deploymentOrder"`
	Change          deployment.Change `json:"change"`
	Fingerprint     string            `json:"fingerprint"`
	Status          outcome           `json:"status"`
}

type commentResult struct {
	Status            outcome `json:"status"`
	CreationUncertain bool    `json:"creationUncertain"`
}

func (s *Service) CommentDefinition() restate.ServiceDefinition {
	return restate.NewObject(commentServiceName).
		Handler("Update", restate.NewObjectHandler(s.UpdateComment, restate.WithIngressPrivate(true), restate.WithInvocationRetryPolicy(
			restate.WithInitialRetryInterval(time.Second), restate.WithMaxRetryInterval(time.Minute),
			restate.WithMaxRetryAttempts(20), restate.PauseOnMaxAttempts(),
		)))
}

func (s *Service) UpdateComment(ctx restate.ObjectContext, req commentRequest) (outcome, error) {
	fingerprint, err := req.Event.Fingerprint()
	if err != nil {
		return skipped, restate.ToTerminalError(err)
	}
	payload, err := deployment.Decode(req.Event)
	if err != nil || req.Target.InstallationID <= 0 || req.Target.RepositoryID <= 0 || req.Target.PullRequest <= 0 ||
		restate.Key(ctx) != commentKey(req.Target) {
		return skipped, restate.ToTerminalError(errors.New("invalid GitHub PR notification"))
	}
	rowKey := "row:" + req.Event.Resource.AppID + ":" + req.Event.Resource.EnvironmentID
	previous, err := restate.Get[*commentRowState](ctx, rowKey)
	if err != nil {
		return skipped, err
	}
	if previous != nil {
		if previous.DeploymentOrder > req.Target.DeploymentOrder {
			return skipped, nil
		}
		if previous.DeploymentOrder == req.Target.DeploymentOrder {
			if previous.Change == payload.Change {
				if previous.Fingerprint != fingerprint {
					return skipped, restate.ToTerminalError(errors.New("PR notification identity reused"))
				}
				return previous.Status, nil
			}
			if previous.Change.Terminal() || previous.Change.Revision() > payload.Change.Revision() {
				return skipped, nil
			}
		}
	}
	uncertain, err := restate.Get[bool](ctx, "creation_uncertain")
	if err != nil {
		return skipped, err
	}
	result, err := restate.Run(ctx, func(restate.RunContext) (commentResult, error) {
		return s.updateComment(req, payload, uncertain)
	}, restate.WithName("upsert shared deployment comment"), restate.WithMaxRetryDuration(30*time.Second))
	if err != nil {
		result = commentResult{Status: operationOutcome(err, req.Event.ID(), "upsert_comment"), CreationUncertain: uncertain}
	}
	restate.Set(ctx, "creation_uncertain", result.CreationUncertain)
	restate.Set(ctx, rowKey, commentRowState{
		DeploymentOrder: req.Target.DeploymentOrder, Change: payload.Change, Fingerprint: fingerprint, Status: result.Status,
	})
	return result.Status, nil
}

func (s *Service) updateComment(req commentRequest, payload deployment.Payload, uncertain bool) (commentResult, error) {
	target := req.Target
	id, body, err := s.github.FindBotComment(target.InstallationID, target.Repository, target.PullRequest, prCommentMainMarker)
	if err != nil {
		return commentResult{}, err
	}
	row := reportRow(req.Event, payload)
	if id == 0 {
		if uncertain {
			return commentResult{Status: unknown, CreationUncertain: true}, nil
		}
		id, err = s.github.CreateIssueComment(target.InstallationID, target.Repository, target.PullRequest, buildFullComment(row))
		if err == nil && id <= 0 {
			err = errors.New("invalid GitHub comment id")
		}
		return commentResult{Status: operationOutcome(err, req.Event.ID(), "create_comment"), CreationUncertain: err != nil}, nil
	}
	legacyMarker := rowMarker(payload.App, payload.Environment)
	name := payload.Project
	if payload.App != "default" {
		name += " / " + payload.App
	}
	lines := strings.Split(body, "\n")
	for index, line := range lines {
		if strings.Contains(line, legacyMarker) && strings.Contains(line, "**"+name+"** ("+payload.Environment+")") {
			lines[index] = strings.Replace(line, legacyMarker, rowMarker(req.Event.Resource.AppID, req.Event.Resource.EnvironmentID), 1)
		}
	}
	body = upsertRow(req.Event.Resource.AppID, req.Event.Resource.EnvironmentID, strings.Join(lines, "\n"), row)
	if err := s.github.UpdateIssueComment(target.InstallationID, target.Repository, id, body); err != nil {
		return commentResult{}, err
	}
	return commentResult{Status: accepted, CreationUncertain: false}, nil
}

func reportRow(event notification.Event, payload deployment.Payload) string {
	name := payload.Project
	if payload.App != "default" {
		name += " / " + payload.App
	}
	preview := "—"
	if payload.EnvironmentURL != "" {
		preview = fmt.Sprintf("[Visit Preview](%s)", payload.EnvironmentURL)
	}
	return fmt.Sprintf("| %s **%s** (%s) | %s | %s | [Inspect](%s) | %s |",
		rowMarker(event.Resource.AppID, event.Resource.EnvironmentID), name, payload.Environment,
		stateLabel(gitHubState(payload.Change)), preview, payload.LogURL,
		time.UnixMilli(event.OccurredAt).UTC().Format("Jan 2, 2006 3:04pm"))
}
