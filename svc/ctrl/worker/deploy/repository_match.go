package deploy

import (
	"strings"

	"github.com/google/go-containerregistry/pkg/name"
)

// sameRepository compares the registry and path, ignoring host case and an
// explicit :443, which config and callers spell differently. Deploy and the
// resolver must agree, or a reference passes the ownership check and is then
// fetched anonymously.
func sameRepository(a, b name.Repository) bool {
	sameHost := strings.EqualFold(strings.TrimSuffix(a.RegistryStr(), ":443"), strings.TrimSuffix(b.RegistryStr(), ":443"))
	return sameHost && strings.EqualFold(a.RepositoryStr(), b.RepositoryStr())
}
