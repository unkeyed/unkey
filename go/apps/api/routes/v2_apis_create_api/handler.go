package handler

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"time"

	"github.com/unkeyed/unkey/go/apps/api/openapi"
	"github.com/unkeyed/unkey/go/internal/services/auditlogs"
	"github.com/unkeyed/unkey/go/internal/services/keys"
	"github.com/unkeyed/unkey/go/pkg/auditlog"
	"github.com/unkeyed/unkey/go/pkg/codes"
	"github.com/unkeyed/unkey/go/pkg/db"
	"github.com/unkeyed/unkey/go/pkg/fault"
	"github.com/unkeyed/unkey/go/pkg/otel/logging"
	"github.com/unkeyed/unkey/go/pkg/rbac"
	"github.com/unkeyed/unkey/go/pkg/uid"
	"github.com/unkeyed/unkey/go/pkg/zen"
)

type Request = openapi.V2ApisCreateApiRequestBody
type Response = openapi.V2ApisCreateApiResponseBody

type Handler struct {
	Logger    logging.Logger
	DB        db.Database
	Keys      keys.KeyService
	Auditlogs auditlogs.AuditLogService
}

func (h *Handler) Method() string {
	return "POST"
}

func (h *Handler) Path() string {
	return "/v2/apis.createApi"
}

func (h *Handler) Handle(ctx context.Context, s *zen.Session) error {
	auth, emit, err := h.Keys.GetRootKey(ctx, s)
	defer emit()
	if err != nil {
		return err
	}

	req, err := zen.BindBody[Request](s)
	if err != nil {
		return err
	}

	err = auth.VerifyRootKey(ctx, keys.WithPermissions(rbac.Or(
		rbac.T(rbac.Tuple{
			ResourceType: rbac.Api,
			ResourceID:   "*",
			Action:       rbac.CreateAPI,
		}),
	)))
	if err != nil {
		return err
	}

	apiId, err := db.TxWithResult(ctx, h.DB.RW(), func(ctx context.Context, tx db.DBTX) (string, error) {
		keyAuthId := uid.New(uid.KeyAuthPrefix)
		err = db.Query.InsertKeyring(ctx, tx, db.InsertKeyringParams{
			ID:                 keyAuthId,
			WorkspaceID:        auth.AuthorizedWorkspaceID,
			CreatedAtM:         time.Now().UnixMilli(),
			DefaultPrefix:      sql.NullString{Valid: false, String: ""},
			DefaultBytes:       sql.NullInt32{Valid: false, Int32: 0},
			StoreEncryptedKeys: false,
		})
		if err != nil {
			return "", fault.Wrap(err,
				fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
				fault.Internal("unable to create key auth"), fault.Public("We're unable to create key authentication for the API."),
			)
		}

		apiId := uid.New(uid.APIPrefix)
		err = db.Query.InsertApi(ctx, tx, db.InsertApiParams{
			ID:          apiId,
			Name:        req.Name,
			WorkspaceID: auth.AuthorizedWorkspaceID,
			AuthType:    db.NullApisAuthType{Valid: true, ApisAuthType: db.ApisAuthTypeKey},
			KeyAuthID:   sql.NullString{Valid: true, String: keyAuthId},
			IpWhitelist: sql.NullString{Valid: false, String: ""},
			CreatedAtM:  time.Now().UnixMilli(),
		})
		if err != nil {
			return "", fault.Wrap(err,
				fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
				fault.Internal("unable to create api"), fault.Public("We're unable to create the API."),
			)
		}

		err = h.Auditlogs.Insert(ctx, tx, []auditlog.AuditLog{
			{
				WorkspaceID: auth.AuthorizedWorkspaceID,
				Event:       auditlog.APICreateEvent,
				Display:     fmt.Sprintf("Created API %s", apiId),
				ActorID:     auth.Key.ID,
				ActorName:   "root key",
				ActorMeta:   map[string]any{},
				ActorType:   auditlog.RootKeyActor,
				RemoteIP:    s.Location(),
				UserAgent:   s.UserAgent(),
				Resources: []auditlog.AuditLogResource{
					{
						ID:          apiId,
						Type:        auditlog.APIResourceType,
						Meta:        nil,
						Name:        req.Name,
						DisplayName: req.Name,
					},
				},
			},
		})
		if err != nil {
			return "", err
		}

		return apiId, nil
	})
	if err != nil {
		return err
	}

	return s.JSON(http.StatusOK, Response{
		Meta: openapi.Meta{
			RequestId: s.RequestID(),
		},
		Data: openapi.V2ApisCreateApiResponseData{
			ApiId: apiId,
		},
	})
}
