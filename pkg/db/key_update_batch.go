package db

import "context"

// UpdateKeyPermissionBatchWrite contains a permission insert and the audit row
// that is written only when that exact permission ID was created.
type UpdateKeyPermissionBatchWrite struct {
	Permission InsertPermissionForUpdateKeyParams
	Outbox     InsertClickhouseOutboxForPermissionUpdateKeyParams
}

// UpdateKeyBatchParams contains the ordered writes for one key update.
type UpdateKeyBatchParams struct {
	WorkspaceID        string
	WorkspaceLock      *LockWorkspaceForUpdateKeyParams
	Project            *InsertDefaultProjectForUpdateKeyParams
	Identity           *InsertIdentityForUpdateKeyParams
	Permissions        []UpdateKeyPermissionBatchWrite
	Update             UpdateKeyParams
	RatelimitDelete    *DeleteKeyRatelimitsForUpdateKeyParams
	Ratelimits         []InsertKeyRatelimitParams
	ReplacePermissions bool
	ReplaceRoles       bool
	KeyPermissions     []InsertKeyPermissionBySlugForUpdateKeyParams
	KeyRoles           []InsertKeyRoleParams
	Outboxes           []InsertClickhouseOutboxParams
}

// UpdateKeyBatch atomically applies every write for one key update in one
// MySQL protocol exchange.
func (r *Replica) UpdateKeyBatch(ctx context.Context, params UpdateKeyBatchParams) error {
	statements := make([]transactionBatchStatement, 0,
		4+len(params.Permissions)*2+len(params.Ratelimits)+len(params.KeyPermissions)+len(params.KeyRoles)+len(params.Outboxes),
	)
	if params.WorkspaceLock != nil {
		statements = append(statements, lockWorkspaceForUpdateKeyTransactionBatchStatement(*params.WorkspaceLock))
	}
	if params.Project != nil {
		statements = append(statements, insertDefaultProjectForUpdateKeyTransactionBatchStatement(*params.Project))
	}
	if params.Identity != nil {
		statements = append(statements, insertIdentityForUpdateKeyTransactionBatchStatement(*params.Identity))
	}
	for _, permission := range params.Permissions {
		statements = append(statements, insertPermissionForUpdateKeyTransactionBatchStatement(permission.Permission))
	}

	statements = append(statements, updateKeyTransactionBatchStatement(params.Update))
	if params.RatelimitDelete != nil {
		statements = append(statements, deleteKeyRatelimitsForUpdateKeyTransactionBatchStatement(*params.RatelimitDelete))
	}
	for _, ratelimit := range params.Ratelimits {
		statements = append(statements, insertKeyRatelimitTransactionBatchStatement(ratelimit))
	}

	relationParams := DeleteKeyPermissionsAndRolesForUpdateKeyParams{
		KeyID:       params.Update.ID,
		WorkspaceID: params.WorkspaceID,
	}
	switch {
	case params.ReplacePermissions && params.ReplaceRoles:
		statements = append(statements, deleteKeyPermissionsAndRolesForUpdateKeyTransactionBatchStatement(relationParams))
	case params.ReplacePermissions:
		statements = append(statements, deleteKeyPermissionsForUpdateKeyTransactionBatchStatement(DeleteKeyPermissionsForUpdateKeyParams(relationParams)))
	case params.ReplaceRoles:
		statements = append(statements, deleteKeyRolesForUpdateKeyTransactionBatchStatement(DeleteKeyRolesForUpdateKeyParams(relationParams)))
	}
	for _, permission := range params.KeyPermissions {
		statements = append(statements, insertKeyPermissionBySlugForUpdateKeyTransactionBatchStatement(permission))
	}
	for _, role := range params.KeyRoles {
		statements = append(statements, insertKeyRoleTransactionBatchStatement(role))
	}
	for _, permission := range params.Permissions {
		statements = append(statements, insertClickhouseOutboxForPermissionUpdateKeyTransactionBatchStatement(permission.Outbox))
	}
	for _, outbox := range params.Outboxes {
		statements = append(statements, insertClickhouseOutboxTransactionBatchStatement(outbox))
	}

	return r.executeTransactionBatch(ctx, "UpdateKeyBatch", statements)
}
