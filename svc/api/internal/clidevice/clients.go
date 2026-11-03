package clidevice

import (
	"context"
	"net"
	"net/url"
	"slices"
	"strings"

	"github.com/unkeyed/unkey/pkg/logger"
)

const metadataClientIDLengthMax = 512

// OAuthClient is what the authorization server trusts about a client, however
// it was identified. Name is self-reported, so the consent page shows a host
// next to it.
type OAuthClient struct {
	ID           string
	Name         string
	RedirectURIs []string
}

// allowsRedirect reports whether redirectURI is registered for the client.
// Loopback redirect URIs match on any port, because native clients listen
// on whatever port is free (RFC 8252 section 7.3).
func (c OAuthClient) allowsRedirect(redirectURI string) bool {
	if slices.Contains(c.RedirectURIs, redirectURI) {
		return true
	}
	requested, err := url.Parse(redirectURI)
	if err != nil || !loopbackRedirect(requested) {
		return false
	}
	for _, raw := range c.RedirectURIs {
		registered, err := url.Parse(raw)
		if err == nil && loopbackRedirect(registered) &&
			registered.Hostname() == requested.Hostname() &&
			registered.EscapedPath() == requested.EscapedPath() &&
			registered.RawQuery == requested.RawQuery {
			return true
		}
	}
	return false
}

// Directory resolves a client_id. The shape of the id picks the source, so
// neither source ever sees the other's ids: an https URL is a Client ID
// Metadata Document, and a registeredClientPrefix id is a stateless dynamic
// registration.
type Directory struct {
	Documents ClientMetadataFetcher
	Registrar *Registrar
}

// Lookup returns the client behind clientID. A client that cannot be trusted
// is an [OAuthError] with [OAuthInvalidRequest]; any other error is an
// infrastructure failure.
func (d Directory) Lookup(ctx context.Context, clientID string) (OAuthClient, error) {
	if strings.HasPrefix(clientID, registeredClientPrefix) {
		return d.Registrar.Lookup(clientID)
	}
	return d.lookupDocument(ctx, clientID)
}

func (d Directory) lookupDocument(ctx context.Context, clientID string) (OAuthClient, error) {
	clientURL, err := parseMetadataClientID(clientID)
	if err != nil {
		return OAuthClient{}, err
	}
	metadata, err := d.Documents.Fetch(ctx, clientID)
	if err != nil {
		logger.Info("oauth client metadata fetch failed", "client_id", clientID, "error", err.Error())
		return OAuthClient{}, oauthError(OAuthInvalidRequest, "The client metadata document could not be loaded.")
	}
	if metadata.ClientID != clientID {
		return OAuthClient{}, oauthError(OAuthInvalidRequest, "The client metadata document does not match client_id.")
	}
	if metadata.TokenEndpointAuthMethod != "" && metadata.TokenEndpointAuthMethod != tokenEndpointAuthNone {
		return OAuthClient{}, oauthError(OAuthInvalidRequest, "Only public clients with token_endpoint_auth_method none are supported.")
	}
	name := strings.TrimSpace(metadata.ClientName)
	if name == "" {
		name = clientURL.Host
	}
	return OAuthClient{
		ID:           clientID,
		Name:         name,
		RedirectURIs: metadata.RedirectURIs,
	}, nil
}

// parseMetadataClientID enforces the client_id URL rules of the OAuth Client
// ID Metadata Document draft.
func parseMetadataClientID(raw string) (*url.URL, error) {
	invalid := oauthError(OAuthInvalidRequest, "client_id must be an https URL with a path and no fragment, or a registered client id.")
	if raw == "" || len(raw) > metadataClientIDLengthMax || strings.Contains(raw, "#") {
		return nil, invalid
	}
	parsed, err := url.Parse(raw)
	if err != nil || parsed.Scheme != "https" || parsed.Host == "" || parsed.User != nil {
		return nil, invalid
	}
	if parsed.Path == "" || parsed.Path == "/" {
		return nil, invalid
	}
	for segment := range strings.SplitSeq(parsed.Path, "/") {
		if segment == "." || segment == ".." {
			return nil, invalid
		}
	}
	return parsed, nil
}

// clientHost is the host the consent page and session list show next to a
// client's self-reported name. A metadata document client is vouched for by
// the host serving its document. A registered client is vouched for by
// nobody, so the host is where the authorization code is sent, and empty
// when that is unknown.
func clientHost(clientID string, redirectURI string) string {
	if strings.HasPrefix(clientID, registeredClientPrefix) {
		return hostOf(redirectURI)
	}
	return hostOf(clientID)
}

func hostOf(raw string) string {
	parsed, err := url.Parse(raw)
	if err != nil {
		return ""
	}
	return parsed.Host
}

// checkRedirectURI accepts https, or http on a loopback host (RFC 8252
// section 7.3), and nothing else. The dashboard navigates the browser to the
// decision redirect, so any other scheme could run script in the dashboard
// origin or hand the code to whichever local app claims the scheme.
func checkRedirectURI(raw string) error {
	invalid := oauthError(OAuthInvalidRequest, "redirect_uri must be an https URL, or http on a loopback address, without a fragment.")
	if strings.Contains(raw, "#") {
		return invalid
	}
	parsed, err := url.Parse(raw)
	if err != nil || parsed.Host == "" || parsed.User != nil {
		return invalid
	}
	if parsed.Scheme == "https" || loopbackRedirect(parsed) {
		return nil
	}
	return invalid
}

func loopbackRedirect(u *url.URL) bool {
	if u.Scheme != "http" {
		return false
	}
	host := u.Hostname()
	if host == "localhost" {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}
