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

type SetHandler struct {
	DB db.Database
}

func (h *SetHandler) Method() string { return http.MethodPost }
func (h *SetHandler) Path() string   { return "/v2/flags.setOverride" }

func (h *SetHandler) Handle(ctx context.Context, s *zen.Session) error {
	p, err := workspaceUser(s, true)
	if err != nil {
		return err
	}
	req, err := zen.BindBody[openapi.V2FlagsSetOverrideRequestBody](s)
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
		if !flag.AllowOptIn {
			return forbidden("Changes to this platform feature are managed by Unkey.")
		}
		data = resolve(flag, sql.NullBool{Bool: req.Value, Valid: true})
		return db.Query.UpsertWorkspaceFlagOverride(ctx, tx, db.UpsertWorkspaceFlagOverrideParams{
			WorkspaceID: p.AuthorizedWorkspaceID, FlagID: flag.ID, Value: req.Value,
		})
	})
	if err != nil {
		return err
	}
	return s.JSON(http.StatusOK, openapi.V2FlagsOverrideResponseBody{Meta: openapi.Meta{RequestId: s.RequestID()}, Data: data})
}
