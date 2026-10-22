package keyleak

import (
	restate "github.com/restatedev/sdk-go"
	hydrav1 "github.com/unkeyed/unkey/gen/proto/hydra/v1"
	"github.com/unkeyed/unkey/pkg/restate/restateutil"
)

// Definition rejects invalid notification requests before they can send alerts.
func (s *Service) Definition(opts ...restate.ServiceDefinitionOption) restate.ServiceDefinition {
	notificationCodec := restate.WithInputCodec(restateutil.ValidatedProtoJSON[*hydrav1.KeyLeakNotification]())
	emailCodec := restate.WithInputCodec(restateutil.ValidatedProtoJSON[*hydrav1.KeyLeakEmailRequest]())
	return hydrav1.NewKeyLeakServiceServer(s, opts...).
		ConfigureHandler("Notify", notificationCodec).
		ConfigureHandler("NotifySlack", notificationCodec).
		ConfigureHandler("NotifyEmail", emailCodec)
}
