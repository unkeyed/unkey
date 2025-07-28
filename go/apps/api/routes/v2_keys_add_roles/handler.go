package handler

import (
	"context"
	"fmt"
	"net/http"
	"slices"
	"time"

	"github.com/unkeyed/unkey/go/apps/api/openapi"
	"github.com/unkeyed/unkey/go/internal/services/auditlogs"
	"github.com/unkeyed/unkey/go/internal/services/keys"
	"github.com/unkeyed/unkey/go/pkg/auditlog"
	"github.com/unkeyed/unkey/go/pkg/cache"
	"github.com/unkeyed/unkey/go/pkg/codes"
	"github.com/unkeyed/unkey/go/pkg/db"
	"github.com/unkeyed/unkey/go/pkg/fault"
	"github.com/unkeyed/unkey/go/pkg/otel/logging"
	"github.com/unkeyed/unkey/go/pkg/rbac"
	"github.com/unkeyed/unkey/go/pkg/zen"
)

type Request = openapi.V2KeysAddRolesRequestBody
type Response = openapi.V2KeysAddRolesResponseBody

type Handler struct {
	Logger    logging.Logger
	DB        db.Database
	Keys      keys.KeyService
	Auditlogs auditlogs.AuditLogService
	KeyCache  cache.Cache[string, db.FindKeyForVerificationRow]
}

// Method returns the HTTP method this route responds to
func (h *Handler) Method() string {
	return "POST"
}

// Path returns the URL path pattern this route matches
func (h *Handler) Path() string {
	return "/v2/keys.addRoles"
}

