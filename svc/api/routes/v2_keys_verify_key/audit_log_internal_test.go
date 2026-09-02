package handler

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/internal/services/keys"
	keysdb "github.com/unkeyed/unkey/internal/services/keys/db"
	"github.com/unkeyed/unkey/pkg/auditlog"
	"github.com/unkeyed/unkey/pkg/auth/principal"
	"github.com/unkeyed/unkey/pkg/batch"
	"github.com/unkeyed/unkey/pkg/zen"
)

// TestBufferAuditLog_SkipsNonRootPrincipal guarantees that only root key
// requests produce direct key verification audit events.
func TestBufferAuditLog_SkipsNonRootPrincipal(t *testing.T) {
	t.Parallel()

	sess := &zen.Session{}
	require.NoError(t, sess.Init(
		httptest.NewRecorder(),
		httptest.NewRequest(http.MethodPost, "/v2/keys.verifyKey", nil),
		0,
	))

	var eventCountFlushed atomic.Int64
	directAuditLogs := batch.New(batch.Config[auditlog.Event]{
		Name:          "key_verify_non_root_audit_test",
		Drop:          false,
		BatchSize:     1,
		BufferSize:    1,
		FlushInterval: time.Hour,
		Consumers:     1,
		Flush: func(_ context.Context, events []auditlog.Event) {
			eventCountFlushed.Add(int64(len(events)))
		},
	})
	t.Cleanup(func() {
		directAuditLogs.Close()
		require.Zero(t, eventCountFlushed.Load())
	})

	h := &Handler{DirectAuditLogs: directAuditLogs} //nolint:exhaustruct
	p := &principal.Principal{
		Version: principal.Version,
		Subject: principal.Subject{
			ID:   "user_123",
			Name: "User",
			Type: principal.SubjectTypeUser,
		},
		Type: principal.TypeJWT,
		Source: principal.JWTSource{
			Header:    nil,
			Payload:   nil,
			Signature: "",
		},
		WorkspaceID: "ws_123",
		Permissions: nil,
	}
	key := &keys.KeyVerifier{ //nolint:exhaustruct
		Key: keysdb.FindKeyForVerificationRow{ID: "key_123"}, //nolint:exhaustruct
	}

	h.bufferAuditLog(sess, p, key, 0)
}
