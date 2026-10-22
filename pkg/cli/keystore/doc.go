// Package keystore stores CLI secrets in the operating system secret store.
//
// It wraps the macOS Keychain through /usr/bin/security, the Linux Secret
// Service through secret-tool, and the Windows Credential Manager. The external
// tools keep the CLI free of cgo and platform keychain libraries.
//
// Secrets never appear in a child process's argv, where other local users could
// read them: secret-tool and `security -i` both receive the secret on stdin.
// `security -i` keeps only 4095 bytes of a command line and drops the rest
// without a reliable error, so a secret that does not fit is stored as several
// items. Those pieces are written under a fresh nonce, and the primary item
// is replaced only after they are stored. A crash before that replacement
// leaves the previous secret in place. A file lock keeps two writers from
// deleting each other's pieces. Because interactive mode also does not
// reliably report failures through its exit status, writes on macOS are
// confirmed by reading the items back and a mismatch restores the previous
// manifest.
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
// `security -i`, splitting a secret that would exceed the 4095-byte command
// line, and then reads the item back because that mode does not report failure
// reliably. quoteSecurityArg quotes values for that command.
// store_linux.go calls secret-tool with the secret on stdin. store_windows.go
// uses the Credential Manager. store_unsupported.go returns [ErrUnavailable].
// classifySecurity and classifySecretTool map tool failures onto [ErrNotFound]
// or [ErrUnavailable]. runCommand applies the two minute subprocess timeout.
package keystore
