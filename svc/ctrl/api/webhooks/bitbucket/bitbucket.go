// Package bitbucket registers and routes inbound Bitbucket Cloud webhook
// events for ctrl-api. Transport concerns (signature verification, routing,
// metrics) live in pkg/webhook. The handlers do no DB access; processing
// happens durably in the same Restate object that serves GitHub pushes, with
// provider="bitbucket" set on the request.
package bitbucket

import (
	"context"
	"fmt"
	"hash/fnv"
	"math"
	"net/http"

	restateingress "github.com/restatedev/sdk-go/ingress"
	"github.com/unkeyed/unkey/pkg/webhook"
	bitbucketverifier "github.com/unkeyed/unkey/pkg/webhook/verifiers/bitbucket"
)

// handler holds the dependencies the Bitbucket event handlers need.
type handler struct {
	restate *restateingress.Client
}

// New builds the /webhooks/bitbucket handler.
func New(restateClient *restateingress.Client, webhookSecret string) http.Handler {
	h := &handler{restate: restateClient}
	return webhook.New("bitbucket", bitbucketverifier.New(webhookSecret)).
		On([]string{"repo:push"}, webhook.Typed(h.push)).
		// Pull requests are not deployment-relevant yet: fork PR builds need a
		// per-provider security story before they can deploy.
		Default(ignoreEvent)
}

func ignoreEvent(_ context.Context, event webhook.Event) error {
	return fmt.Errorf("%w: no deployment action for %s events", webhook.ErrIgnore, event.Type)
}

// RepoNumericID maps a Bitbucket repository UUID (e.g.
// "{9970a9b6-2d86-413f-8555-da8e1ac0e542}") onto the bigint
// installation_id/repository_id columns the connection schema inherited from
// GitHub. Bitbucket has no numeric repo id, so this FNV-64a hash (masked
// positive to survive signed bigint and proto int64) stands in for one. The
// connect flow and the webhook receiver must both use it so pushes find their
// connection rows. POC shim: the proper implementation stores provider-native
// string ids instead.
func RepoNumericID(uuid string) int64 {
	h := fnv.New64a()
	h.Write([]byte(uuid))
	return int64(h.Sum64() & math.MaxInt64)
}
