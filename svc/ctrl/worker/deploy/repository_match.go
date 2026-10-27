package deploy

import (
	"strings"

	"github.com/google/go-containerregistry/pkg/name"
)

// sameRepository compares two repositories, ignoring host case and an explicit
// :443. The ownership check and the resolver both use it, so an image they see
// as outside the build repository is also fetched without its credentials.
// For example, "Registry.Unkey.cloud:443/builds" matches "registry.unkey.cloud/builds".
func sameRepository(a, b name.Repository) bool {
	sameHost := strings.EqualFold(strings.TrimSuffix(a.RegistryStr(), ":443"), strings.TrimSuffix(b.RegistryStr(), ":443"))
	return sameHost && strings.EqualFold(a.RepositoryStr(), b.RepositoryStr())
}
