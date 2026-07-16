package deployment

import (
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/svc/api/openapi"
)

// mapDomains turns the joined route rows into wire domains. A row with no
// matching custom_domains entry is a system hostname (always active); a match is
// a custom domain whose status reflects its verification.
func mapDomains(rows []db.ListDeploymentDomainsRow) []openapi.DeploymentDomain {
	domains := make([]openapi.DeploymentDomain, 0, len(rows))
	for _, r := range rows {
		domain := openapi.DeploymentDomain{
			Domain: r.Domain,
			Type:   openapi.DeploymentDomainTypeSystem,
			Status: openapi.DeploymentDomainStatusActive,
		}
		if r.CustomVerificationStatus.Valid {
			domain.Type = openapi.DeploymentDomainTypeCustom
			domain.Status = customDomainStatus(r.CustomVerificationStatus.CustomDomainsVerificationStatus)
		}
		domains = append(domains, domain)
	}
	return domains
}

func customDomainStatus(s db.CustomDomainsVerificationStatus) openapi.DeploymentDomainStatus {
	switch s {
	case db.CustomDomainsVerificationStatusVerified:
		return openapi.DeploymentDomainStatusActive
	case db.CustomDomainsVerificationStatusFailed:
		return openapi.DeploymentDomainStatusFailed
	case db.CustomDomainsVerificationStatusPending, db.CustomDomainsVerificationStatusVerifying:
		return openapi.DeploymentDomainStatusPending
	default:
		return openapi.DeploymentDomainStatusPending
	}
}
