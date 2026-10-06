package auditlogs

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/auditlog"
)

func decodeBucket(t *testing.T, payload json.RawMessage) string {
	t.Helper()
	var event auditlog.Event
	require.NoError(t, json.Unmarshal(payload, &event))
	return event.Bucket
}

func TestPrepareOutboxRows_Bucket(t *testing.T) {
	ctx := context.Background()
	base := auditlog.AuditLog{
		WorkspaceID: "ws_1",
		Event:       auditlog.KeyCreateEvent,
		ActorType:   auditlog.SystemActor,
		ActorID:     "system",
	}

	t.Run("defaults to unkey_mutations", func(t *testing.T) {
		rows, err := PrepareOutboxRows(ctx, []auditlog.AuditLog{base})
		require.NoError(t, err)
		require.Len(t, rows, 1)
		require.Equal(t, auditlog.BucketUnkeyMutations, decodeBucket(t, rows[0].Payload))
	})

	t.Run("honours back office bucket", func(t *testing.T) {
		log := base
		log.Bucket = auditlog.BucketBackoffice
		rows, err := PrepareOutboxRows(ctx, []auditlog.AuditLog{log})
		require.NoError(t, err)
		require.Len(t, rows, 1)
		require.Equal(t, auditlog.BucketBackoffice, decodeBucket(t, rows[0].Payload))
	})

	t.Run("rejects unknown bucket", func(t *testing.T) {
		log := base
		log.Bucket = "customer_chosen"
		_, err := PrepareOutboxRows(ctx, []auditlog.AuditLog{log})
		require.Error(t, err)
	})
}
