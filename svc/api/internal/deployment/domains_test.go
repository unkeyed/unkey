package deployment

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/svc/api/openapi"
)

func TestMapDomains(t *testing.T) {
	custom := func(s db.CustomDomainsVerificationStatus) db.NullCustomDomainsVerificationStatus {
		return db.NullCustomDomainsVerificationStatus{Valid: true, CustomDomainsVerificationStatus: s}
	}

	rows := []db.ListDeploymentDomainsRow{
		{Domain: "kebap-app.unkey.app"},
		{Domain: "verified.kebap.com", CustomVerificationStatus: custom(db.CustomDomainsVerificationStatusVerified)},
		{Domain: "pending.kebap.com", CustomVerificationStatus: custom(db.CustomDomainsVerificationStatusPending)},
		{Domain: "verifying.kebap.com", CustomVerificationStatus: custom(db.CustomDomainsVerificationStatusVerifying)},
		{Domain: "failed.kebap.com", CustomVerificationStatus: custom(db.CustomDomainsVerificationStatusFailed)},
	}

	got := mapDomains(rows)
	require.Len(t, got, 5)

	// No custom_domains match: system, always active.
	require.Equal(t, openapi.DeploymentDomainTypeSystem, got[0].Type)
	require.Equal(t, openapi.DeploymentDomainStatusActive, got[0].Status)

	// Custom domain status maps verification -> serving status.
	require.Equal(t, openapi.DeploymentDomainTypeCustom, got[1].Type)
	require.Equal(t, openapi.DeploymentDomainStatusActive, got[1].Status)  // verified
	require.Equal(t, openapi.DeploymentDomainStatusPending, got[2].Status) // pending
	require.Equal(t, openapi.DeploymentDomainStatusPending, got[3].Status) // verifying
	require.Equal(t, openapi.DeploymentDomainStatusFailed, got[4].Status)  // failed
}

func TestMapDomainsEmpty(t *testing.T) {
	require.Empty(t, mapDomains(nil))
}
