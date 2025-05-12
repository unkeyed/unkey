package handler

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http"

	"github.com/unkeyed/unkey/go/apps/api/openapi"
	vaultv1 "github.com/unkeyed/unkey/go/gen/proto/vault/v1"
	"github.com/unkeyed/unkey/go/internal/services/caches"
	"github.com/unkeyed/unkey/go/internal/services/keys"
	"github.com/unkeyed/unkey/go/internal/services/permissions"
	"github.com/unkeyed/unkey/go/pkg/codes"
	"github.com/unkeyed/unkey/go/pkg/db"
	"github.com/unkeyed/unkey/go/pkg/fault"
	"github.com/unkeyed/unkey/go/pkg/otel/logging"
	"github.com/unkeyed/unkey/go/pkg/ptr"
	"github.com/unkeyed/unkey/go/pkg/rbac"
	"github.com/unkeyed/unkey/go/pkg/vault"
	"github.com/unkeyed/unkey/go/pkg/zen"
)

type Request = openapi.V2ApisListKeysRequestBody
type Response = openapi.V2ApisListKeysResponseBody

type Services struct {
	Logger      logging.Logger
	DB          db.Database
	Keys        keys.KeyService
	Permissions permissions.PermissionService
	Vault       *vault.Service
	Caches      caches.Caches
}

func New(svc Services) zen.Route {
	return zen.NewRoute("POST", "/v2/apis.listKeys", func(ctx context.Context, s *zen.Session) error {

		auth, err := svc.Keys.VerifyRootKey(ctx, s)
		if err != nil {
			return err
		}

		var req Request
		err = s.BindBody(&req)
		if err != nil {
			return fault.Wrap(err,
				fault.WithDesc("invalid request body", "The request body is invalid."),
			)
		}

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
		// Check if API belongs to the authorized workspace
		if api.WorkspaceID != auth.AuthorizedWorkspaceID {
			return fault.New("wrong workspace",
				fault.WithCode(codes.Data.Api.NotFound.URN()),
				fault.WithDesc("wrong workspace, masking as 404", "The requested API does not exist or has been deleted."),
			)
		}
		// Check if API is deleted
		if api.DeletedAtM.Valid {
			return fault.New("api not found",
				fault.WithCode(codes.Data.Api.NotFound.URN()),
				fault.WithDesc("api not found", "The requested API does not exist or has been deleted."),
			)
		}

		// Check if API is set up to handle keys
		if !api.KeyAuthID.Valid {
			return fault.New("api not set up for keys",
				fault.WithCode(codes.App.Precondition.PreconditionFailed.URN()),
				fault.WithDesc("api not set up for keys", "The requested API is not set up to handle keys."),
			)
		}

		// 5. Query the keys
		var identityId string
		if req.ExternalId != nil && *req.ExternalId != "" {
			identity, err := db.Query.FindIdentityByExternalID(ctx, svc.DB.RO(), db.FindIdentityByExternalIDParams{
				WorkspaceID: auth.AuthorizedWorkspaceID,
				ExternalID:  *req.ExternalId,
			})
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
					Data: []openapi.KeyResponse{},
				})
			}
			identityId = identity.ID
		}

		limit := ptr.SafeDeref(req.Limit, 100)
		cursor := ptr.SafeDeref(req.Cursor, "")
		// List keys
		keys, err := db.Query.FindKeysByKeyAuthId(
			ctx,
			svc.DB.RO(),
			db.FindKeysByKeyAuthIdParams{
				KeyAuthID:  api.KeyAuthID.String,
				Limit:      int32(limit + 1),
				IDCursor:   cursor,
				IdentityID: sql.NullString{Valid: identityId != "", String: identityId},
			},
		)
		if err != nil {
			return fault.Wrap(err,
				fault.WithCode(codes.App.Internal.ServiceUnavailable.URN()),
				fault.WithDesc("database error", "Failed to retrieve keys."),
			)
		}

		// If user requested decryption, check permissions and decrypt
		plaintextMap := make(map[string]string)
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

			for _, key := range keys {
				if key.EncryptedKey.Valid && key.EncryptionKeyID.Valid {
					decrypted, err := svc.Vault.Decrypt(ctx, &vaultv1.DecryptRequest{
						Keyring:   key.Key.WorkspaceID,
						Encrypted: key.EncryptedKey.String,
					})
					if err != nil {
						svc.Logger.Error("failed to decrypt key",
							"keyId", key.Key.ID,
							"error", err,
						)
						continue
					}
					plaintextMap[key.Key.ID] = decrypted.GetPlaintext()
				}
			}
		}

		// Transform keys into the response format
		responseData := make([]openapi.KeyResponse, 0, len(keys))
		for _, key := range keys {
			k := openapi.KeyResponse{
				KeyId:     key.Key.ID,
				Start:     key.Key.Start,
				CreatedAt: key.Key.CreatedAtM,
			}

			if key.Key.Name.Valid {
				k.Name = ptr.P(key.Key.Name.String)
			}

			if key.Key.Meta.Valid {
				err = json.Unmarshal([]byte(key.Key.Meta.String), &k.Meta)
				if err != nil {
					svc.Logger.Error("unable to unmarshal key metadata",
						"keyId", key.Key.ID,
						"error", err,
					)
				}
			}

			if key.Key.UpdatedAtM.Valid {
				k.UpdatedAt = ptr.P(key.Key.UpdatedAtM.Int64)
			}

			if key.Key.Expires.Valid {
				k.Expires = ptr.P(key.Key.Expires.Time.UnixMilli())
			}

			if key.Key.RemainingRequests.Valid {
				k.Credits = &openapi.KeyCredits{
					Remaining: int(key.Key.RemainingRequests.Int32),
				}
				if key.Key.RefillAmount.Valid {
					k.Credits.Refill = &openapi.KeyCreditsRefill{
						Amount:    int(key.Key.RemainingRequests.Int32),
						RefillDay: ptr.P(int(key.Key.RefillDay.Int16)),
					}
					if key.Key.LastRefillAt.Valid {
						k.Credits.Refill.LastRefillAt = ptr.P(key.Key.LastRefillAt.Time.UnixMilli())
					}
				}
			}

			// Add plaintext if available
			if plaintextMap != nil {
				if plaintext, ok := plaintextMap[key.Key.ID]; ok {
					k.Plaintext = ptr.P(plaintext)
				}
			}

			// Add identity information if available
			if key.IdentityID.Valid {

				k.Identity = &openapi.Identity{
					ExternalId: key.ExternalID.String,
					Id:         key.IdentityID.String,
				}
				if key.IdentityMeta != nil && len(key.IdentityMeta) > 0 {
					err = json.Unmarshal(key.IdentityMeta, &k.Identity.Meta)
					if err != nil {
						return fault.Wrap(err, fault.WithCode(codes.App.Internal.UnexpectedError.URN()),
							fault.WithDesc("unable to unmarshal identity meta", "We encountered an error while trying to unmarshal the identity meta data."))
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
		if len(keys) > 0 && len(keys) >= limit {
			cursor := keys[len(keys)-1].Key.ID
			nextCursor = &cursor
		}

		return s.JSON(http.StatusOK, Response{
			Meta: openapi.Meta{
				RequestId: s.RequestID(),
			},
			Data: []openapi.KeyResponse{},
			Pagination: &openapi.Pagination{
				Cursor:  nextCursor,
				HasMore: ptr.P(len(keys) > limit),
			},
		})
	})
}
