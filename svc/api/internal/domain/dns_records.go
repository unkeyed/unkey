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

// DnsRecords builds the records a caller must publish for domainName, resolved for whether it is an
// apex or a subdomain. routingVerified and ownershipVerified report which records have been read
// back with their expected values; a domain no check has run against yet passes false for both.
//
// createDomain and getDomain both return these and the verification worker reads the published
// values back, so a divergence here leaves domains stuck unverified with no error pointing at it.
func DnsRecords(domainName, targetCname, verificationToken string, routingVerified, ownershipVerified bool) []openapi.DnsRecord {
	routing := openapi.DnsRecord{
		Type:     openapi.CNAME,
		Name:     domainName,
		Value:    targetCname,
		Ttl:      dnsRecordTTLSeconds,
		Verified: routingVerified,
		Note:     ptr.P("Create as DNS-only if your provider offers the choice."),
	}
	txt := openapi.DnsRecord{
		Type:     openapi.TXT,
		Name:     dns.OwnershipTXTName(domainName),
		Value:    dns.OwnershipTXTValue(verificationToken),
		Ttl:      dnsRecordTTLSeconds,
		Verified: ownershipVerified,
		Note:     ptr.P("Proves ownership. Create it alongside the routing record."),
	}

	if domainconnect.IsApexDomain(domainName) {
		routing.Type = openapi.ALIAS
		routing.Note = ptr.P("Apex domains cannot hold a CNAME. Use ALIAS, ANAME, or a flattened CNAME depending on your provider.")
		txt.Note = ptr.P("Proves ownership. An apex domain cannot be verified through its routing record, so this is the only proof available.")
	}

	return []openapi.DnsRecord{routing, txt}
}
