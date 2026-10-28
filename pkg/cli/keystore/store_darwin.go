//go:build darwin

package keystore

func platformGet(service, account string) (string, error) {
	return securityGet("/usr/bin/security", service, account)
}

func platformSet(service, account, secret string) error {
	return securitySet("/usr/bin/security", service, account, secret)
}

func platformDelete(service, account string) error {
	return securityDelete("/usr/bin/security", service, account)
}
