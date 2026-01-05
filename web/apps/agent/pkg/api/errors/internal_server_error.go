package errors

import (
	"context"
	"net/http"

	"github.com/Southclaws/fault/fmsg"
	"github.com/unkeyed/unkey/svc/agent/pkg/api/ctxutil"
	"github.com/unkeyed/unkey/svc/agent/pkg/openapi"
)

// HandleError takes in any unforseen error and returns a BaseError to be sent to the client
func HandleError(ctx context.Context, err error) openapi.BaseError {

	return openapi.BaseError{
		Title:     "Internal Server Error",
		Detail:    fmsg.GetIssue(err),
		Instance:  "https://errors.unkey.com/todo",
		Status:    http.StatusInternalServerError,
		RequestId: ctxutil.GetRequestId(ctx),
		Type:      "TODO docs link",
	}

}
