package keys

import (
	"context"

	keysv1 "github.com/unkeyed/unkey/apps/agent/gen/proto/keys/v1"
	"github.com/unkeyed/unkey/apps/agent/pkg/cache"
	"github.com/unkeyed/unkey/apps/agent/pkg/events"
)

type KeyService interface {
	CreateKey(context.Context, *keysv1.CreateKeyRequest) (*keysv1.CreateKeyResponse, error)
	SoftDeleteKey(context.Context, *keysv1.SoftDeleteKeyRequest) (*keysv1.SoftDeleteKeyResponse, error)
}

type Database interface {
	InsertKey(context.Context, *keysv1.Key) error
	SoftDeleteKey(context.Context, string) error
	FindKeyById(context.Context, string) (*keysv1.Key, bool, error)
}

type Config struct {
	Database Database
	Events   events.EventBus
	KeyCache cache.Cache[*keysv1.Key]
}

type keyService struct {
	db       Database
	events   events.EventBus
	keyCache cache.Cache[*keysv1.Key]
}

type Middleware func(KeyService) KeyService

func New(config Config, mws ...Middleware) KeyService {
	keyCache := config.KeyCache
	if keyCache == nil {
		keyCache = cache.NewNoopCache[*keysv1.Key]()
	}
	var svc KeyService = &keyService{
		db:       config.Database,
		events:   config.Events,
		keyCache: keyCache,
	}

	for _, mw := range mws {
		svc = mw(svc)
	}
	return svc
}
