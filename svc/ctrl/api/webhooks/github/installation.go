package github

import (
	"context"
	"fmt"

	restate "github.com/restatedev/sdk-go"
	hydrav1 "github.com/unkeyed/unkey/gen/proto/hydra/v1"
	"github.com/unkeyed/unkey/pkg/webhook"
)

type installationPayload struct {
	Action       string           `json:"action"`
	Installation pushInstallation `json:"installation"`
}

func (h *handler) installation(ctx context.Context, event webhook.Event, payload installationPayload) error {
	if payload.Action != "deleted" {
		return fmt.Errorf("%w: installation action %s", webhook.ErrIgnore, payload.Action)
	}

	client := hydrav1.NewGitHubWebhookServiceIngressClient(h.restate, fmt.Sprintf("%d", payload.Installation.ID))
	_, err := client.HandleInstallationDeleted().Send(ctx, &hydrav1.HandleInstallationDeletedRequest{
		InstallationId: payload.Installation.ID,
	}, restate.WithIdempotencyKey(event.ID))
	if err != nil {
		return fmt.Errorf("enqueue installation deletion: %w", err)
	}
	return nil
}
