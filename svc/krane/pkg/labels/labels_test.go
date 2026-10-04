package labels

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/uid"
)

func TestMatches(t *testing.T) {
	want := New().ManagedByKrane().Component("private-dns")

	require.True(t, want.Matches(map[string]string{LabelKeyManagedBy: "krane", LabelKeyComponent: "private-dns", LabelKeyAppID: uid.New(uid.AppPrefix)}))
	require.False(t, want.Matches(map[string]string{LabelKeyManagedBy: "krane", LabelKeyComponent: "deployment"}))
	require.False(t, want.Matches(map[string]string{LabelKeyManagedBy: "krane"}))
	require.False(t, New().AppID("").Matches(nil), "an empty value must still be present")
	require.True(t, New().Matches(nil))
}
