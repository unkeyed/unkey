package cdc

import (
	"time"

	binlog "vitess.io/vitess/go/vt/proto/binlogdata"
)

// Event contains exactly one of a FIELD/ROW change, a checkpoint token, a copy
// completion, or a heartbeat.
type Event struct {
	Change      *binlog.VEvent
	ResumeToken []byte
	// CommitTime is when the checkpoint's last transaction committed, in
	// whole seconds of the source's clock. It is zero when unknown: during the
	// initial copy and for keyspaces with more than one shard.
	CommitTime time.Time
	// CopyCompleted reports that a watch started with an empty token has copied
	// every matching row. Later changes are live.
	CopyCompleted bool
	// Heartbeat reports that an idle upstream stream is still alive.
	Heartbeat bool
}
