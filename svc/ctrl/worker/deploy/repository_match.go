package deploy

import (
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

// sameRegistry reports whether two references name the same registry host.
// Host names are case-insensitive and :443 is the HTTPS default, so
// "Registry.Depot.dev:443" and "registry.depot.dev" reach the same registry.
func sameRegistry(a, b imagename.Registry) bool {
	return strings.EqualFold(strings.TrimSuffix(a.RegistryStr(), ":443"), strings.TrimSuffix(b.RegistryStr(), ":443"))
}
