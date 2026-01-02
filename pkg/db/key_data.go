package db

import (
	"database/sql"
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
func ToKeyData[T KeyRow](row T) *KeyData {
	switch r := any(row).(type) {
	case FindLiveKeyByHashRow:
		return buildKeyData(&r)
	case *FindLiveKeyByHashRow:
		return buildKeyData(r)
	case FindLiveKeyByIDRow:
		return buildKeyDataFromID(&r)
	case *FindLiveKeyByIDRow:
		return buildKeyDataFromID(r)
	case ListLiveKeysByKeySpaceIDRow:
		return buildKeyDataFromKeySpace(&r)
	case *ListLiveKeysByKeySpaceIDRow:
		return buildKeyDataFromKeySpace(r)
	default:
		return nil
	}
}

func buildKeyDataFromID(r *FindLiveKeyByIDRow) *KeyData {
	hr := FindLiveKeyByHashRow(*r) // safe value copy
	return buildKeyData(&hr)
}

func buildKeyDataFromKeySpace(r *ListLiveKeysByKeySpaceIDRow) *KeyData {
	kd := &KeyData{
		Key: Key{
			Pk:                 r.Pk,
			ID:                 r.ID,
			KeyAuthID:          r.KeyAuthID,
			Hash:               r.Hash,
			Start:              r.Start,
			WorkspaceID:        r.WorkspaceID,
			ForWorkspaceID:     r.ForWorkspaceID,
			Name:               r.Name,
			OwnerID:            r.OwnerID,
			IdentityID:         r.IdentityID,
			Meta:               r.Meta,
			Expires:            r.Expires,
			CreatedAtM:         r.CreatedAtM,
			UpdatedAtM:         r.UpdatedAtM,
			DeletedAtM:         r.DeletedAtM,
			RefillDay:          r.RefillDay,
			RefillAmount:       r.RefillAmount,
			LastRefillAt:       r.LastRefillAt,
			Enabled:            r.Enabled,
			RemainingRequests:  r.RemainingRequests,
			RatelimitAsync:     r.RatelimitAsync,
			RatelimitLimit:     r.RatelimitLimit,
			RatelimitDuration:  r.RatelimitDuration,
			Environment:        r.Environment,
			PendingMigrationID: r.PendingMigrationID,
		},
		// nolint: exhaustruct
		Identity: nil,
		// nolint: exhaustruct
		Api: Api{}, // Empty Api since not in this query
		// nolint: exhaustruct
		KeyAuth: KeyAuth{}, // Empty KeyAuth since not in this query
		// nolint: exhaustruct
		Workspace: Workspace{}, // Empty Workspace since not in this query

		EncryptedKey:    r.EncryptedKey,
		EncryptionKeyID: r.EncryptionKeyID,
		Roles:           nil,
		Permissions:     nil,
		RolePermissions: nil,
		Ratelimits:      nil,
	}

	if r.IdentityID.Valid {
		//nolint:exhaustruct
		kd.Identity = &Identity{
			Pk:          r.Pk,
			ID:          r.IdentityID.String,
			ExternalID:  r.IdentityExternalID.String,
			WorkspaceID: r.WorkspaceID,
			Meta:        r.IdentityMeta,
		}
	}

	// Unmarshal JSON fields, silently ignoring errors
	roles, _ := UnmarshalNullableJSONTo[[]RoleInfo](r.Roles)
	kd.Roles = roles

	permissions, _ := UnmarshalNullableJSONTo[[]PermissionInfo](r.Permissions)
	kd.Permissions = permissions

	rolePermissions, _ := UnmarshalNullableJSONTo[[]PermissionInfo](r.RolePermissions)
	kd.RolePermissions = rolePermissions

	ratelimits, _ := UnmarshalNullableJSONTo[[]RatelimitInfo](r.Ratelimits)
	kd.Ratelimits = ratelimits

	return kd
}

func buildKeyData(r *FindLiveKeyByHashRow) *KeyData {
	//nolint:exhaustruct
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
		Roles:           nil,
		Permissions:     nil,
		RolePermissions: nil,
		Ratelimits:      nil,
	}

	if r.IdentityTableID.Valid {
		//nolint: exhaustruct
		kd.Identity = &Identity{
			ID:          r.IdentityTableID.String,
			ExternalID:  r.IdentityExternalID.String,
			WorkspaceID: r.WorkspaceID,
			Meta:        r.IdentityMeta,
		}
	}

	// Unmarshal JSON fields, silently ignoring errors
	roles, _ := UnmarshalNullableJSONTo[[]RoleInfo](r.Roles)
	kd.Roles = roles

	permissions, _ := UnmarshalNullableJSONTo[[]PermissionInfo](r.Permissions)
	kd.Permissions = permissions

	rolePermissions, _ := UnmarshalNullableJSONTo[[]PermissionInfo](r.RolePermissions)
	kd.RolePermissions = rolePermissions

	ratelimits, _ := UnmarshalNullableJSONTo[[]RatelimitInfo](r.Ratelimits)
	kd.Ratelimits = ratelimits

	return kd
}
