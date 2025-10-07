package build

import (
	"os"

	"github.com/unkeyed/unkey/go/gen/proto/ctrl/v1/ctrlv1connect"
	"github.com/unkeyed/unkey/go/pkg/db"
)

type Service struct {
	ctrlv1connect.UnimplementedBuildServiceHandler
	instanceID string
	db         db.Database
	depotToken string
}

func New(instanceID string, database db.Database) *Service {
	depotToken := os.Getenv("DEPOT_TOKEN")
	if depotToken == "" {
		panic("DEPOT_TOKEN environment variable is required")
	}

	return &Service{
		UnimplementedBuildServiceHandler: ctrlv1connect.UnimplementedBuildServiceHandler{},
		instanceID:                       instanceID,
		db:                               database,
		depotToken:                       depotToken,
	}
}
