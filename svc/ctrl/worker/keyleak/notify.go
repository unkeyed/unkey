package keyleak

import (
	"context"
	"time"

	restate "github.com/restatedev/sdk-go"
	hydrav1 "github.com/unkeyed/unkey/gen/proto/hydra/v1"
)

func (s *Service) Notify(ctx restate.Context, req *hydrav1.KeyLeakNotification) (*hydrav1.KeyLeakResponse, error) {
	client := hydrav1.NewKeyLeakServiceClient(ctx)
	client.NotifySlack().Send(req)

	recipients, err := restate.Run(ctx, func(rc restate.RunContext) ([]string, error) {
		lookupCtx, cancel := context.WithTimeout(rc, 10*time.Second)
		defer cancel()

		return s.members.MemberEmails(lookupCtx, req.GetOrgId())
	}, runOptions("resolve key leak recipients")...)
	if err != nil {
		return nil, err
	}

	for _, recipient := range recipients {
		client.NotifyEmail().Send(&hydrav1.KeyLeakEmailRequest{Leak: req, Email: recipient})
	}

	return &hydrav1.KeyLeakResponse{}, nil
}
