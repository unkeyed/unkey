package db

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// TestDeploymentStatusSetsMatchIsTerminal guards against the status-set
// declarations drifting from IsTerminal. If a status is added to the
// enum it must be classified in IsTerminal and appear in exactly one of
// TerminalDeploymentStatuses or ProgressingDeploymentStatuses.
func TestDeploymentStatusSetsMatchIsTerminal(t *testing.T) {
	inTerminal := make(map[DeploymentsStatus]bool, len(TerminalDeploymentStatuses))
	for _, s := range TerminalDeploymentStatuses {
		inTerminal[s] = true
	}
	inProgressing := make(map[DeploymentsStatus]bool, len(ProgressingDeploymentStatuses))
	for _, s := range ProgressingDeploymentStatuses {
		inProgressing[s] = true
	}

	for _, s := range AllDeploymentStatuses {
		require.Equal(t, s.IsTerminal(), inTerminal[s],
			"status %q: IsTerminal=%v but TerminalDeploymentStatuses membership=%v",
			s, s.IsTerminal(), inTerminal[s])
		require.Equal(t, !s.IsTerminal(), inProgressing[s],
			"status %q: IsTerminal=%v but ProgressingDeploymentStatuses membership=%v",
			s, s.IsTerminal(), inProgressing[s])
		require.NotEqual(t, inTerminal[s], inProgressing[s],
			"status %q must belong to exactly one of TerminalDeploymentStatuses / ProgressingDeploymentStatuses",
			s)
	}
}
