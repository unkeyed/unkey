package deploy

import (
	"net"
	"strings"

	imagename "github.com/google/go-containerregistry/pkg/name"
)

// sameRepository reports whether two references name the same repository.
// The caller writes the image string, so a plain string comparison would let
// another spelling of the build repository skip the ownership check. The
// resolver uses the same function to decide when to send Unkey's credentials,
// so the check and the pull always agree on which images are ours.
func sameRepository(a, b imagename.Repository) bool {
	return sameRegistry(a.Registry, b.Registry) && strings.EqualFold(a.RepositoryStr(), b.RepositoryStr())
}

// sameRegistry reports whether two references name the same registry host,
// ignoring case and port. A port can be spelled many ways (":443", ":0443"),
// so comparing it would let one spelling of the build host skip the check.
func sameRegistry(a, b imagename.Registry) bool {
	return strings.EqualFold(registryHost(a), registryHost(b))
}

func registryHost(r imagename.Registry) string {
	host, _, err := net.SplitHostPort(r.RegistryStr())
	if err != nil {
		return r.RegistryStr()
	}
	return host
}
