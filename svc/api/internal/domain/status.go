package domain

import (
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/svc/api/openapi"
)

// Status maps the stored verification state onto the API enum. An unrecognised value
// means the database enum grew without this switch, so it reports pending rather than
// inventing a terminal state a caller would act on.
func Status(s db.CustomDomainsVerificationStatus) openapi.DomainStatus {
	switch s {
	case db.CustomDomainsVerificationStatusVerifying:
		return openapi.DomainStatusVerifying
	case db.CustomDomainsVerificationStatusVerified:
		return openapi.DomainStatusVerified
	case db.CustomDomainsVerificationStatusFailed:
		return openapi.DomainStatusFailed
	case db.CustomDomainsVerificationStatusPending:
		return openapi.DomainStatusPending
	default:
		return openapi.DomainStatusPending
	}
}
