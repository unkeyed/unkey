package ctrl

import (
	"github.com/unkeyed/unkey/gen/proto/ctrl/v1/ctrlv1connect"
	"github.com/unkeyed/unkey/pkg/db"
)

type Service struct {
	ctrlv1connect.UnimplementedCtrlServiceHandler
	instanceID string
	db         db.Database
}

func New(instanceID string, database db.Database) *Service {
	return &Service{
		UnimplementedCtrlServiceHandler: ctrlv1connect.UnimplementedCtrlServiceHandler{},
		instanceID:                      instanceID,
		db:                              database,
	}
}
