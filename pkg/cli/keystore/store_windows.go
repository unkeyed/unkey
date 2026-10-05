//go:build windows

package keystore

import (
	"errors"

	"github.com/danieljoos/wincred"
)

func windowsCredentialTarget(service, account string) string {
	return service + "/" + account
}

func platformGet(service, account string) (string, error) {
	cred, err := wincred.GetGenericCredential(windowsCredentialTarget(service, account))
	if err != nil {
		if errors.Is(err, wincred.ErrElementNotFound) {
			return "", ErrNotFound
		}
		return "", err
	}
	if len(cred.CredentialBlob) == 0 {
		return "", ErrNotFound
	}
	return string(cred.CredentialBlob), nil
}

func platformSet(service, account, secret string) error {
	cred := wincred.NewGenericCredential(windowsCredentialTarget(service, account))
	cred.UserName = account
	cred.Comment = secretToolLabel
	cred.CredentialBlob = []byte(secret)
	cred.Persist = wincred.PersistLocalMachine
	return cred.Write()
}

func platformDelete(service, account string) error {
	cred, err := wincred.GetGenericCredential(windowsCredentialTarget(service, account))
	if err != nil {
		if errors.Is(err, wincred.ErrElementNotFound) {
			return nil
		}
		return err
	}
	return cred.Delete()
}
