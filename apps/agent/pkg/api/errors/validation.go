package errors

import (
	"context"
	"net/http"

	"github.com/Southclaws/fault/fmsg"
	"github.com/unkeyed/unkey/apps/agent/gen/openapi"
	"github.com/unkeyed/unkey/apps/agent/pkg/api/ctxutil"
)

func HandleError(ctx context.Context, err error) openapi.ErrorModel {

	issues := fmsg.GetIssues(err)
	details := make([]openapi.ErrorDetail, len(issues))
	for i, issue := range issues {
		details[i] = openapi.ErrorDetail{
			Message: issue,
		}
	}

	return openapi.ErrorModel{
		Title:     "Internal Server Error",
		Detail:    "An internal server error occurred",
		Errors:    details,
		Instance:  "https://errors.unkey.com/todo",
		Status:    http.StatusInternalServerError,
		RequestId: ctxutil.GetRequestId(ctx),
		Type:      "TODO docs link",
	}

}
