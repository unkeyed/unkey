// Package github registers and routes inbound GitHub App webhook events for
// ctrl-api. Transport concerns (signature verification, routing, metrics) live
// in pkg/webhook; each event type is handled in its own file. The handlers do
// no DB access; processing happens durably in the Restate object they dispatch
// to.
package github

import (
	"context"
	"fmt"
	"net/http"

	restateingress "github.com/restatedev/sdk-go/ingress"
	"github.com/unkeyed/unkey/pkg/webhook"
	githubverifier "github.com/unkeyed/unkey/pkg/webhook/verifiers/github"
)

// handler holds the dependencies the GitHub event handlers need.
type handler struct {
	restate *restateingress.Client
}

// New builds the /webhooks/github handler.
func New(restateClient *restateingress.Client, webhookSecret string) http.Handler {
	h := &handler{restate: restateClient}
	return webhook.New("github", githubverifier.New(webhookSecret)).
		On([]string{"push"}, webhook.Typed(h.push)).
		On([]string{"pull_request"}, webhook.Typed(h.pullRequest)).
		// Branch lifecycle events carry no code to deploy; the first push of a
		// new branch arrives as its own push event (created: true).
		On([]string{"create", "delete", "installation"}, ignoreEvent).
		// GitHub Apps receive every subscribed event; anything without a handler
		// is deliberately not deployment-relevant.
		Default(ignoreEvent)
}

func ignoreEvent(_ context.Context, event webhook.Event) error {
	return fmt.Errorf("%w: no deployment action for %s events", webhook.ErrIgnore, event.Type)
}
