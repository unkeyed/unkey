// Package gitea registers and routes inbound Gitea webhook events for
// ctrl-api. Transport concerns (signature verification, routing, metrics)
// live in pkg/webhook. The handlers do no DB access; processing happens
// durably in the same Restate object that serves GitHub pushes, with
// provider="gitea" set on the request.
//
// Gitea is the first self-hosted provider: there is no fixed host, so the
// connection row carries the instance host (provider_host) and the worker
// derives the clone URL and BuildKit auth secret id from it.
package gitea

import (
	"context"
	"fmt"
	"net/http"

	restateingress "github.com/restatedev/sdk-go/ingress"
	"github.com/unkeyed/unkey/pkg/webhook"
	giteaverifier "github.com/unkeyed/unkey/pkg/webhook/verifiers/gitea"
)

// handler holds the dependencies the Gitea event handlers need.
type handler struct {
	restate *restateingress.Client
}

// New builds the /webhooks/gitea handler.
func New(restateClient *restateingress.Client, webhookSecret string) http.Handler {
	h := &handler{restate: restateClient}
	return webhook.New("gitea", giteaverifier.New(webhookSecret)).
		On([]string{"push"}, webhook.Typed(h.push)).
		// Pull requests are not deployment-relevant yet: fork PR builds need a
		// per-provider security story before they can deploy.
		Default(ignoreEvent)
}

func ignoreEvent(_ context.Context, event webhook.Event) error {
	return fmt.Errorf("%w: no deployment action for %s events", webhook.ErrIgnore, event.Type)
}
