package db

import (
	"database/sql"

	"github.com/unkeyed/unkey/go/pkg/otel/logging"
)

// KeyData represents the complete data for a key including all relationships
type KeyData struct {
	Key             Key
	Api             Api
	KeyAuth         KeyAuth
	Workspace       Workspace
	Identity        *Identity // Is optional
	EncryptedKey    sql.NullString
	EncryptionKeyID sql.NullString
	Roles           []RoleInfo
	Permissions     []PermissionInfo // Direct permissions attached to the key
	RolePermissions []PermissionInfo // Permissions inherited from roles
	Ratelimits      []RatelimitInfo
}

// KeyRow constraint for types that can be converted to KeyData
type KeyRow interface {
	FindLiveKeyByHashRow | FindLiveKeyByIDRow | ListLiveKeysByKeySpaceIDRow
}

// ToKeyData converts either query result into KeyData using generics
func ToKeyData[T KeyRow](row T, logger logging.Logger) *KeyData {
	switch r := any(row).(type) {
	case FindLiveKeyByHashRow:
		return buildKeyData(&r, logger)
	case *FindLiveKeyByHashRow:
		return buildKeyData(r, logger)
	case FindLiveKeyByIDRow:
		return buildKeyDataFromID(&r, logger)
	case *FindLiveKeyByIDRow:
		return buildKeyDataFromID(r, logger)
	case ListLiveKeysByKeySpaceIDRow:
		return buildKeyDataFromKeySpace(&r, logger)
	case *ListLiveKeysByKeySpaceIDRow:
		return buildKeyDataFromKeySpace(r, logger)
	default:
		return nil
	}
}

func buildKeyDataFromID(r *FindLiveKeyByIDRow, logger logging.Logger) *KeyData {
	hr := FindLiveKeyByHashRow(*r) // safe value copy
	return buildKeyData(&hr, logger)
}

func buildKeyDataFromKeySpace(r *ListLiveKeysByKeySpaceIDRow, logger logging.Logger) *KeyData {
	kd := &KeyData{
		Key: Key{
			ID:                r.ID,
			KeyAuthID:         r.KeyAuthID,
			Hash:              r.Hash,
			Start:             r.Start,
			WorkspaceID:       r.WorkspaceID,
			ForWorkspaceID:    r.ForWorkspaceID,
			Name:              r.Name,
			OwnerID:           r.OwnerID,
			IdentityID:        r.IdentityID,
			Meta:              r.Meta,
			Expires:           r.Expires,
			CreatedAtM:        r.CreatedAtM,
			UpdatedAtM:        r.UpdatedAtM,
			DeletedAtM:        r.DeletedAtM,
			RefillDay:         r.RefillDay,
			RefillAmount:      r.RefillAmount,
			LastRefillAt:      r.LastRefillAt,
			Enabled:           r.Enabled,
			RemainingRequests: r.RemainingRequests,
			RatelimitAsync:    r.RatelimitAsync,
			RatelimitLimit:    r.RatelimitLimit,
			RatelimitDuration: r.RatelimitDuration,
			Environment:       r.Environment,
		},
		Api:             Api{},       // Empty Api since not in this query
		KeyAuth:         KeyAuth{},   // Empty KeyAuth since not in this query
		Workspace:       Workspace{}, // Empty Workspace since not in this query
		EncryptedKey:    r.EncryptedKey,
		EncryptionKeyID: r.EncryptionKeyID,
		Roles:           UnmarshalNullableJSONTo[[]RoleInfo](r.Roles, logger),
		Permissions:     UnmarshalNullableJSONTo[[]PermissionInfo](r.Permissions, logger),
		RolePermissions: UnmarshalNullableJSONTo[[]PermissionInfo](r.RolePermissions, logger),
		Ratelimits:      UnmarshalNullableJSONTo[[]RatelimitInfo](r.Ratelimits, logger),
	}

	if r.IdentityID.Valid {
		kd.Identity = &Identity{
			ID:          r.IdentityID.String,
			ExternalID:  r.IdentityExternalID.String,
			WorkspaceID: r.WorkspaceID,
			Meta:        r.IdentityMeta,
		}
	}

	return kd
}

func buildKeyData(r *FindLiveKeyByHashRow, logger logging.Logger) *KeyData {
	kd := &KeyData{
		Key: Key{
			ID:                r.ID,
			KeyAuthID:         r.KeyAuthID,
			Hash:              r.Hash,
			Start:             r.Start,
			WorkspaceID:       r.WorkspaceID,
			ForWorkspaceID:    r.ForWorkspaceID,
			Name:              r.Name,
			OwnerID:           r.OwnerID,
			IdentityID:        r.IdentityID,
			Meta:              r.Meta,
			Expires:           r.Expires,
			CreatedAtM:        r.CreatedAtM,
			UpdatedAtM:        r.UpdatedAtM,
			DeletedAtM:        r.DeletedAtM,
			RefillDay:         r.RefillDay,
			RefillAmount:      r.RefillAmount,
			LastRefillAt:      r.LastRefillAt,
			Enabled:           r.Enabled,
			RemainingRequests: r.RemainingRequests,
			RatelimitAsync:    r.RatelimitAsync,
			RatelimitLimit:    r.RatelimitLimit,
			RatelimitDuration: r.RatelimitDuration,
			Environment:       r.Environment,
		},
		Api:             r.Api,
		KeyAuth:         r.KeyAuth,
		Workspace:       r.Workspace,
		EncryptedKey:    r.EncryptedKey,
		EncryptionKeyID: r.EncryptionKeyID,
		Roles:           UnmarshalNullableJSONTo[[]RoleInfo](r.Roles, logger),
		Permissions:     UnmarshalNullableJSONTo[[]PermissionInfo](r.Permissions, logger),
		RolePermissions: UnmarshalNullableJSONTo[[]PermissionInfo](r.RolePermissions, logger),
		Ratelimits:      UnmarshalNullableJSONTo[[]RatelimitInfo](r.Ratelimits, logger),
	}

	if r.IdentityTableID.Valid {
		kd.Identity = &Identity{
			ID:          r.IdentityTableID.String,
			ExternalID:  r.IdentityExternalID.String,
			WorkspaceID: r.WorkspaceID,
			Meta:        r.IdentityMeta,
		}
	}

	return kd
}
