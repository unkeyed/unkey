package keystore

import (
	"errors"
	"fmt"
)

const (
	Service = "unkey"
	Account = "root_key"
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
	if service == "" || account == "" {
		return fmt.Errorf("keystore: service and account are required")
	}
	return nil
}
