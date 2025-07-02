package version

import (
	"github.com/unkeyed/unkey/go/gen/proto/ctrl/v1/ctrlv1connect"
	"github.com/unkeyed/unkey/go/pkg/db"
)

type Service struct {
	ctrlv1connect.UnimplementedVersionServiceHandler
	db db.Database
}

func New(database db.Database) *Service {
	return &Service{
		UnimplementedVersionServiceHandler: ctrlv1connect.UnimplementedVersionServiceHandler{},
		db:                                 database,
	}
}
