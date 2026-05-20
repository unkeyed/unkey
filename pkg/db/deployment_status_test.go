package db

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// TestTerminalDeploymentStatusesMatchesIsTerminal guards against the two
// declarations drifting. If a status is added to the enum it must be
// classified in IsTerminal; if it's classified as terminal it must also
// appear in TerminalDeploymentStatuses (and vice versa).
func TestTerminalDeploymentStatusesMatchesIsTerminal(t *testing.T) {
	all := []DeploymentsStatus{
		DeploymentsStatusPending,
		DeploymentsStatusStarting,
		DeploymentsStatusBuilding,
		DeploymentsStatusDeploying,
		DeploymentsStatusNetwork,
		DeploymentsStatusFinalizing,
		DeploymentsStatusReady,
		DeploymentsStatusFailed,
		DeploymentsStatusSkipped,
		DeploymentsStatusAwaitingApproval,
		DeploymentsStatusStopped,
		DeploymentsStatusSuperseded,
		DeploymentsStatusCancelled,
	}

	inSlice := make(map[DeploymentsStatus]bool, len(TerminalDeploymentStatuses))
	for _, s := range TerminalDeploymentStatuses {
		inSlice[s] = true
	}

	for _, s := range all {
		require.Equal(t, s.IsTerminal(), inSlice[s],
			"status %q: IsTerminal=%v but TerminalDeploymentStatuses membership=%v",
			s, s.IsTerminal(), inSlice[s])
	}
}
