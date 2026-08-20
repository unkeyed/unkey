package db

import (
	"context"
	"fmt"
	"strings"
)

// CreateKeyWithAuditBatch atomically creates a key and writes its audit outbox
// event in one MySQL protocol exchange.
func (r *Replica) CreateKeyWithAuditBatch(
	ctx context.Context,
	key InsertKeyParams,
	encryption *InsertKeyEncryptionParams,
	ratelimits []InsertKeyRatelimitParams,
	outbox InsertClickhouseOutboxParams,
) error {
	statements := []transactionBatchStatement{insertKeyTransactionBatchStatement(key)}
	if encryption != nil {
		statements = append(statements, insertKeyEncryptionTransactionBatchStatement(*encryption))
	}
	if len(ratelimits) > 0 {
		statements = append(statements, insertKeyRatelimitsTransactionBatchStatement(ratelimits))
	}
	statements = append(statements, insertClickhouseOutboxTransactionBatchStatement(outbox))
	return r.executeTransactionBatch(ctx, "CreateKeyWithAuditBatch", statements)
}

func insertKeyRatelimitsTransactionBatchStatement(args []InsertKeyRatelimitParams) transactionBatchStatement {
	valueClauses := make([]string, len(args))
	queryArgs := make([]any, 0, len(args)*8+1)
	for i, arg := range args {
		valueClauses[i] = "(?, ?, ?, ?, ?, ?, ?, ?)"
		queryArgs = append(queryArgs,
			arg.ID,
			arg.WorkspaceID,
			arg.KeyID,
			arg.Name,
			arg.Limit,
			arg.Duration,
			arg.AutoApply,
			arg.CreatedAt,
		)
	}
	queryArgs = append(queryArgs, args[0].UpdatedAt)
	return transactionBatchStatement{
		query: fmt.Sprintf(bulkInsertKeyRatelimit, strings.Join(valueClauses, ", ")),
		args:  queryArgs,
	}
}
