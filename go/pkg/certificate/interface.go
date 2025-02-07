package certificate

import (
	"crypto/tls"
)

type Source interface {
	GetCertificate(*tls.ClientHelloInfo) (*tls.Certificate, error)
}
