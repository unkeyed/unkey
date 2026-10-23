package namespace

import (
	"encoding/hex"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/uid"
)

// The expected hash was computed outside Go. If it changes, every global
// limit loses its state once, so a change must be deliberate.
func TestGlobalFingerprintEncodingIsStable(t *testing.T) {
	fingerprint := GlobalFingerprint("ws_123", "ns_456", "user_789", time.Minute)
	require.Equal(t, "daabab6efd4ffcb0daab8878dce557feac9a97f89a48fc2a726111b792d8777e", hex.EncodeToString(fingerprint[:]))
}

func TestGlobalFingerprintSeparatesFields(t *testing.T) {
	workspaceID := uid.New(uid.WorkspacePrefix)
	namespaceID := uid.New(uid.RatelimitNamespacePrefix)

	require.NotEqual(t,
		GlobalFingerprint(workspaceID, namespaceID+"a", "bc", time.Minute),
		GlobalFingerprint(workspaceID, namespaceID+"ab", "c", time.Minute),
	)
	require.NotEqual(t,
		GlobalFingerprint(workspaceID, namespaceID, "user", time.Minute),
		GlobalFingerprint(workspaceID, namespaceID, "user", time.Hour),
	)
}
