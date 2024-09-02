package errors

import (
	"context"
	"net/http"

	"github.com/Southclaws/fault/fmsg"
	"github.com/unkeyed/unkey/apps/agent/gen/openapi"
	"github.com/unkeyed/unkey/apps/agent/pkg/api/ctxutil"
)

func HandleValidationError(ctx context.Context, err error) openapi.ErrorModel {

	issues := fmsg.GetIssues(err)
	details := make([]openapi.ErrorDetail, len(issues))
	for i, issue := range issues {
		details[i] = openapi.ErrorDetail{
			Message: issue,
		}
	}

	return openapi.ErrorModel{
		Title:     "Validation Error",
		Detail:    "One or more fields failed validation",
		Errors:    details,
		Instance:  "https://errors.unkey.com/todo",
		Status:    http.StatusBadRequest,
		RequestId: ctxutil.GetRequestId(ctx),
		Type:      "TODO docs link",
	}

}
