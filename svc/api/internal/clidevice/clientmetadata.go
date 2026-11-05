package clidevice

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/unkeyed/unkey/pkg/fault"
	"github.com/unkeyed/unkey/pkg/ssrf"
)

const (
	clientMetadataTimeout  = 5 * time.Second
	clientMetadataBytesMax = 5 * 1024
)

// ClientMetadata is the part of an OAuth Client ID Metadata Document that
// authorization reads. The document is hosted at the client_id URL.
type ClientMetadata struct {
	ClientID                string   `json:"client_id"`
	ClientName              string   `json:"client_name"`
	RedirectURIs            []string `json:"redirect_uris"`
	TokenEndpointAuthMethod string   `json:"token_endpoint_auth_method"`
}

// ClientMetadataFetcher loads the metadata document a client_id URL points to.
type ClientMetadataFetcher interface {
	Fetch(ctx context.Context, clientID string) (ClientMetadata, error)
}

// HTTPClientMetadata fetches client metadata documents over HTTP.
type HTTPClientMetadata struct {
	Client *http.Client
}

// NewHTTPClientMetadata fetches through an SSRF-guarded client, because the
// client_id URL is chosen by whoever starts the authorization.
func NewHTTPClientMetadata() *HTTPClientMetadata {
	return &HTTPClientMetadata{Client: ssrf.New(ssrf.WithTimeout(clientMetadataTimeout))}
}

// Fetch downloads and decodes the document. It does not check its contents.
func (f *HTTPClientMetadata) Fetch(ctx context.Context, clientID string) (ClientMetadata, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, clientID, nil)
	if err != nil {
		return ClientMetadata{}, fault.Wrap(err, fault.Internal("build client metadata request"))
	}
	req.Header.Set("Accept", "application/json")

	resp, err := f.Client.Do(req)
	if err != nil {
		return ClientMetadata{}, fault.Wrap(err, fault.Internal("fetch client metadata"))
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return ClientMetadata{}, fault.New(fmt.Sprintf("client metadata returned status %d", resp.StatusCode))
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, clientMetadataBytesMax+1))
	if err != nil {
		return ClientMetadata{}, fault.Wrap(err, fault.Internal("read client metadata"))
	}
	if len(body) > clientMetadataBytesMax {
		return ClientMetadata{}, fault.New(fmt.Sprintf("client metadata exceeds %d bytes", clientMetadataBytesMax))
	}

	var metadata ClientMetadata
	if err := json.Unmarshal(body, &metadata); err != nil {
		return ClientMetadata{}, fault.Wrap(err, fault.Internal("decode client metadata"))
	}
	return metadata, nil
}
