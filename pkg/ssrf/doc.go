// Package ssrf builds HTTP clients for requests to customer-supplied
// endpoints (logdrains, outbound webhooks, and similar integrations).
//
// URLs that customers control are untrusted input: a hostname can resolve to
// our own infrastructure (cloud metadata services, internal services,
// loopback). Clients from [New] therefore guard against SSRF at dial time:
// every resolved IP is checked with [IsForbiddenIP], redirects are not
// followed unless [FollowRedirects] is set (and then only to https targets),
// and response header size is capped. The guard runs after DNS resolution,
// so DNS-rebinding to a private address is also caught.
//
// Use [ValidateEndpoint] at configuration time to reject non-https URLs
// early, and [New] for the actual requests:
//
//	if err := ssrf.ValidateEndpoint(endpoint); err != nil {
//		return err
//	}
//	client := ssrf.New(ssrf.WithTimeout(30 * time.Second))
//	resp, err := client.Do(req)
//
// The guard is not the right tool for calls that legitimately target private
// addresses, such as in-cluster service traffic; it protects egress to
// endpoints outside our infrastructure.
package ssrf
