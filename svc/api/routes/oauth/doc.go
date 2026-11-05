// Package oauth serves the OAuth 2.1 authorization server MCP clients sign in
// through: RFC 8414 metadata, the authorization and token endpoints, and RFC
// 7591 dynamic client registration.
//
// They are protocol endpoints, not v2 API operations. They are absent from the
// OpenAPI document, so they stay out of the SDKs and the MCP tool list, and
// they answer with RFC 6749 shapes instead of the API's response envelope.
// [clidevice.Service] holds the flow. These handlers parse parameters and map
// [clidevice.OAuthError] to HTTP. Any other error falls through to the API's
// error middleware.
package oauth
