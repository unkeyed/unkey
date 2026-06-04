package handler

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"time"

	"github.com/unkeyed/unkey/internal/services/auditlogs"
	"github.com/unkeyed/unkey/pkg/auditlog"
	"github.com/unkeyed/unkey/pkg/auth"
	"github.com/unkeyed/unkey/pkg/codes"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/fault"
	"github.com/unkeyed/unkey/pkg/rbac"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/pkg/zen"
	"github.com/unkeyed/unkey/svc/api/openapi"
)

type (
	Request  = openapi.V2ApisCreateApiRequestBody
	Response = openapi.V2ApisCreateApiResponseBody
)

type Handler struct {
	DB        db.Database
	Auth      auth.Service
	Auditlogs auditlogs.AuditLogService
}

func (h *Handler) Method() string {
	return "POST"
}

func (h *Handler) Path() string {
	return "/v2/apis.createApi"
}

func (h *Handler) Handle(ctx context.Context, s *zen.Session) error {
	principal, err := h.Auth.Authenticate(ctx, s)
	if err != nil {
		return err
	}

	req, err := zen.BindBody[Request](s)
	if err != nil {
		return err
	}

	err = h.Auth.Authorize(ctx, principal, rbac.T(rbac.Tuple{
		ResourceType: rbac.Api,
		ResourceID:   "*",
		Action:       rbac.CreateAPI,
	}))
	if err != nil {
		return err
	}

	apiId, err := db.TxWithResultRetry(ctx, h.DB.RW(), func(ctx context.Context, tx db.DBTX) (string, error) {
		keySpaceId := uid.New(uid.KeySpacePrefix)
		err = db.Query.InsertKeySpace(ctx, tx, db.InsertKeySpaceParams{
			ID:                 keySpaceId,
			WorkspaceID:        principal.WorkspaceID,
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
			WorkspaceID: principal.WorkspaceID,
			AuthType:    db.NullApisAuthType{Valid: true, ApisAuthType: db.ApisAuthTypeKey},
			KeyAuthID:   sql.NullString{Valid: true, String: keySpaceId},
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
				WorkspaceID:   principal.WorkspaceID,
				Event:         auditlog.APICreateEvent,
				Display:       fmt.Sprintf("Created API %s", apiId),
				ActorID:       principal.Subject.ID,
				ActorName:     principal.Subject.Name,
				ActorMeta:     map[string]any{},
				ActorType:     auditlog.AuditLogActor(principal.Subject.Type),
				RemoteIP:      s.Location(),
				UserAgent:     s.UserAgent(),
				CorrelationID: "",
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
