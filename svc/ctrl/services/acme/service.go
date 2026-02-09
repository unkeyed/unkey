package acme

import (
	"github.com/unkeyed/unkey/gen/proto/ctrl/v1/ctrlv1connect"
	"github.com/unkeyed/unkey/pkg/cache"
	"github.com/unkeyed/unkey/pkg/db"
)

type Service struct {
	ctrlv1connect.UnimplementedAcmeServiceHandler
	db             db.Database
	domainCache    cache.Cache[string, db.CustomDomain]
	challengeCache cache.Cache[string, db.AcmeChallenge]
}

type Config struct {
	DB             db.Database
	DomainCache    cache.Cache[string, db.CustomDomain]
	ChallengeCache cache.Cache[string, db.AcmeChallenge]
}

func New(cfg Config) *Service {
	return &Service{
		UnimplementedAcmeServiceHandler: ctrlv1connect.UnimplementedAcmeServiceHandler{},
		db:                              cfg.DB,
		domainCache:                     cfg.DomainCache,
		challengeCache:                  cfg.ChallengeCache,
	}
}
