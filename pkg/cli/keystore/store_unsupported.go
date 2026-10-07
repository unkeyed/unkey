//go:build !darwin && !linux && !windows

package keystore

func platformGet(service, account string) (string, error) {
	return "", ErrUnavailable
}

func platformSet(service, account, secret string) error {
	return ErrUnavailable
}

func platformDelete(service, account string) error {
	return ErrUnavailable
}
