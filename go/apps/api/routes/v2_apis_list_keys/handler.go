package handler

import (
	"context"
	"net/http"

	"github.com/unkeyed/unkey/go/apps/api/openapi"
	"github.com/unkeyed/unkey/go/internal/services/keys"
	"github.com/unkeyed/unkey/go/internal/services/permissions"
	"github.com/unkeyed/unkey/go/internal/services/vault"
	"github.com/unkeyed/unkey/go/pkg/codes"
	"github.com/unkeyed/unkey/go/pkg/db"
	"github.com/unkeyed/unkey/go/pkg/fault"
	"github.com/unkeyed/unkey/go/pkg/otel/logging"
	"github.com/unkeyed/unkey/go/pkg/rbac"
	"github.com/unkeyed/unkey/go/pkg/zen"
)

type Request = openapi.V2ApisListKeysRequestBody
type Response = openapi.V2ApisListKeysResponseBody

type Services struct {
	Logger      logging.Logger
	DB          db.Database
	Keys        keys.KeyService
	Permissions permissions.PermissionService
	Vault       vault.VaultService
}

func New(svc Services) zen.Route {
	return zen.NewRoute("POST", "/v2/apis.listKeys", func(ctx context.Context, s *zen.Session) error {
		svc.Logger.Debug("handling request", "requestId", s.RequestID(), "path", "/v2/apis.listKeys")

		// 1. Authentication
		auth, err := svc.Keys.VerifyRootKey(ctx, s)
		if err != nil {
			return err
		}

		// 2. Request validation
		var req Request
		err = s.BindBody(&req)
		if err != nil {
			return fault.Wrap(err,
				fault.WithDesc("invalid request body", "The request body is invalid."),
			)
		}

		// Set default limit if not provided
		if req.Limit == 0 {
			req.Limit = 100
		} else if req.Limit > 100 {
			req.Limit = 100
		}

		// 3. Permission check
		permissionCheck, err := svc.Permissions.Check(
			ctx,
			auth.KeyID,
			rbac.Or(
				rbac.And(
					rbac.Or(
						rbac.T(rbac.Tuple{
							ResourceType: rbac.Api,
							ResourceID:   "*",
							Action:       rbac.ReadKey,
						}),
						rbac.T(rbac.Tuple{
							ResourceType: rbac.Api,
							ResourceID:   req.ApiId,
							Action:       rbac.ReadKey,
						}),
					),
					rbac.Or(
						rbac.T(rbac.Tuple{
							ResourceType: rbac.Api,
							ResourceID:   "*",
							Action:       rbac.ReadAPI,
						}),
						rbac.T(rbac.Tuple{
							ResourceType: rbac.Api,
							ResourceID:   req.ApiId,
							Action:       rbac.ReadAPI,
						}),
					),
				),
			),
		)
		if err != nil {
			return fault.Wrap(err,
				fault.WithDesc("unable to check permissions", "We're unable to check the permissions of your key."),
			)
		}

		if !permissionCheck.Valid {
			return fault.New("insufficient permissions",
				fault.WithCode(codes.Auth.Authorization.InsufficientPermissions.URN()),
				fault.WithDesc(permissionCheck.Message, permissionCheck.Message),
			)
		}

		// 4. Get API from database
		api, err := db.Query.FindApiById(ctx, svc.DB.RO(), req.ApiId)
		if err != nil {
			if db.IsNotFound(err) {
				return fault.New("api not found",
					fault.WithCode(codes.Data.Api.NotFound.URN()),
					fault.WithDesc("api not found", "The requested API does not exist or has been deleted."),
				)
			}
			return fault.Wrap(err,
				fault.WithCode(codes.App.Internal.ServiceUnavailable.URN()),
				fault.WithDesc("database error", "Failed to retrieve API information."),
			)
		}

		// Check if API is deleted
		if api.DeletedAtM.Valid {
			return fault.New("api not found",
				fault.WithCode(codes.Data.Api.NotFound.URN()),
				fault.WithDesc("api not found", "The requested API does not exist or has been deleted."),
			)
		}

		// Check if API belongs to the authorized workspace
		if api.WorkspaceID != auth.AuthorizedWorkspaceID {
			return fault.New("wrong workspace",
				fault.WithCode(codes.Data.Api.NotFound.URN()),
				fault.WithDesc("wrong workspace, masking as 404", "The requested API does not exist or has been deleted."),
			)
		}

		// Check if API is set up to handle keys
		if !api.KeyAuthID.Valid {
			return fault.New("api not set up for keys",
				fault.WithCode(codes.Data.Precondition.PreconditionFailed.URN()),
				fault.WithDesc("api not set up for keys", "The requested API is not set up to handle keys."),
			)
		}

		// 5. Query the keys
		var identityId string
		if req.ExternalId != nil && *req.ExternalId != "" {
			identity, err := db.Query.FindIdentityByExternalId(ctx, svc.DB.RO(), auth.AuthorizedWorkspaceID, *req.ExternalId)
			if err != nil {
				if !db.IsNotFound(err) {
					return fault.Wrap(err,
						fault.WithCode(codes.App.Internal.ServiceUnavailable.URN()),
						fault.WithDesc("database error", "Failed to retrieve identity information."),
					)
				}
				// If identity not found, return empty result
				return s.JSON(http.StatusOK, Response{
					Meta: openapi.Meta{
						RequestId: s.RequestID(),
					},
					Data: openapi.ApisListKeysResponseData{
						Keys:  []openapi.Key{},
						Total: 0,
					},
				})
			}
			identityId = identity.ID
		}

		// List keys
		keys, total, err := db.Query.ListKeysForApi(
			ctx,
			svc.DB.RO(),
			db.ListKeysForApiParams{
				KeyAuthID:   api.KeyAuthID.String,
				Limit:       int(req.Limit),
				Cursor:      req.Cursor,
				IdentityID:  identityId,
				WorkspaceID: auth.AuthorizedWorkspaceID,
			},
		)
		if err != nil {
			return fault.Wrap(err,
				fault.WithCode(codes.App.Internal.ServiceUnavailable.URN()),
				fault.WithDesc("database error", "Failed to retrieve keys."),
			)
		}

		// If user requested decryption, check permissions and decrypt
		var plaintextMap map[string]string
		if req.Decrypt != nil && *req.Decrypt {
			decryptPermission, err := svc.Permissions.Check(
				ctx,
				auth.KeyID,
				rbac.Or(
					rbac.T(rbac.Tuple{
						ResourceType: rbac.Api,
						ResourceID:   "*",
						Action:       rbac.DecryptKey,
					}),
					rbac.T(rbac.Tuple{
						ResourceType: rbac.Api,
						ResourceID:   req.ApiId,
						Action:       rbac.DecryptKey,
					}),
				),
			)
			if err != nil {
				return fault.Wrap(err,
					fault.WithDesc("unable to check decrypt permissions", "We're unable to check the decrypt permissions of your key."),
				)
			}

			if !decryptPermission.Valid {
				return fault.New("insufficient permissions to decrypt keys",
					fault.WithCode(codes.Auth.Authorization.InsufficientPermissions.URN()),
					fault.WithDesc("insufficient permissions to decrypt keys", decryptPermission.Message),
				)
			}

			plaintextMap = make(map[string]string)
			for _, key := range keys {
				if key.Encrypted.Valid {
					plaintext, err := svc.Vault.Decrypt(ctx, key.WorkspaceID, key.Encrypted.String)
					if err != nil {
						svc.Logger.Warn("failed to decrypt key", "keyId", key.ID, "error", err)
						continue
					}
					plaintextMap[key.ID] = plaintext
				}
			}
		}

		// Transform keys into the response format
		responseKeys := make([]openapi.Key, 0, len(keys))
		for _, key := range keys {
			k := openapi.Key{
				Id:          key.ID,
				Start:       key.Start,
				ApiId:       api.ID,
				WorkspaceId: key.WorkspaceID,
			}

			if key.Name.Valid {
				k.Name = &key.Name.String
			}

			if key.OwnerID.Valid {
				k.OwnerId = &key.OwnerID.String
			}

			if key.Meta.Valid {
				var meta map[string]interface{}
				if err := key.Meta.Unmarshal(&meta); err == nil {
					k.Meta = meta
				}
			}

			if key.CreatedAtM.Valid {
				createdAt := key.CreatedAtM.Time.Format(http.TimeFormat)
				k.CreatedAt = &createdAt
			}

			if key.UpdatedAtM.Valid {
				updatedAt := key.UpdatedAtM.Time.Format(http.TimeFormat)
				k.UpdatedAt = &updatedAt
			}

			if key.Expires.Valid {
				expires := key.Expires.Time.Format(http.TimeFormat)
				k.Expires = &expires
			}

			if key.RatelimitLimit.Valid && key.RatelimitDuration.Valid {
				isAsync := key.RatelimitAsync.Valid && key.RatelimitAsync.Bool
				ratelimitType := "consistent"
				if isAsync {
					ratelimitType = "fast"
				}

				k.Ratelimit = &openapi.KeyRatelimit{
					Type:           ratelimitType,
					Limit:          int(key.RatelimitLimit.Int64),
					Duration:       int(key.RatelimitDuration.Int64),
					RefillRate:     int(key.RatelimitLimit.Int64),
					RefillInterval: int(key.RatelimitDuration.Int64),
				}
			}

			if key.Remaining.Valid {
				remaining := int(key.Remaining.Int64)
				k.Remaining = &remaining
			}

			if key.Environment.Valid {
				k.Environment = &key.Environment.String
			}

			// Add plaintext if available
			if plaintextMap != nil {
				if plaintext, ok := plaintextMap[key.ID]; ok {
					k.Plaintext = &plaintext
				}
			}

			// Add identity information if available
			if key.IdentityID.Valid {
				identity, err := db.Query.FindIdentityById(ctx, svc.DB.RO(), key.IdentityID.String)
				if err == nil {
					k.Identity = &openapi.KeyIdentity{
						Id:         identity.ID,
						ExternalId: identity.ExternalID,
					}

					if identity.Meta.Valid {
						var meta map[string]interface{}
						if err := identity.Meta.Unmarshal(&meta); err == nil {
							k.Identity.Meta = meta
						}
					}
				}
			}

			// Get permissions and roles for the key
			permissions, err := db.Query.FindPermissionsByKeyId(ctx, svc.DB.RO(), key.ID)
			if err == nil {
				permNames := make([]string, 0, len(permissions))
				for _, perm := range permissions {
					permNames = append(permNames, perm.Name)
				}
				k.Permissions = permNames
			}

			// Get roles for the key - would need to implement a similar query for roles
			// For now, just set an empty array
			k.Roles = []string{}

			responseKeys = append(responseKeys, k)
		}

		// Determine the cursor for the next page
		var nextCursor *string
		if len(keys) > 0 && len(keys) == int(req.Limit) {
			cursor := keys[len(keys)-1].ID
			nextCursor = &cursor
		}

		return s.JSON(http.StatusOK, Response{
			Meta: openapi.Meta{
				RequestId: s.RequestID(),
			},
			Data: openapi.ApisListKeysResponseData{
				Keys:   responseKeys,
				Total:  int(total),
				Cursor: nextCursor,
			},
		})
	})
}