// Handle processes the HTTP request
func (h *Handler) Handle(ctx context.Context, s *zen.Session) error {
	h.Logger.Debug("handling request", "requestId", s.RequestID(), "path", "/v2/keys.addRoles")

	auth, emit, err := h.Keys.GetRootKey(ctx, s)
	defer emit()
	if err != nil {
		return err
	}

	// 2. Request validation
	req, err := zen.BindBody[Request](s)
	if err != nil {
		return err
	}

	// 3. Permission check
	err = auth.VerifyRootKey(ctx, keys.WithPermissions(rbac.Or(
		rbac.T(rbac.Tuple{
			ResourceType: rbac.Api,
			ResourceID:   "*",
			Action:       rbac.UpdateKey,
		}),
	)))
	if err != nil {
		return err
	}

	// 4. Validate key exists and belongs to workspace
	key, err := db.Query.FindKeyByID(ctx, h.DB.RO(), req.KeyId)
	if err != nil {
		if db.IsNotFound(err) {
			return fault.New("key not found",
				fault.Code(codes.Data.Key.NotFound.URN()),
				fault.Internal("key not found"), fault.Public("The specified key was not found."),
			)
		}
		return fault.Wrap(err,
			fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
			fault.Internal("database error"), fault.Public("Failed to retrieve key."),
		)
	}

	// Validate key belongs to authorized workspace
	if key.WorkspaceID != auth.AuthorizedWorkspaceID {
		return fault.New("key not found",
			fault.Code(codes.Data.Key.NotFound.URN()),
			fault.Internal("key belongs to different workspace"), fault.Public("The specified key was not found."),
		)
	}

	if key.DeletedAtM.Valid {
		return fault.New("key not found",
			fault.Code(codes.Data.Key.NotFound.URN()),
			fault.Internal("key is deleted"), fault.Public("The specified key was not found."),
		)
	}

	// 5. Get current roles for the key
	currentRoles, err := db.Query.ListRolesByKeyID(ctx, h.DB.RO(), req.KeyId)
	if err != nil {
		return fault.Wrap(err,
			fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
			fault.Internal("database error"), fault.Public("Failed to retrieve current roles."),
		)
	}

	// 6. Resolve and validate requested roles
	requestedRoles := make([]db.Role, 0, len(req.Roles))
	for _, roleRef := range req.Roles {
		var role db.Role

		if roleRef.Id != nil {
			// Find by ID
			role, err = db.Query.FindRoleByID(ctx, h.DB.RO(), *roleRef.Id)
			if err != nil {
				if db.IsNotFound(err) {
					return fault.New("role not found",
						fault.Code(codes.Data.Role.NotFound.URN()),
						fault.Internal("role not found"), fault.Public(fmt.Sprintf("Role with ID '%s' was not found.", *roleRef.Id)),
					)
				}
				return fault.Wrap(err,
					fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
					fault.Internal("database error"), fault.Public("Failed to retrieve role."),
				)
			}
		} else if roleRef.Name != nil {
			// Find by name
			role, err = db.Query.FindRoleByNameAndWorkspaceID(ctx, h.DB.RO(), db.FindRoleByNameAndWorkspaceIDParams{
				Name:        *roleRef.Name,
				WorkspaceID: auth.AuthorizedWorkspaceID,
			})
			if err != nil {
				if db.IsNotFound(err) {
					return fault.New("role not found",
						fault.Code(codes.Data.Role.NotFound.URN()),
						fault.Internal("role not found"), fault.Public(fmt.Sprintf("Role with name '%s' was not found.", *roleRef.Name)),
					)
				}
				return fault.Wrap(err,
					fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
					fault.Internal("database error"), fault.Public("Failed to retrieve role."),
				)
			}
		} else {
			return fault.New("invalid role reference",
				fault.Code(codes.App.Validation.InvalidInput.URN()),
				fault.Internal("role missing id and name"), fault.Public("Each role must specify either 'id' or 'name'."),
			)
		}

		// Validate role belongs to the same workspace
		if role.WorkspaceID != auth.AuthorizedWorkspaceID {
			return fault.New("role not found",
				fault.Code(codes.Data.Role.NotFound.URN()),
				fault.Internal("role belongs to different workspace"), fault.Public(fmt.Sprintf("Role '%s' was not found.", role.Name)),
			)
		}

		requestedRoles = append(requestedRoles, role)
	}

	// 7. Determine which roles to add (only add roles that aren't already assigned)
	currentRoleIDs := make(map[string]bool)
	for _, role := range currentRoles {
		currentRoleIDs[role.ID] = true
	}

	rolesToAdd := make([]db.Role, 0)
	for _, role := range requestedRoles {
		if !currentRoleIDs[role.ID] {
			rolesToAdd = append(rolesToAdd, role)
		}
	}

	// 8. Apply changes in transaction (only if there are roles to add)
	if len(rolesToAdd) > 0 {
		err = db.Tx(ctx, h.DB.RW(), func(ctx context.Context, tx db.DBTX) error {
			var auditLogs []auditlog.AuditLog

			// Add new roles
			for _, role := range rolesToAdd {
				err = db.Query.InsertKeyRole(ctx, tx, db.InsertKeyRoleParams{
					KeyID:       req.KeyId,
					RoleID:      role.ID,
					WorkspaceID: auth.AuthorizedWorkspaceID,
					CreatedAtM:  time.Now().UnixMilli(),
				})
				if err != nil {
					return fault.Wrap(err,
						fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
						fault.Internal("database error"), fault.Public("Failed to add role assignment."),
					)
				}

				auditLogs = append(auditLogs, auditlog.AuditLog{
					WorkspaceID: auth.AuthorizedWorkspaceID,
					Event:       auditlog.AuthConnectRoleKeyEvent,
					ActorType:   auditlog.RootKeyActor,
					ActorID:     auth.Key.ID,
					ActorName:   "root key",
					ActorMeta:   map[string]any{},
					Display:     fmt.Sprintf("Added role %s to key %s", role.Name, req.KeyId),
					RemoteIP:    s.Location(),
					UserAgent:   s.UserAgent(),
					Resources: []auditlog.AuditLogResource{
						{
							Type:        "key",
							ID:          req.KeyId,
							Name:        key.Name.String,
							DisplayName: key.Name.String,
							Meta:        map[string]any{},
						},
						{
							Type:        "role",
							ID:          role.ID,
							Name:        role.Name,
							DisplayName: role.Name,
							Meta:        map[string]any{},
						},
					},
				})
			}

			// Insert audit logs
			if len(auditLogs) > 0 {
				err = h.Auditlogs.Insert(ctx, tx, auditLogs)
				if err != nil {
					return err
				}
			}

			return nil
		})
		if err != nil {
			return err
		}

		h.KeyCache.Remove(ctx, key.Hash)
	}

	// 9. Get final state of roles and build response
	finalRoles, err := db.Query.ListRolesByKeyID(ctx, h.DB.RW(), req.KeyId)
	if err != nil {
		return fault.Wrap(err,
			fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
			fault.Internal("database error"), fault.Public("Failed to retrieve final role state."),
		)
	}

	// Sort roles alphabetically by name for consistent response
	slices.SortFunc(finalRoles, func(a, b db.Role) int {
		if a.Name < b.Name {
			return -1
		} else if a.Name > b.Name {
			return 1
		}
		return 0
	})

	// Build response data
	responseData := make(openapi.V2KeysAddRolesResponseData, len(finalRoles))
	for i, role := range finalRoles {
		responseData[i] = struct {
			Id   string `json:"id"`
			Name string `json:"name"`
		}{
			Id:   role.ID,
			Name: role.Name,
		}
	}

	// 10. Return success response
	return s.JSON(http.StatusOK, Response{
		Meta: openapi.Meta{
			RequestId: s.RequestID(),
		},
		Data: responseData,
	})
}
