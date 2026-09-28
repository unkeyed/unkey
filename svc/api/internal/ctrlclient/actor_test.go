package ctrlclient

import (
	"testing"

	"github.com/stretchr/testify/require"
	ctrlv1 "github.com/unkeyed/unkey/gen/proto/ctrl/v1"
	"github.com/unkeyed/unkey/pkg/auth/principal"
)

func TestSubjectTypeOIDCVercel(t *testing.T) {
	t.Parallel()

	require.Equal(
		t,
		ctrlv1.ActorType_ACTOR_TYPE_OIDC_VERCEL,
		subjectType(principal.SubjectTypeOIDCVercel),
	)
}
