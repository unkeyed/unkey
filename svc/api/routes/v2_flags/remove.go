package handler

import (
	"context"
	"database/sql"
	"net/http"

	"github.com/unkeyed/unkey/pkg/codes"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/fault"
	"github.com/unkeyed/unkey/pkg/zen"
	"github.com/unkeyed/unkey/svc/api/openapi"
)

type RemoveHandler struct {
	DB db.Database
}

func (h *RemoveHandler) Method() string { return http.MethodPost }
func (h *RemoveHandler) Path() string   { return "/v2/flags.removeOverride" }

func (h *RemoveHandler) Handle(ctx context.Context, s *zen.Session) error {
	p, err := workspaceUser(s, true)
	if err != nil {
		return err
	}
	req, err := zen.BindBody[openapi.V2FlagsRemoveOverrideRequestBody](s)
	if err != nil {
		return err
	}
	var data openapi.WorkspaceFlag
	err = db.TxRetry(ctx, h.DB.RW(), func(ctx context.Context, tx db.DBTX) error {
		flag, err := db.Query.FindFlagBySlug(ctx, tx, req.Slug)
		if db.IsNotFound(err) {
			return fault.New("unknown flag", fault.Code(codes.App.Validation.InvalidInput.URN()), fault.Public("Unknown platform feature."))
		}
		if err != nil {
			return err
		}
		if !flag.AllowOptOut {
			return forbidden("Only Unkey can restore the default for this platform feature.")
		}
		data = resolve(flag, sql.NullBool{})
		return db.Query.DeleteWorkspaceFlagOverride(ctx, tx, db.DeleteWorkspaceFlagOverrideParams{
			WorkspaceID: p.AuthorizedWorkspaceID, FlagID: flag.ID,
		})
	})
	if err != nil {
		return err
	}
	return s.JSON(http.StatusOK, openapi.V2FlagsOverrideResponseBody{Meta: openapi.Meta{RequestId: s.RequestID()}, Data: data})
}
