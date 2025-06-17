// Package tls provides utilities for configuring TLS (Transport Layer Security) in HTTP servers.
//
// This package simplifies the creation and configuration of TLS settings for secure HTTPS
// connections. It wraps the standard library's crypto/tls package, providing convenient
// functions to create properly configured TLS settings from certificate and key data.
//
// Common use cases include:
//   - Creating a TLS configuration from certificate and key PEM data
//   - Loading TLS certificates and keys from files
//   - Configuring HTTPS servers with sensible security defaults
//   - Enabling HTTPS in command-line applications
package tls
