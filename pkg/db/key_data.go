package db

import "database/sql"

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
	FindLiveKeyByHashRow | FindLiveKeyByIDRow | ListLiveKeysByKeySpaceIDRow | ListLiveKeysByKeySpaceIDsRow
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
	case ListLiveKeysByKeySpaceIDsRow:
		// The plural-keyspace query selects the same columns as the singular one,
		// so the generated rows are field-identical and convert directly.
		kr := ListLiveKeysByKeySpaceIDRow(r)
		return buildKeyDataFromKeySpace(&kr)
	case *ListLiveKeysByKeySpaceIDsRow:
		kr := ListLiveKeysByKeySpaceIDRow(*r)
		return buildKeyDataFromKeySpace(&kr)
	default:
		return nil
	}
}

func buildKeyDataFromID(r *FindLiveKeyByIDRow) *KeyData {
	//nolint:exhaustruct
	kd := &KeyData{
		Key: Key{
			ID:                r.KeyID,
			KeyAuthID:         r.KeyKeyAuthID,
			Hash:              r.KeyHash,
			Start:             r.KeyStart,
			WorkspaceID:       r.KeyWorkspaceID,
			ForWorkspaceID:    r.KeyForWorkspaceID,
			Name:              r.KeyName,
			IdentityID:        r.KeyIdentityID,
			Meta:              r.KeyMeta,
			Expires:           r.KeyExpires,
			CreatedAtM:        r.KeyCreatedAtM,
			UpdatedAtM:        r.KeyUpdatedAtM,
			RefillDay:         r.KeyRefillDay,
			RefillAmount:      r.KeyRefillAmount,
			Enabled:           r.KeyEnabled,
			RemainingRequests: r.KeyRemainingRequests,
			LastUsedAt:        r.KeyLastUsedAt,
		},
		Api: Api{ID: r.ApiID, Name: r.ApiName},
		KeyAuth: KeyAuth{
			ID:                 r.KeyAuthID,
			ProjectID:          r.KeyAuthProjectID,
			StoreEncryptedKeys: r.KeyAuthStoreEncryptedKeys,
			DefaultPrefix:      r.KeyAuthDefaultPrefix,
			DefaultBytes:       r.KeyAuthDefaultBytes,
		},
		Workspace:       Workspace{},
		EncryptedKey:    r.EncryptedKey,
		EncryptionKeyID: r.EncryptionKeyID,
	}

	populateFindLiveKeyRelationships(kd, r.KeyWorkspaceID, r.IdentityTableID,
		r.IdentityExternalID, r.IdentityMeta, r.Roles, r.Permissions, r.RolePermissions, r.Ratelimits)
	return kd
}

func buildKeyDataFromKeySpace(r *ListLiveKeysByKeySpaceIDRow) *KeyData {
	//nolint:exhaustruct
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
			Environment:        r.Environment,
			LastUsedAt:         r.LastUsedAt,
			PendingMigrationID: r.PendingMigrationID,
		},
		Identity:  nil,
		Api:       Api{},       // Empty Api since not in this query
		KeyAuth:   KeyAuth{},   // Empty KeyAuth since not in this query
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
			ID:                r.KeyID,
			KeyAuthID:         r.KeyKeyAuthID,
			Start:             r.KeyStart,
			WorkspaceID:       r.KeyWorkspaceID,
			Name:              r.KeyName,
			Meta:              r.KeyMeta,
			Expires:           r.KeyExpires,
			CreatedAtM:        r.KeyCreatedAtM,
			UpdatedAtM:        r.KeyUpdatedAtM,
			RefillDay:         r.KeyRefillDay,
			RefillAmount:      r.KeyRefillAmount,
			Enabled:           r.KeyEnabled,
			RemainingRequests: r.KeyRemainingRequests,
			LastUsedAt:        r.KeyLastUsedAt,
		},
		Api: Api{ID: r.ApiID},
		KeyAuth: KeyAuth{
			ID:                 r.KeyAuthID,
			ProjectID:          r.KeyAuthProjectID,
			StoreEncryptedKeys: r.KeyAuthStoreEncryptedKeys,
		},
		Workspace:       Workspace{},
		EncryptedKey:    r.EncryptedKey,
		EncryptionKeyID: r.EncryptionKeyID,
		Roles:           nil,
		Permissions:     nil,
		RolePermissions: nil,
		Ratelimits:      nil,
	}

	populateFindLiveKeyRelationships(kd, r.KeyWorkspaceID, r.IdentityTableID,
		r.IdentityExternalID, r.IdentityMeta, r.Roles, r.Permissions, r.RolePermissions, r.Ratelimits)
	return kd
}

func populateFindLiveKeyRelationships(kd *KeyData, workspaceID string, identityID, identityExternalID sql.NullString,
	identityMeta []byte, roles, permissions, rolePermissions, ratelimits interface{},
) {
	if identityID.Valid {
		//nolint:exhaustruct
		kd.Identity = &Identity{
			ID:          identityID.String,
			ExternalID:  identityExternalID.String,
			WorkspaceID: workspaceID,
			Meta:        identityMeta,
		}
	}

	// Unmarshal JSON fields, silently ignoring errors
	kd.Roles, _ = UnmarshalNullableJSONTo[[]RoleInfo](roles)

	kd.Permissions, _ = UnmarshalNullableJSONTo[[]PermissionInfo](permissions)

	kd.RolePermissions, _ = UnmarshalNullableJSONTo[[]PermissionInfo](rolePermissions)

	kd.Ratelimits, _ = UnmarshalNullableJSONTo[[]RatelimitInfo](ratelimits)
}
