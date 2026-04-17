package sentinel

import (
	"fmt"

	restate "github.com/restatedev/sdk-go"
	"github.com/unkeyed/unkey/pkg/logger"
	"github.com/unkeyed/unkey/svc/ctrl/internal/slack"
)

// notifySlack sends a notification if a webhook URL is configured in the
// rollout state. Errors are logged but do not fail the rollout.
func notifySlack(ctx restate.ObjectContext, webhookURL, title, message string) {
	if webhookURL == "" {
		return
	}

	_, err := restate.Run(ctx, func(rc restate.RunContext) (restate.Void, error) {
		payload := slack.Payload{
			Text: fmt.Sprintf("%s: %s", title, message),
			Blocks: []slack.Block{
				slack.NewHeaderBlock(title),
				slack.NewSectionBlock(
					slack.NewMarkdownField(message),
				),
			},
		}

		sendErr := slack.NewClient().Send(rc, webhookURL, payload)
		if sendErr != nil {
			logger.Error("failed to send rollout slack notification", "error", sendErr)
		}

		return restate.Void{}, nil
	}, restate.WithName("send slack notification"))
	if err != nil {
		logger.Error("restate run failed for slack notification", "error", err)
	}
}
