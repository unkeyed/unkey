// Package gitlab registers and routes inbound GitLab webhook events for
// ctrl-api. Transport concerns (token verification, routing, metrics) live in
// pkg/webhook. The handlers do no DB access; processing happens durably in the
// same Restate object that serves GitHub pushes, with provider="gitlab" set on
// the request.
package gitlab

import (
	"context"
	"fmt"
	"net/http"

	restateingress "github.com/restatedev/sdk-go/ingress"
	"github.com/unkeyed/unkey/pkg/webhook"
	gitlabverifier "github.com/unkeyed/unkey/pkg/webhook/verifiers/gitlab"
)

// handler holds the dependencies the GitLab event handlers need.
type handler struct {
	restate *restateingress.Client
}

// New builds the /webhooks/gitlab handler.
func New(restateClient *restateingress.Client, webhookSecret string) http.Handler {
	h := &handler{restate: restateClient}
	return webhook.New("gitlab", gitlabverifier.New(webhookSecret)).
		On([]string{"Push Hook"}, webhook.Typed(h.push)).
		// Merge requests are not deployment-relevant yet: fork MR builds need a
		// per-provider security story before they can deploy.
		Default(ignoreEvent)
}

func ignoreEvent(_ context.Context, event webhook.Event) error {
	return fmt.Errorf("%w: no deployment action for %s events", webhook.ErrIgnore, event.Type)
}
