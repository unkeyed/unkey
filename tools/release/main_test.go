package main

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestHasStableTag(t *testing.T) {
	stable := plannedTag{service: "api", version: "v1.2.3"}
	rc := plannedTag{service: "api", version: "v1.2.3-rc.1"}

	require.True(t, hasStableTag([]plannedTag{stable}))
	require.True(t, hasStableTag([]plannedTag{rc, stable}))
	require.False(t, hasStableTag([]plannedTag{rc}))
	require.False(t, hasStableTag([]plannedTag{
		{service: "vault", version: "v0.4.0-beta"},
		{service: "api", version: "v1.2.3-rc.2"},
	}))
	require.False(t, hasStableTag(nil))
}

func TestUniqueServices(t *testing.T) {
	tags := []plannedTag{
		{service: "api", version: "v1.0.1"},
		{service: "vault", version: "v0.4.1"},
		{service: "api", version: "v1.0.1-rc.1"},
	}
	require.Equal(t, []string{"api", "vault"}, uniqueServices(tags))
	require.Nil(t, uniqueServices(nil))
}

func TestServicesFor(t *testing.T) {
	ranges := []serviceRange{
		{service: "api", baseline: "api/v1.0.0", hashes: map[string]bool{"a": true, "b": true}},
		{service: "vault", baseline: "vault/v0.4.0", hashes: map[string]bool{"b": true}},
	}
	require.Equal(t, []string{"api"}, servicesFor("a", ranges))
	require.Equal(t, []string{"api", "vault"}, servicesFor("b", ranges))
	require.Nil(t, servicesFor("missing", ranges))
}
