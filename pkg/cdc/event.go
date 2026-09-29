package cdc

import binlog "vitess.io/vitess/go/vt/proto/binlogdata"

// Event contains either a FIELD/ROW change or a checkpoint token, never both.
type Event struct {
	Change      *binlog.VEvent
	ResumeToken []byte
}
