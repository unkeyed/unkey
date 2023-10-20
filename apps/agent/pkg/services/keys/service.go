package keys

import (
	"context"

	keysv1 "github.com/unkeyed/unkey/apps/agent/gen/proto/keys/v1"
	"github.com/unkeyed/unkey/apps/agent/pkg/events"
)

type KeyService interface {
	CreateKey(context.Context, *keysv1.CreateKeyRequest) (*keysv1.CreateKeyResponse, error)
}

type Database interface {
	InsertKey(context.Context, *keysv1.Key) error
}

type Config struct {
	Database Database
	Events   events.EventBus
}

type keyService struct {
	db     Database
	events events.EventBus
}

type Middleware func(KeyService) KeyService

func New(config Config, mws ...Middleware) KeyService {
	var svc KeyService = &keyService{
		db:     config.Database,
		events: config.Events,
	}

	for _, mw := range mws {
		svc = mw(svc)
	}
	return svc
}
