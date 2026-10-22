package keystore

import (
	"errors"
	"fmt"

	"github.com/unkeyed/unkey/pkg/assert"
)

const (
	serviceName    = "unkey"
	rootKeyAccount = "root_key"
	sessionAccount = "cli_sessions"
)

var (
	// ErrNotFound is returned when the secret store has no matching item.
	ErrNotFound = errors.New("keystore: secret not found")
	// ErrUnavailable is returned when the secret store cannot be opened.
	ErrUnavailable = errors.New("keystore: secret store unavailable")
)

func Get(service, account string) (string, error) {
	if err := validateNames(service, account); err != nil {
		return "", err
	}
	secret, err := platformGet(service, account)
	if err != nil {
		return "", err
	}
	if secret == "" {
		return "", ErrNotFound
	}
	return secret, nil
}

// Set writes secret for service and account, replacing any previous value.
func Set(service, account, secret string) error {
	if err := validateNames(service, account); err != nil {
		return err
	}
	if secret == "" {
		return fmt.Errorf("keystore: secret cannot be empty")
	}
	return platformSet(service, account, secret)
}

// Delete removes the secret for service and account. A missing item is not an error.
func Delete(service, account string) error {
	if err := validateNames(service, account); err != nil {
		return err
	}
	return platformDelete(service, account)
}

func validateNames(service, account string) error {
	return assert.All(
		assert.NotEmpty(service, "keystore: service is required"),
		assert.NotEmpty(account, "keystore: account is required"),
	)
}

// GetRootKey reads the stored root key from the operating system secret store.
func GetRootKey() (string, error) {
	return Get(serviceName, rootKeyAccount)
}

// SetRootKey writes the root key, replacing any previous value.
func SetRootKey(secret string) error {
	return Set(serviceName, rootKeyAccount, secret)
}

// GetSessions reads the CLI session blob.
func GetSessions() (string, error) {
	return Get(serviceName, sessionAccount)
}

// windowsCredentialBlobLimit is the Credential Manager maximum for one blob.
// A measured CLI session blob is 761 bytes for an admin, 2,850 for a viewer,
// and 6,111 for a developer, so a viewer or developer session does not fit.
const windowsCredentialBlobLimit = 2560

// ExceedsWindowsCredentialBlob reports whether secret is larger than Windows
// Credential Manager can store.
func ExceedsWindowsCredentialBlob(secret string) bool {
	return len(secret) > windowsCredentialBlobLimit
}

// SetSessions writes the CLI session blob.
func SetSessions(secret string) error {
	return Set(serviceName, sessionAccount, secret)
}

// DeleteSessions removes the CLI session blob. A missing item is not an error.
func DeleteSessions() error {
	return Delete(serviceName, sessionAccount)
}
