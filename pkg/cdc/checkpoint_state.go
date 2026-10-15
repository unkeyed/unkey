package cdc

import (
	"encoding/json"
	"time"

	"github.com/unkeyed/unkey/pkg/clock"
	"google.golang.org/protobuf/encoding/protojson"
	binlog "vitess.io/vitess/go/vt/proto/binlogdata"
)

// checkpointState separates unfinished transactions from safe resume positions.
type checkpointState struct {
	pending        *binlog.VGtid
	committed      *binlog.VGtid
	committedAt    time.Time
	lastCheckpoint time.Time
	changed        bool
}

// advance updates progress after the caller has delivered the event.
// A heartbeat can send a checkpoint, but only for a finished transaction.
func (s *checkpointState) advance(event *binlog.VEvent, rules []Rule, clock clock.Clock, apply func(Event) error) error {
	flush := false
	switch {
	case event.Type == binlog.VEventType_FIELD || event.Type == binlog.VEventType_ROW:
		s.changed = s.changed || len(event.GetRowEvent().GetRowChanges()) > 0
	case event.Type == binlog.VEventType_VGTID:
		s.pending = event.Vgtid
	case event.Type == binlog.VEventType_COMMIT || event.Type == binlog.VEventType_DDL || event.Type == binlog.VEventType_OTHER:
		if s.pending != nil {
			s.committed = s.pending
			s.committedAt = commitTime(event, s.committed)
			s.pending = nil
			flush = s.changed
			s.changed = false
		}
	}
	if s.committed == nil || (!flush && clock.Now().Sub(s.lastCheckpoint) < 30*time.Second) {
		return nil
	}
	encoded, err := protojson.Marshal(s.committed)
	if err != nil {
		return err
	}
	next, err := json.Marshal(resumeToken{Rules: rules, Position: encoded})
	if err != nil {
		return err
	}
	if err := apply(Event{Change: nil, ResumeToken: next, CommitTime: s.committedAt, CopyCompleted: false, Heartbeat: false}); err != nil {
		return err
	}
	s.committed = nil
	s.lastCheckpoint = clock.Now()
	return nil
}

// commitTime is unknown for a position with several shards, because one
// shard's commit does not show that the others are caught up.
func commitTime(event *binlog.VEvent, position *binlog.VGtid) time.Time {
	if event.Timestamp <= 0 || len(position.GetShardGtids()) != 1 {
		return time.Time{}
	}
	return time.Unix(event.Timestamp, 0)
}
