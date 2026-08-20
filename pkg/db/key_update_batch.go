package db

import "context"

// UpdateKeyWithAuditBatch atomically updates a key and writes its audit outbox
// event in one MySQL protocol exchange.
func (r *Replica) UpdateKeyWithAuditBatch(
	ctx context.Context,
	update UpdateKeyParams,
	outbox InsertClickhouseOutboxParams,
) error {
	return r.executeTransactionBatch(ctx, "UpdateKeyWithAuditBatch", []transactionBatchStatement{
		updateKeyTransactionBatchStatement(update),
		insertClickhouseOutboxTransactionBatchStatement(outbox),
	})
}
