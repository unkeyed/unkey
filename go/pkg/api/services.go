package api

import (
	"github.com/unkeyed/unkey/go/pkg/api/validation"
	"github.com/unkeyed/unkey/go/pkg/logging"
)

type Services struct {
	Logger    logging.Logger
	Validator validation.OpenAPIValidator
}
