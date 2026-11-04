package v2_ratelimit_multi_limit

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/auditlog"
	"github.com/unkeyed/unkey/pkg/auth/principal"
	"github.com/unkeyed/unkey/pkg/batch"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/pkg/zen"
)

// TestBufferAuditLogs_SkipsNonRootPrincipal guarantees that only root key
// requests produce direct rate limit audit events.
func TestBufferAuditLogs_SkipsNonRootPrincipal(t *testing.T) {
	t.Parallel()

	sess := &zen.Session{}
	require.NoError(t, sess.Init(
		httptest.NewRecorder(),
		httptest.NewRequest(http.MethodPost, "/v2/ratelimit.multiLimit", nil),
		0,
	))

	var eventCountFlushed atomic.Int64
	auditLogBuffer := batch.New(batch.Config[auditlog.Event]{
		Name:          "ratelimit_multi_limit_non_root_audit_test",
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
		auditLogBuffer.Close()
		require.Zero(t, eventCountFlushed.Load())
	})

	workspaceID := uid.New(uid.WorkspacePrefix)
	h := &Handler{AuditLogBuffer: auditLogBuffer} //nolint:exhaustruct
	p := &principal.Principal{
		Version: principal.Version,
		Subject: principal.Subject{
			ID:   uid.New("user"),
			Name: "User",
			Type: principal.SubjectTypeUser,
		},
		Type: principal.TypeJWT,
		Source: principal.JWTSource{
			Header:    nil,
			Payload:   nil,
			Signature: "",
		},
		AuthorizedWorkspaceID: workspaceID,
		Permissions:           nil,
	}
	check := checkMeta{ //nolint:exhaustruct
		namespace: db.FindRatelimitNamespace{ //nolint:exhaustruct
			ID:          uid.New(uid.RatelimitNamespacePrefix),
			WorkspaceID: workspaceID,
			Name:        "namespace",
		},
	}

	h.bufferAuditLogs(sess, p, []checkMeta{check, check}, 0)
}
