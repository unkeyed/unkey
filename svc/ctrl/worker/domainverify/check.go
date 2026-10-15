package domainverify

import (
	"context"
	"strings"

	"github.com/unkeyed/unkey/pkg/dns"
	"github.com/unkeyed/unkey/pkg/dns/domainconnect"
	"github.com/unkeyed/unkey/pkg/fault"
	"github.com/unkeyed/unkey/pkg/logger"
	"github.com/unkeyed/unkey/svc/ctrl/internal/db"
)

// Domain is the part of a hostname row that verification reads.
type Domain struct {
	Hostname          string
	WorkspaceID       string
	TargetCname       string
	VerificationToken string
}

// Outcome is one DNS check of a [Domain]. The flags are persisted on the row so
// the tenant can see which record is still missing.
type Outcome struct {
	CnameVerified bool
	// TxtVerified is true when TXT proof was found, and also when it was not
	// required, matching what the row has always stored as ownership_verified.
	TxtVerified    bool
	RequiresTxt    bool
	Contested      bool
	IsApex         bool
	ApexHasRecords bool
	Verified       bool
}

// ClaimFinder looks up a verified claim on a hostname outside a workspace.
type ClaimFinder interface {
	FindVerifiedDomainClaimExcludingWorkspace(ctx context.Context, arg db.FindVerifiedDomainClaimExcludingWorkspaceParams) (db.FindVerifiedDomainClaimExcludingWorkspaceRow, error)
}

// Check resolves d once and decides whether it is verified.
//
// A CNAME to the row's unique target verifies a subdomain. Apex domains cannot
// expose a CNAME (ALIAS/ANAME, flattening and proxies hide it), so they prove
// ownership with a TXT record at _unkey.<domain> and must also resolve to an
// address, because TXT proves ownership but not that routing is configured.
// TXT is the only proof accepted when another workspace holds the hostname
// verified in either table.
//
// DNS errors are logged and read as a missing record, so a resolver blip costs
// one retry rather than failing the workflow. Only the contention lookup
// returns an error.
func Check(ctx context.Context, resolver Resolver, claims ClaimFinder, d Domain) (Outcome, error) {
	cnameVerified, err := checkCNAME(ctx, resolver, d.Hostname, d.TargetCname)
	if err != nil {
		logger.Warn("CNAME check error", "domain", d.Hostname, "error", err)
	}

	// Contention only matters once the CNAME passes: without it TXT is already
	// required.
	contested := false
	if cnameVerified {
		_, findErr := claims.FindVerifiedDomainClaimExcludingWorkspace(ctx, db.FindVerifiedDomainClaimExcludingWorkspaceParams{
			Domain:      d.Hostname,
			WorkspaceID: d.WorkspaceID,
		})
		if findErr != nil && !db.IsNotFound(findErr) {
			return Outcome{}, fault.Wrap(findErr, fault.Internal("failed to check domain contention"))
		}
		contested = findErr == nil
	}

	requiresTxt := !cnameVerified || contested
	txtVerified := true
	if requiresTxt {
		txtVerified, err = checkTXTRecord(ctx, resolver, d.Hostname, d.VerificationToken)
		if err != nil {
			logger.Warn("TXT check error", "domain", d.Hostname, "error", err)
		}
	}

	isApex := domainconnect.IsApexDomain(d.Hostname)
	apexHasRecords := true
	if isApex {
		apexHasRecords, err = hasAddressRecords(ctx, resolver, d.Hostname)
		if err != nil {
			logger.Warn("apex DNS lookup error", "domain", d.Hostname, "error", err)
		}
	}

	outcome := Outcome{
		CnameVerified:  cnameVerified,
		TxtVerified:    txtVerified,
		RequiresTxt:    requiresTxt,
		Contested:      contested,
		IsApex:         isApex,
		ApexHasRecords: apexHasRecords,
		Verified:       false,
	}
	outcome.Verified = decide(outcome)
	return outcome, nil
}

// decide applies the verification rule to the DNS evidence.
func decide(o Outcome) bool {
	verified := o.CnameVerified || o.TxtVerified
	if o.Contested {
		verified = o.TxtVerified
	}
	if o.IsApex {
		verified = verified && o.ApexHasRecords
	}
	return verified
}

// checkTXTRecord reports whether _unkey.<domain> carries the ownership value for
// expectedToken.
func checkTXTRecord(ctx context.Context, resolver Resolver, domain, expectedToken string) (bool, error) {
	ctx, cancel := context.WithTimeout(ctx, dns.DefaultTimeout)
	defer cancel()

	txtRecords, err := resolver.LookupTXT(ctx, dns.OwnershipTXTName(domain))
	if err != nil {
		if dns.IsNotFoundError(err) {
			return false, nil
		}
		return false, err
	}

	expected := dns.OwnershipTXTValue(expectedToken)
	for _, txt := range txtRecords {
		if strings.EqualFold(txt, expected) {
			return true, nil
		}
	}
	return false, nil
}

// checkCNAME reports whether domain has a CNAME to expectedCname.
func checkCNAME(ctx context.Context, resolver Resolver, domain, expectedCname string) (bool, error) {
	ctx, cancel := context.WithTimeout(ctx, dns.DefaultTimeout)
	defer cancel()

	expectedCname = strings.ToLower(strings.TrimSuffix(expectedCname, "."))

	cname, err := resolver.LookupCNAME(ctx, domain)
	if err != nil {
		if dns.IsNotFoundError(err) {
			return false, nil
		}
		return false, err
	}

	// The resolver answers with the domain itself when it has no CNAME record.
	if cname == strings.ToLower(strings.TrimSuffix(domain, ".")) {
		return false, nil
	}

	return cname == expectedCname, nil
}

// hasAddressRecords reports whether domain resolves to at least one A or AAAA
// address. Addresses are not compared to a target because proxies and shared
// CDN IPs make that unreliable.
func hasAddressRecords(ctx context.Context, resolver Resolver, domain string) (bool, error) {
	ctx, cancel := context.WithTimeout(ctx, dns.DefaultTimeout)
	defer cancel()

	addrs, err := resolver.LookupHost(ctx, domain)
	if err != nil {
		if dns.IsNotFoundError(err) {
			return false, nil
		}
		return false, err
	}
	return len(addrs) > 0, nil
}
