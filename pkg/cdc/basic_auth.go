package cdc

import "context"

// basicAuth holds the base64-encoded username and password. It requires TLS.
type basicAuth string

// GetRequestMetadata sends the credentials in PlanetScale's HTTP Basic format.
func (a basicAuth) GetRequestMetadata(context.Context, ...string) (map[string]string, error) {
	return map[string]string{"authorization": "Basic " + string(a)}, nil
}

// RequireTransportSecurity prevents sending credentials without TLS.
func (basicAuth) RequireTransportSecurity() bool { return true }
