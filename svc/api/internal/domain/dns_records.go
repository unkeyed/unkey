// Package domain maps custom domain state onto its API representation.
package domain

import (
	"github.com/unkeyed/unkey/pkg/dns"
	"github.com/unkeyed/unkey/pkg/dns/domainconnect"
	"github.com/unkeyed/unkey/pkg/ptr"
	"github.com/unkeyed/unkey/svc/api/openapi"
)

// dnsRecordTTLSeconds is the TTL returned alongside each record.
const dnsRecordTTLSeconds = 60

// DnsRecordsInput carries the stored domain state DnsRecords renders.
type DnsRecordsInput struct {
	// Domain is the canonical name the records are built for.
	Domain string
	// TargetCname is the value of the routing record.
	TargetCname string
	// VerificationToken is the secret behind the ownership TXT record's value.
	VerificationToken string
	// RoutingVerified reports whether the routing record was read back with its
	// expected value. False on a domain no check has run against yet.
	RoutingVerified bool
	// OwnershipVerified is the same read-back result for the TXT record.
	OwnershipVerified bool
}

// DnsRecords builds the records a caller must publish, resolved for whether the
// domain is an apex or a subdomain.
//
// createDomain and getDomain both return these and the verification worker reads the published
// values back, so a divergence here leaves domains stuck unverified with no error pointing at it.
//
// The worker checks each record separately, so its two results are reported per record: a caller
// fixing setup needs to know which record is outstanding, not how many are.
func DnsRecords(in DnsRecordsInput) []openapi.DnsRecord {
	routing := openapi.DnsRecord{
		Type:     openapi.CNAME,
		Name:     in.Domain,
		Value:    in.TargetCname,
		Ttl:      dnsRecordTTLSeconds,
		Verified: in.RoutingVerified,
		Note:     ptr.P("Create as DNS-only if your provider offers the choice."),
	}
	txt := openapi.DnsRecord{
		Type:     openapi.TXT,
		Name:     dns.OwnershipTXTName(in.Domain),
		Value:    dns.OwnershipTXTValue(in.VerificationToken),
		Ttl:      dnsRecordTTLSeconds,
		Verified: in.OwnershipVerified,
		Note:     ptr.P("Proves ownership. Create it alongside the routing record."),
	}

	if domainconnect.IsApexDomain(in.Domain) {
		routing.Type = openapi.ALIAS
		routing.Note = ptr.P("Apex domains cannot hold a CNAME. Use ALIAS, ANAME, or a flattened CNAME depending on your provider.")
		txt.Note = ptr.P("Proves ownership. An apex domain cannot be verified through its routing record, so this is the only proof available.")
	}

	return []openapi.DnsRecord{routing, txt}
}
