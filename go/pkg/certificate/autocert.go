package certificate

import (
	"crypto/tls"
	"log"
	"sync/atomic"

	"golang.org/x/crypto/acme/autocert"
)

type devCertificateSource struct {
	manager *autocert.Manager
	counter atomic.Uint64
}

func (cs *devCertificateSource) GetCertificate(info *tls.ClientHelloInfo) (*tls.Certificate, error) {
	log.Println("getCertificate", cs.counter.Add(1))

	return cs.manager.GetCertificate(info)
}

func NewDevCertificateSource() (*devCertificateSource, error) {

	m := &autocert.Manager{
		Prompt: autocert.AcceptTOS,
		//		HostPolicy: autocert.HostWhitelist("andreas.localhost.com"),
	}

	return &devCertificateSource{
		manager: m,
		counter: atomic.Uint64{},
	}, nil

}
