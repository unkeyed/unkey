package main

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestFindPortalContext(t *testing.T) {
	t.Run("legacy hostname contains thread ID", func(t *testing.T) {
		t.Setenv("AMP_THREAD_ID", "")

		threadID, domain, err := findPortalContext([]string{
			"https://t-01a00f14-9206-7742-bf90-7216a37cfc8e-p24324.onamp.dev/",
		})

		require.NoError(t, err)
		require.Equal(t, "T-01a00f14-9206-7742-bf90-7216a37cfc8e", threadID)
		require.Equal(t, "onamp.dev", domain)
	})

	t.Run("encoded hostname uses service thread ID", func(t *testing.T) {
		t.Setenv("AMP_THREAD_ID", "T-01a00f14-9206-7742-bf90-7216a37cfc8e")

		threadID, domain, err := findPortalContext([]string{
			"https://t-03gp4jnwivnge0aab4n5k12tq-p24324.onamp.dev/",
		})

		require.NoError(t, err)
		require.Equal(t, "T-01a00f14-9206-7742-bf90-7216a37cfc8e", threadID)
		require.Equal(t, "onamp.dev", domain)
	})
}

func TestFrontlineProxyPortalHostname(t *testing.T) {
	proxy := &frontlineProxy{
		publicHostname: "t-01a00f14-9206-7742-bf90-7216a37cfc8e-p25109.onamp.dev",
		portalDomain:   "onamp.dev",
	}

	testCases := []struct {
		name     string
		hostname string
		want     bool
	}{
		{name: "configured hostname", hostname: proxy.publicHostname, want: true},
		{name: "encoded portal hostname", hostname: "t-03gp4jnwivnge0aab4n5k12tq-p24324.onamp.dev", want: true},
		{name: "direct orb hostname", hostname: "24324-sandbox.e2b.app", want: true},
		{name: "different domain", hostname: "t-03gp4jnwivnge0aab4n5k12tq-p24324.example.com", want: false},
		{name: "domain suffix attack", hostname: "onamp.dev.example.com", want: false},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			require.Equal(t, testCase.want, proxy.isPortalHostname(testCase.hostname))
		})
	}
}
