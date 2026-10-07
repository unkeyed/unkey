//go:build linux

package keystore

const secretToolBin = "secret-tool"

func platformGet(service, account string) (string, error) {
	return secretToolGet(secretToolBin, service, account)
}

func platformSet(service, account, secret string) error {
	return secretToolSet(secretToolBin, service, account, secret)
}

func platformDelete(service, account string) error {
	return secretToolDelete(secretToolBin, service, account)
}
