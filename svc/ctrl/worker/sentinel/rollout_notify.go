package sentinel

import (
	"fmt"
	"strings"
	"time"

	restate "github.com/restatedev/sdk-go"
	"github.com/unkeyed/unkey/pkg/logger"
	"github.com/unkeyed/unkey/svc/ctrl/internal/slack"
)

// nowMs captures wall-clock time deterministically inside a restate.Run so
// replays produce the same timestamp.
func nowMs(ctx restate.ObjectContext) int64 {
	ts, err := restate.Run(ctx, func(_ restate.RunContext) (int64, error) {
		return time.Now().UnixMilli(), nil
	}, restate.WithName("now"))
	if err != nil {
		logger.Error("failed to capture timestamp, falling back to local clock", "error", err)
		return time.Now().UnixMilli()
	}
	return ts
}

// durationMs wraps a unix-ms delta in a time.Duration so call sites can
// format it directly via `%s` — time.Duration.String() already renders
// values like "1h32m0s".
func durationMs(ms int64) time.Duration {
	return time.Duration(ms) * time.Millisecond
}

// truncateIDs returns a comma-separated list of up to maxShow IDs with a
// trailing "(+N more)" if truncated. Useful for surfacing failing sentinels
// without flooding a Slack message.
func truncateIDs(ids []string, maxShow int) string {
	if len(ids) == 0 {
		return ""
	}
	if len(ids) <= maxShow {
		return "`" + strings.Join(ids, "`, `") + "`"
	}
	return "`" + strings.Join(ids[:maxShow], "`, `") + "` (+" + fmt.Sprintf("%d more", len(ids)-maxShow) + ")"
}

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
