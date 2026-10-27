package vaultreencryption

import (
	"context"

	"github.com/unkeyed/unkey/pkg/logger"
)

// batch shares cursor and failure handling across table-specific scanners.
// Only result leaves the Restate Run, so ciphertext stays out of the journal.
type batch struct {
	table  string
	result batchResult
}

// scanFailed preserves the cursor so a later invocation retries the same page.
func (b *batch) scanFailed() {
	logger.Error("ciphertext scan failed", "table", b.table)
	b.result.Failed++
}

// process advances past failed rows so one corrupt value cannot block the scan.
// The handler reports these failures after scanning all tables and withholds
// the heartbeat. Cancellation before an attempt returns false without advancing
// the cursor, leaving that row for a later invocation.
func (b *batch) process(ctx context.Context, pk uint64, reencrypt func() (int64, error)) bool {
	if ctx.Err() != nil {
		b.result.Failed++
		return false
	}
	b.result.Cursor.Pk = pk
	_, err := reencrypt()
	if err != nil {
		logger.Error("ciphertext re-encryption failed", "table", b.table, "pk", pk)
		b.result.Failed++
	}
	return true
}

// finish resets the cursor after a short page so the next sweep starts over.
// A full page keeps its cursor because more rows may exist or arrive during it.
func (b *batch) finish(count int) {
	if count < batchSize {
		b.result.Cursor = cursor{Pk: 0}
	}
}
