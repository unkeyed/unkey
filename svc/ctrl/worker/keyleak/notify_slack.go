package keyleak

import (
	"fmt"

	restate "github.com/restatedev/sdk-go"
	hydrav1 "github.com/unkeyed/unkey/gen/proto/hydra/v1"
	"github.com/unkeyed/unkey/svc/ctrl/internal/slack"
)

func (s *Service) NotifySlack(ctx restate.Context, req *hydrav1.KeyLeakNotification) (*hydrav1.KeyLeakResponse, error) {
	leakDetails := fmt.Sprintf("Type: %s\nSource: %s\nDate: %s\nURL: %s", req.GetType(), req.GetSource(), req.GetDate(), req.GetUrl())
	keyDetails := fmt.Sprintf("Key: %s\nWorkspace: %s\nTenant: %s", req.GetKeyId(), req.GetWorkspaceName(), req.GetOrgId())
	message := slack.Payload{
		Text: "Leaked Key Found",
		Blocks: []slack.Block{slack.NewSectionBlock(
			slack.Field{Type: "plain_text", Text: leakDetails},
			slack.Field{Type: "plain_text", Text: keyDetails},
		)},
	}

	err := restate.RunVoid(ctx, func(rc restate.RunContext) error {
		return s.slack.Send(rc, s.slackWebhookURL, message)
	}, runOptions("send key leak Slack alert")...)

	return &hydrav1.KeyLeakResponse{}, err
}
