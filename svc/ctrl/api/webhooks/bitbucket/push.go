package bitbucket

import (
	"context"
	"fmt"

	restate "github.com/restatedev/sdk-go"
	hydrav1 "github.com/unkeyed/unkey/gen/proto/hydra/v1"
	githubclient "github.com/unkeyed/unkey/pkg/github"
	"github.com/unkeyed/unkey/pkg/logger"
	"github.com/unkeyed/unkey/pkg/webhook"
)

type pushPayload struct {
	Push       pushDetails `json:"push"`
	Repository pushRepo    `json:"repository"`
	Actor      pushActor   `json:"actor"`
}

type pushDetails struct {
	Changes []pushChange `json:"changes"`
}

// pushChange is one ref update inside a push. New is nil when the ref was
// deleted, Old is nil when it was created.
type pushChange struct {
	New *pushRef `json:"new"`
}

type pushRef struct {
	// Type distinguishes "branch" from "tag" and friends.
	Type   string     `json:"type"`
	Name   string     `json:"name"`
	Target pushTarget `json:"target"`
}

// pushTarget is the commit the ref now points at. Bitbucket push payloads
// carry no per-commit file lists, so changed files are always empty and
// watch-path filtering has nothing to match against.
type pushTarget struct {
	Hash    string `json:"hash"`
	Message string `json:"message"`
	Date    string `json:"date"`
}

type pushRepo struct {
	// UUID is Bitbucket's only repository identifier (no numeric id exists);
	// RepoNumericID hashes it into the bigint connection columns.
	UUID     string `json:"uuid"`
	FullName string `json:"full_name"`
}

type pushActor struct {
	Nickname string    `json:"nickname"`
	Links    pushLinks `json:"links"`
}

type pushLinks struct {
	Avatar pushHref `json:"avatar"`
}

type pushHref struct {
	Href string `json:"href"`
}

// push parses the repo:push payload and sends a HandlePush request with
// provider="bitbucket" to the shared webhook Restate object.
func (h *handler) push(ctx context.Context, event webhook.Event, payload pushPayload) error {
	// A single push event can carry several ref updates. POC: handle the first
	// branch update only; a second send would reuse the delivery id and be
	// deduplicated by Restate anyway. Proper impl keys idempotency per ref.
	var change *pushRef
	for i := range payload.Push.Changes {
		if n := payload.Push.Changes[i].New; n != nil && n.Type == "branch" {
			change = n
			break
		}
	}
	if change == nil {
		return fmt.Errorf("%w: no branch update in push for %s", webhook.ErrIgnore, payload.Repository.FullName)
	}

	repoID := RepoNumericID(payload.Repository.UUID)
	commit := githubclient.CommitInfoFromRaw(
		change.Target.Hash,
		change.Target.Message,
		payload.Actor.Nickname,
		payload.Actor.Links.Avatar.Href,
		change.Target.Date,
	)

	// Key by provider:uuid for per-repository serialization, mirroring the
	// GitHub installation:repo and GitLab project id keys.
	objectKey := fmt.Sprintf("bitbucket:%s", payload.Repository.UUID)
	client := hydrav1.NewGitHubWebhookServiceIngressClient(h.restate, objectKey)

	deliveryID := event.ID
	var sendOpts []restate.IngressSendOption
	if deliveryID != "" {
		sendOpts = append(sendOpts, restate.WithIdempotencyKey(deliveryID))
	}

	_, err := client.HandlePush().Send(ctx, &hydrav1.HandlePushRequest{
		Provider:              "bitbucket",
		InstallationId:        repoID,
		RepositoryId:          repoID,
		RepositoryFullName:    payload.Repository.FullName,
		Branch:                change.Name,
		After:                 change.Target.Hash,
		CommitMessage:         commit.Message,
		CommitAuthorHandle:    commit.AuthorHandle,
		CommitAuthorAvatarUrl: commit.AuthorAvatarURL,
		CommitTimestamp:       commit.Timestamp.UnixMilli(),
		DeliveryId:            deliveryID,
		ChangedFiles:          nil,
		SenderLogin:           payload.Actor.Nickname,
	}, sendOpts...)
	if err != nil {
		return fmt.Errorf("enqueue push for %s: %w", payload.Repository.FullName, err)
	}

	logger.Info("Bitbucket push webhook enqueued to Restate",
		"delivery_id", deliveryID,
		"repository", payload.Repository.FullName,
		"branch", change.Name,
		"commit_sha", change.Target.Hash,
	)
	return nil
}
