package version

import (
	"context"

	"connectrpc.com/connect"
	ctrlv1 "github.com/unkeyed/unkey/go/gen/proto/ctrl/v1"
	"github.com/unkeyed/unkey/go/gen/proto/ctrl/v1/ctrlv1connect"
	"github.com/unkeyed/unkey/go/pkg/db"
)

type BuildService interface {
	CreateBuild(ctx context.Context, req *connect.Request[ctrlv1.CreateBuildRequest]) (*connect.Response[ctrlv1.CreateBuildResponse], error)
}

type Service struct {
	ctrlv1connect.UnimplementedVersionServiceHandler
	db           db.Database
	buildService BuildService
}

func New(database db.Database, buildService BuildService) *Service {
	return &Service{
		UnimplementedVersionServiceHandler: ctrlv1connect.UnimplementedVersionServiceHandler{},
		db:                                 database,
		buildService:                       buildService,
	}
}
