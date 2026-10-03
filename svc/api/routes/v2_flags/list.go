package handler

import (
	"context"
	"net/http"

	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/zen"
	"github.com/unkeyed/unkey/svc/api/openapi"
)

type ListHandler struct {
	DB db.Database
}

func (h *ListHandler) Method() string { return http.MethodPost }
func (h *ListHandler) Path() string   { return "/v2/flags.listFlags" }

func (h *ListHandler) Handle(ctx context.Context, s *zen.Session) error {
	p, err := workspaceUser(s, false)
	if err != nil {
		return err
	}
	if _, err := zen.BindBody[openapi.V2FlagsListFlagsRequestBody](s); err != nil {
		return err
	}
	rows, err := db.Query.ListFlags(ctx, h.DB.RO(), p.AuthorizedWorkspaceID)
	if err != nil {
		return err
	}
	data := make([]openapi.WorkspaceFlag, 0, len(rows))
	for _, row := range rows {
		flag := resolve(db.Flag{
			Pk: row.Pk, ID: row.ID, Slug: row.Slug, Description: row.Description,
			DefaultValue: row.DefaultValue, AllowOptIn: row.AllowOptIn, AllowOptOut: row.AllowOptOut,
		}, row.OverrideValue)
		data = append(data, flag)
	}
	return s.JSON(http.StatusOK, openapi.V2FlagsListFlagsResponseBody{
		Meta: openapi.Meta{RequestId: s.RequestID()}, Data: data,
	})
}
