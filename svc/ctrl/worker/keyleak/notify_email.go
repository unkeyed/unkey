package keyleak

import (
	"context"
	"time"

	restate "github.com/restatedev/sdk-go"
	hydrav1 "github.com/unkeyed/unkey/gen/proto/hydra/v1"
	"github.com/unkeyed/unkey/pkg/assert"
	"github.com/unkeyed/unkey/pkg/email"
)

func (s *Service) NotifyEmail(ctx restate.Context, req *hydrav1.KeyLeakEmailRequest) (*hydrav1.KeyLeakResponse, error) {
	message, err := leakEmail(req, "leaked-key:"+ctx.Request().ID)
	if err != nil {
		return nil, restate.ToTerminalError(err)
	}

	err = restate.RunVoid(ctx, func(rc restate.RunContext) error {
		sendCtx, cancel := context.WithTimeout(rc, 10*time.Second)
		defer cancel()

		return s.email.Send(sendCtx, message)
	}, runOptions("send key leak email")...)

	return &hydrav1.KeyLeakResponse{}, err
}

func leakEmail(req *hydrav1.KeyLeakEmailRequest, idempotencyKey string) (email.Email, error) {
	leak := req.GetLeak()
	reporter := reporterOf(leak)
	template, found := reporter.emails[leak.GetKeyKind()]
	if err := assert.True(found, reporter.name+" has no email template for "+leak.GetKeyKind().String()); err != nil {
		return email.Email{}, err
	}

	variables := map[string]string{
		"DATE":   leak.GetDate(),
		"SOURCE": leak.GetSource(),
		"URL":    leak.GetUrl(),
	}
	if template.keyID {
		variables["KEY_ID"] = leak.GetKeyId()
	}
	return email.Email{
		From:           "",
		To:             []string{req.GetEmail()},
		Subject:        "",
		TemplateID:     template.id,
		Variables:      variables,
		IdempotencyKey: idempotencyKey,
	}, nil
}
