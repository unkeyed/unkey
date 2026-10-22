package keyleak

import (
	"context"
	"time"

	restate "github.com/restatedev/sdk-go"
	hydrav1 "github.com/unkeyed/unkey/gen/proto/hydra/v1"
	"github.com/unkeyed/unkey/pkg/email"
)

func (s *Service) NotifyEmail(ctx restate.Context, req *hydrav1.KeyLeakEmailRequest) (*hydrav1.KeyLeakResponse, error) {
	message := email.Email{
		From:       "",
		To:         []string{req.GetEmail()},
		Subject:    "",
		TemplateID: "secret-scanning-key-detected",
		Variables: map[string]string{
			"DATE":   req.GetLeak().GetDate(),
			"SOURCE": req.GetLeak().GetSource(),
			"URL":    req.GetLeak().GetUrl(),
		},
		IdempotencyKey: "leaked-key:" + ctx.Request().ID,
	}

	err := restate.RunVoid(ctx, func(rc restate.RunContext) error {
		sendCtx, cancel := context.WithTimeout(rc, 10*time.Second)
		defer cancel()

		return s.email.Send(sendCtx, message)
	}, runOptions("send key leak email")...)

	return &hydrav1.KeyLeakResponse{}, err
}
