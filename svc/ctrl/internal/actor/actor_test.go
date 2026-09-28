package actor

import (
	"testing"

	"github.com/stretchr/testify/require"
	ctrlv1 "github.com/unkeyed/unkey/gen/proto/ctrl/v1"
	"github.com/unkeyed/unkey/pkg/auditlog"
)

func TestAuditTypeOIDCVercel(t *testing.T) {
	t.Parallel()

	require.Equal(t, auditlog.OIDCVercelActor, AuditType(ctrlv1.ActorType_ACTOR_TYPE_OIDC_VERCEL))
}
