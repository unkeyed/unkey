// Package mcp is a staging spike that checks WorkOS AuthKit access tokens for
// two MCP resources. It is not a production MCP server, and it does not mint
// the internal API JWTs that a later phase will use.
//
// Verification accepts an aud claim encoded as either a JSON string or a JSON
// array. [github.com/unkeyed/unkey/pkg/jwt] only decodes aud arrays, while
// WorkOS Connect tokens use a string.
package mcp
