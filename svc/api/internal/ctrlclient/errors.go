package ctrlclient

import (
	"errors"
	"fmt"

	"connectrpc.com/connect"
	"github.com/unkeyed/unkey/pkg/codes"
	"github.com/unkeyed/unkey/pkg/fault"
)

// HandleError converts Connect RPC errors from ctrl services to fault errors
// with appropriate error codes and user-facing messages.
//
// The context parameter should describe the operation being performed (e.g., "create deployment",
// "generate upload URL") and will be used to generate user-facing error messages.
func HandleError(err error, context string) error {
	// Convert Connect errors to fault errors
	var connectErr *connect.Error
	if !errors.As(err, &connectErr) {
		// Non-Connect errors
		return fault.Wrap(err,
			fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
			fault.Public(fmt.Sprintf("Failed to %s.", context)),
		)
	}

	//nolint:exhaustive // Default case handles all other Connect error codes
	switch connectErr.Code() {
	case connect.CodeNotFound:
		return fault.Wrap(err,
			fault.Code(codes.Data.Project.NotFound.URN()),
			fault.Public("Project not found."),
		)
	case connect.CodeInvalidArgument:
		return fault.Wrap(err,
			fault.Code(codes.App.Validation.InvalidInput.URN()),
			fault.Public(fmt.Sprintf("Invalid request for %s.", context)),
		)
	case connect.CodeUnauthenticated:
		return fault.Wrap(err,
			fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
			fault.Public("Failed to authenticate with service."),
		)
	default:
		// All other Connect errors (Internal, Unavailable, etc.)
		return fault.Wrap(err,
			fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
			fault.Public(fmt.Sprintf("Failed to %s.", context)),
		)
	}
}
