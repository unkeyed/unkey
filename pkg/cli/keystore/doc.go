// Package keystore stores CLI secrets in the operating system secret store.
//
// It wraps the macOS Keychain through /usr/bin/security, the Linux Secret
// Service through secret-tool, and the Windows Credential Manager. The external
// tools keep the CLI free of cgo and platform keychain libraries.
//
// Secrets never appear in a child process's argv, where other local users could
// read them: secret-tool and `security -i` both receive the secret on stdin.
// Because `security -i` does not reliably report failures through its exit
// status, writes on macOS are confirmed by reading the item back.
//
// # Errors
//
// [ErrUnavailable] means the store cannot be used at all, such as a missing
// tool, no Secret Service session, a timeout, or a keychain that will not keep
// the secret. Callers fall back to another storage location. [ErrNotFound] means
// the store works but holds no matching item.
//
// # Usage
//
//	if err := keystore.SetRootKey(key); errors.Is(err, keystore.ErrUnavailable) {
//		// fall back to a file
//	}
//
// # Platform files
//
// store_darwin.go calls securityGet and securitySet. securitySet writes through
// `security -i` and then reads the item back because that mode does not report
// failure reliably. quoteSecurityArg quotes values for that command.
// store_linux.go calls secret-tool with the secret on stdin. store_windows.go
// uses the Credential Manager. store_unsupported.go returns [ErrUnavailable].
// classifySecurity and classifySecretTool map tool failures onto [ErrNotFound]
// or [ErrUnavailable]. runCommand applies the two minute subprocess timeout.
package keystore
