package cdc

import binlog "vitess.io/vitess/go/vt/proto/binlogdata"

// Event contains either a FIELD/ROW change or a checkpoint token, never both.
// Forward delivers these events so relays can pass checkpoints to their consumers.
type Event struct {
	Change      *binlog.VEvent
	ResumeToken []byte
}
