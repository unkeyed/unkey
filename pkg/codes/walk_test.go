package codes

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCollectURNs_SingleCode(t *testing.T) {
	t.Parallel()

	got := CollectURNs(Frontline.Auth.InvalidKey)
	require.Equal(t, []URN{Frontline.Auth.InvalidKey.URN()}, got)
}

func TestCollectURNs_WalksNestedNamespace(t *testing.T) {
	t.Parallel()

	got := CollectURNs(Frontline.Auth)
	require.Contains(t, got, Frontline.Auth.InvalidKey.URN())
	require.Contains(t, got, Frontline.Auth.MissingCredentials.URN())
	require.Contains(t, got, Frontline.Auth.RateLimited.URN())
}

// TestCollectURNs_FrontlineRoot guards against the walker silently
// returning a tiny set if it stops recursing too early. The exact
// count is allowed to grow, but the namespace must contain at least
// the major leaves we use elsewhere.
func TestCollectURNs_FrontlineRoot(t *testing.T) {
	t.Parallel()

	got := CollectURNs(Frontline)
	require.GreaterOrEqual(t, len(got), 15,
		"Frontline namespace should expose at least ~15 URNs; "+
			"got %d — walker may be stopping early", len(got))

	required := []URN{
		Frontline.Auth.InvalidKey.URN(),
		Frontline.Proxy.DialTimeout.URN(),
		Frontline.Proxy.PeerFrontlineHostUnreachable.URN(),
		Frontline.Internal.InternalServerError.URN(),
		Frontline.Routing.NoRunningInstances.URN(),
	}
	for _, urn := range required {
		require.Contains(t, got, urn)
	}
}

func TestCollectURNs_NonStructIsEmpty(t *testing.T) {
	t.Parallel()

	require.Empty(t, CollectURNs(42))
	require.Empty(t, CollectURNs("hello"))
	require.Empty(t, CollectURNs(nil))
}
