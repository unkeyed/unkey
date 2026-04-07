// Package domainconnect implements the Domain Connect protocol (v2) for one-click
// DNS configuration of custom domains.
//
// Domain Connect is an open standard that lets DNS providers offer a consent-based
// UI for automatic record provisioning. Instead of asking users to manually copy
// CNAME and TXT values into their DNS provider, we redirect them to a provider-hosted
// page where the records are applied with one click.
//
// # Why we own this implementation
//
// The signing step is security-sensitive and must exactly match the spec. No existing
// Go library produces spec-compliant signatures, so we implement discovery, URL
// building, and signing ourselves with test coverage against the spec rules.
//
// # Protocol flow
//
//  1. Discovery: Look up _domainconnect.{zone} TXT record to find the provider host
//  2. Settings:  GET https://{host}/v2/{zone}/settings to learn the provider's sync URL
//  3. Build URL: Construct /v2/domainTemplates/providers/{id}/services/{svc}/apply with query params
//  4. Sign URL:  RSA-SHA256 over the full query string (excluding "sig" and "key")
//  5. Redirect:  User visits the signed URL, provider shows a consent page, records are applied
//
// Steps 1-4 happen server-side in [Discover]. Step 5 is a browser redirect initiated
// by the dashboard frontend.
//
// # Signing rules (spec section 5.2.1)
//
// The data to sign is built by taking all query parameters except "sig" and "key",
// sorting them alphabetically by key, URL-encoding values, and joining with "&".
// This is exactly what [net/url.Values.Encode] produces.
//
// The hash is SHA-256, signed with RSA PKCS1v15. The signature is standard base64
// encoded (not URL-safe base64). The "sig" parameter must be the last query parameter
// in the URL (Cloudflare requirement).
//
// The provider verifies the signature by fetching the public key from a TXT record at
// {keyID}.{syncPubKeyDomain} (configured in the service template).
//
// # Apex domains and CNAME limitations
//
// Domain Connect templates provision CNAME records, but CNAME cannot coexist with
// other record types at the zone apex (RFC 1034 section 3.6.2). Some providers work
// around this:
//
//   - Cloudflare: "CNAME flattening" resolves the CNAME chain and returns A/AAAA records
//   - Other providers: Most reject CNAME at @ outright
//
// The spec defines an APEXCNAME extension for this, but neither Cloudflare nor Vercel
// implements it. We expose [IsApexDomain] so callers can skip Domain Connect for apex
// domains on non-Cloudflare providers and fall back to manual DNS instructions.
//
// # Usage
//
// Discover whether a domain's DNS provider supports Domain Connect and get a signed URL:
//
//	result, err := domainconnect.Discover(ctx, "api.example.com", privateKeyPEM,
//	    map[string]string{"target": "abc123.cname.unkey.cloud"},
//	    "https://app.unkey.com/ws/project/settings",
//	)
//	if err != nil {
//	    // Discovery or signing failed
//	}
//	if result == nil {
//	    // Provider doesn't support Domain Connect; show manual DNS instructions
//	}
//	// result.URL is the signed redirect URL
//	// result.ProviderName is "Cloudflare", "Vercel", etc.
//
// Check whether to offer Domain Connect for apex domains:
//
//	if domainconnect.IsApexDomain(domain) && provider != "Cloudflare" {
//	    // Skip Domain Connect, show manual ALIAS/ANAME instructions
//	}
//
// Validate a private key at startup:
//
//	if err := domainconnect.ValidatePrivateKey(pemBytes); err != nil {
//	    log.Fatal("invalid domain connect signing key", "error", err)
//	}
//
// # Spec references
//
//   - Protocol spec: https://github.com/Domain-Connect/spec/blob/master/Domain%20Connect%20Spec%20Draft.adoc
//   - Template registry: https://github.com/Domain-Connect/Templates
//   - Signing spec: spec section 5.2.1 "Digitally Signing Requests"
package domainconnect
