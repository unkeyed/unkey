// Package oauth serves the OAuth 2.1 authorization server endpoints: RFC 8414
// metadata, the authorization endpoint, and the token endpoint.
//
// They are protocol endpoints, not v2 API operations. They are absent from the
// OpenAPI document and answer with RFC 6749 shapes instead of the API's
// response envelope. [clidevice.Service] holds the flow. These handlers parse
// parameters and map [clidevice.OAuthError] to HTTP. Any other error falls
// through to the API's error middleware.
package oauth
