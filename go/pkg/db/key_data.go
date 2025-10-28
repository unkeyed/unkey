package db

import (
	"database/sql"
	"encoding/json"
)

// KeyData represents the complete data for a key including all relationships
type KeyData struct {
	Key             Key
	Api             Api
	KeyAuth         KeyAuth
	Workspace       Workspace
	Identity        *Identity // Is optional
	KeyCredits      *Credit   // Credits associated with the key
	IdentityCredits *Credit   // Credits associated with the identity
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
	} //nolint:exhaustruct

	if r.IdentityTableID.Valid {
		kd.Identity = &Identity{
			ID:          r.IdentityTableID.String,
			ExternalID:  r.IdentityExternalID.String,
			WorkspaceID: r.WorkspaceID,
			Meta:        r.IdentityMeta,
		}
	}

	// Populate key credits if they exist
	if r.CreditID.Valid {
		kd.KeyCredits = &Credit{
			ID:           r.CreditID.String,
			WorkspaceID:  r.WorkspaceID,
			KeyID:        sql.NullString{Valid: true, String: r.ID},
			IdentityID:   sql.NullString{Valid: false},
			Remaining:    r.CreditRemaining.Int32,
			RefillDay:    r.CreditRefillDay,
			RefillAmount: r.CreditRefillAmount,
			RefilledAt:   r.CreditRefilledAt,
		}
	}

	// Populate identity credits if they exist
	if r.IdentityCreditID.Valid {
		kd.IdentityCredits = &Credit{
			ID:           r.IdentityCreditID.String,
			WorkspaceID:  r.WorkspaceID,
			KeyID:        sql.NullString{Valid: false},
			IdentityID:   r.IdentityID,
			Remaining:    r.IdentityCreditRemaining.Int32,
			RefillDay:    r.IdentityCreditRefillDay,
			RefillAmount: r.IdentityCreditRefillAmount,
			RefilledAt:   r.IdentityCreditRefilledAt,
		}
	}

	// It's fine to fail here
	if roleBytes, ok := r.Roles.([]byte); ok && roleBytes != nil {
		_ = json.Unmarshal(roleBytes, &kd.Roles) // Ignore error, default to empty array
	}
	if permissionsBytes, ok := r.Permissions.([]byte); ok && permissionsBytes != nil {
		_ = json.Unmarshal(permissionsBytes, &kd.Permissions) // Ignore error, default to empty array
	}
	if rolePermissionsBytes, ok := r.RolePermissions.([]byte); ok && rolePermissionsBytes != nil {
		_ = json.Unmarshal(rolePermissionsBytes, &kd.RolePermissions) // Ignore error, default to empty array
	}
	if ratelimitsBytes, ok := r.Ratelimits.([]byte); ok && ratelimitsBytes != nil {
		_ = json.Unmarshal(ratelimitsBytes, &kd.Ratelimits) // Ignore error, default to empty array
	}

	return kd
}

func buildKeyDataFromKeySpace(r *ListLiveKeysByKeySpaceIDRow) *KeyData {
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
			ID:          r.IdentityID.String,
			ExternalID:  r.IdentityExternalID.String,
			WorkspaceID: r.WorkspaceID,
			Meta:        r.IdentityMeta,
		}
	}

	// It's fine to fail here
	if roleBytes, ok := r.Roles.([]byte); ok && roleBytes != nil {
		_ = json.Unmarshal(roleBytes, &kd.Roles) // Ignore error, default to empty array
	}
	if permissionsBytes, ok := r.Permissions.([]byte); ok && permissionsBytes != nil {
		_ = json.Unmarshal(permissionsBytes, &kd.Permissions) // Ignore error, default to empty array
	}
	if rolePermissionsBytes, ok := r.RolePermissions.([]byte); ok && rolePermissionsBytes != nil {
		_ = json.Unmarshal(rolePermissionsBytes, &kd.RolePermissions) // Ignore error, default to empty array
	}
	if ratelimitsBytes, ok := r.Ratelimits.([]byte); ok && ratelimitsBytes != nil {
		_ = json.Unmarshal(ratelimitsBytes, &kd.Ratelimits) // Ignore error, default to empty array
	}

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

	// Populate key credits if they exist
	if r.CreditID.Valid {
		kd.KeyCredits = &Credit{
			ID:           r.CreditID.String,
			WorkspaceID:  r.WorkspaceID,
			KeyID:        sql.NullString{Valid: true, String: r.ID},
			IdentityID:   sql.NullString{Valid: false},
			Remaining:    r.CreditRemaining.Int32,
			RefillDay:    r.CreditRefillDay,
			RefillAmount: r.CreditRefillAmount,
			RefilledAt:   r.CreditRefilledAt,
		}
	}

	// Populate identity credits if they exist
	if r.IdentityCreditID.Valid {
		kd.IdentityCredits = &Credit{
			ID:           r.IdentityCreditID.String,
			WorkspaceID:  r.WorkspaceID,
			KeyID:        sql.NullString{Valid: false},
			IdentityID:   r.IdentityID,
			Remaining:    r.IdentityCreditRemaining.Int32,
			RefillDay:    r.IdentityCreditRefillDay,
			RefillAmount: r.IdentityCreditRefillAmount,
			RefilledAt:   r.IdentityCreditRefilledAt,
		}
	}

	// It's fine to fail here
	if roleBytes, ok := r.Roles.([]byte); ok && roleBytes != nil {
		_ = json.Unmarshal(roleBytes, &kd.Roles) // Ignore error, default to empty array
	}
	if permissionsBytes, ok := r.Permissions.([]byte); ok && permissionsBytes != nil {
		_ = json.Unmarshal(permissionsBytes, &kd.Permissions) // Ignore error, default to empty array
	}
	if rolePermissionsBytes, ok := r.RolePermissions.([]byte); ok && rolePermissionsBytes != nil {
		_ = json.Unmarshal(rolePermissionsBytes, &kd.RolePermissions) // Ignore error, default to empty array
	}
	if ratelimitsBytes, ok := r.Ratelimits.([]byte); ok && ratelimitsBytes != nil {
		_ = json.Unmarshal(ratelimitsBytes, &kd.Ratelimits) // Ignore error, default to empty array
	}

	return kd
}
