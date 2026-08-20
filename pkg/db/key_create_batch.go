package db

import "context"

// CreateKeyWithAuditBatch atomically creates a key and writes its audit outbox
// event in one MySQL protocol exchange.
func (r *Replica) CreateKeyWithAuditBatch(
	ctx context.Context,
	key InsertKeyParams,
	outbox InsertClickhouseOutboxParams,
) error {
	return r.executeTransactionBatch(ctx, "CreateKeyWithAuditBatch", []transactionBatchStatement{
		insertKeyTransactionBatchStatement(key),
		insertClickhouseOutboxTransactionBatchStatement(outbox),
	})
}
