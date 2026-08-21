package db

import (
	"context"
	"fmt"
	"strings"
)

type CreateKeyBatchPermission struct {
	Permission *UpsertPermissionParams
	Link       InsertKeyPermissionParams
	Outbox     InsertClickhouseOutboxParams
}

type CreateKeyBatchParams struct {
	Project     *UpsertDefaultProjectParams
	Identity    *UpsertIdentityParams
	Key         InsertKeyParams
	Encryption  *InsertKeyEncryptionParams
	Ratelimits  []InsertKeyRatelimitParams
	Permissions []CreateKeyBatchPermission
	Roles       []InsertKeyRoleParams
	Outbox      []InsertClickhouseOutboxParams
}

// CreateKeyWithAuditBatch atomically creates a key and its related resources
// in one MySQL protocol exchange.
func (r *Replica) CreateKeyWithAuditBatch(ctx context.Context, params CreateKeyBatchParams) error {
	statements := make([]transactionBatchStatement, 0, 8+len(params.Permissions)*4+len(params.Roles)+len(params.Outbox))
	nextResult := 0
	var projectResult *transactionBatchResult
	if params.Project != nil {
		result := transactionBatchResult{index: nextResult}
		nextResult++
		projectResult = &result
		statements = append(statements,
			upsertDefaultProjectTransactionBatchStatement(*params.Project),
			findDefaultProjectForBatchTransactionBatchStatement(result, FindDefaultProjectForBatchParams{
				WorkspaceID: params.Project.WorkspaceID,
				Slug:        "default",
			}),
		)
	}

	var identityResult *transactionBatchResult
	if params.Identity != nil {
		result := transactionBatchResult{index: nextResult}
		nextResult++
		identityResult = &result
		insert := upsertIdentityTransactionBatchStatement(*params.Identity)
		if projectResult != nil {
			insert = insert.withResultArgument("project_id", *projectResult)
		}
		statements = append(statements,
			insert,
			findIdentityForBatchTransactionBatchStatement(result, FindIdentityForBatchParams{
				WorkspaceID: params.Identity.WorkspaceID,
				ExternalID:  params.Identity.ExternalID,
			}),
		)
	}

	permissionResults := make(map[string]transactionBatchResult, len(params.Permissions))
	for _, permission := range params.Permissions {
		if permission.Permission == nil {
			continue
		}
		result := transactionBatchResult{index: nextResult}
		nextResult++
		permissionResults[permission.Link.PermissionID] = result
		insert := upsertPermissionTransactionBatchStatement(*permission.Permission)
		if projectResult != nil {
			insert = insert.withResultArgument("project_id", *projectResult)
		}
		statements = append(statements,
			insert,
			findPermissionForBatchTransactionBatchStatement(result, FindPermissionForBatchParams{
				WorkspaceID: permission.Permission.WorkspaceID,
				Slug:        permission.Permission.Slug,
			}),
		)
	}

	insertKey := insertKeyTransactionBatchStatement(params.Key)
	if identityResult != nil {
		insertKey = insertKey.withResultArgument("identity_id", *identityResult)
	}
	statements = append(statements, insertKey)
	if params.Encryption != nil {
		statements = append(statements, insertKeyEncryptionTransactionBatchStatement(*params.Encryption))
	}
	if len(params.Ratelimits) > 0 {
		statements = append(statements, insertKeyRatelimitsTransactionBatchStatement(params.Ratelimits))
	}
	for _, permission := range params.Permissions {
		insert := insertKeyPermissionTransactionBatchStatement(permission.Link)
		if result, ok := permissionResults[permission.Link.PermissionID]; ok {
			insert = insert.withResultArgument("permission_id", result)
		}
		statements = append(statements, insert)
	}
	for _, role := range params.Roles {
		statements = append(statements, insertKeyRoleTransactionBatchStatement(role))
	}
	for _, permission := range params.Permissions {
		result, ok := permissionResults[permission.Link.PermissionID]
		if !ok {
			statements = append(statements, insertClickhouseOutboxTransactionBatchStatement(permission.Outbox))
			continue
		}
		outbox := InsertClickhouseOutboxWithResultTargetParams{
			Version:        permission.Outbox.Version,
			WorkspaceID:    permission.Outbox.WorkspaceID,
			EventID:        permission.Outbox.EventID,
			Payload:        permission.Outbox.Payload,
			ResultTargetID: nil,
			CreatedAt:      permission.Outbox.CreatedAt,
		}
		statements = append(statements,
			insertClickhouseOutboxWithResultTargetTransactionBatchStatement(outbox).withResultArgument("result_target_id", result),
		)
	}
	for _, outbox := range params.Outbox {
		statements = append(statements, insertClickhouseOutboxTransactionBatchStatement(outbox))
	}
	return r.executeTransactionBatch(ctx, "CreateKeyWithAuditBatch", statements)
}

func insertKeyRatelimitsTransactionBatchStatement(args []InsertKeyRatelimitParams) transactionBatchStatement {
	valueClauses := make([]string, len(args))
	queryArgs := make([]transactionBatchArgument, 0, len(args)*8+1)
	for i, arg := range args {
		valueClauses[i] = "(?, ?, ?, ?, ?, ?, ?, ?)"
		queryArgs = append(queryArgs,
			transactionBatchArgument{name: "id", value: arg.ID, result: nil},
			transactionBatchArgument{name: "workspace_id", value: arg.WorkspaceID, result: nil},
			transactionBatchArgument{name: "key_id", value: arg.KeyID, result: nil},
			transactionBatchArgument{name: "name", value: arg.Name, result: nil},
			transactionBatchArgument{name: "limit", value: arg.Limit, result: nil},
			transactionBatchArgument{name: "duration", value: arg.Duration, result: nil},
			transactionBatchArgument{name: "auto_apply", value: arg.AutoApply, result: nil},
			transactionBatchArgument{name: "created_at", value: arg.CreatedAt, result: nil},
		)
	}
	queryArgs = append(queryArgs, transactionBatchArgument{name: "updated_at", value: args[0].UpdatedAt, result: nil})
	return transactionBatchStatement{
		query:  fmt.Sprintf(bulkInsertKeyRatelimit, strings.Join(valueClauses, ", ")),
		args:   queryArgs,
		result: nil,
	}
}
