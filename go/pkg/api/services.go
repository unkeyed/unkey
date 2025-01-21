package api

import (
	"github.com/unkeyed/unkey/go/pkg/database"
	"github.com/unkeyed/unkey/go/pkg/logging"
)

type Services struct {
	Logger   logging.Logger
	Database database.Database
}
