package deploy

import (
	"strings"

	imagename "github.com/google/go-containerregistry/pkg/name"
)

// sameRepository reports whether two references name the same repository.
// Host names are case-insensitive and :443 is the HTTPS default, so
// "Registry.Depot.dev:443/<org>" and "registry.depot.dev/<org>" reach the same
// registry. The caller writes the image string, so a plain string comparison
// would let another spelling of the build repository skip the ownership check.
// The resolver uses the same function to decide when to send Unkey's
// credentials, so the check and the pull always agree on which images are ours.
func sameRepository(a, b imagename.Repository) bool {
	sameHost := strings.EqualFold(strings.TrimSuffix(a.RegistryStr(), ":443"), strings.TrimSuffix(b.RegistryStr(), ":443"))
	return sameHost && strings.EqualFold(a.RepositoryStr(), b.RepositoryStr())
}
