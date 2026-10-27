package deploy

import (
	"strings"

	"github.com/google/go-containerregistry/pkg/name"
)

// sameRepository reports whether two references name the same repository. The
// build registry is written by hand in config and named by callers, so the two
// spellings differ on host case and an explicit :443. Deploy and the resolver
// must agree on that, or a reference passes the ownership check and is then
// fetched anonymously.
func sameRepository(a, b name.Repository) bool {
	sameHost := strings.EqualFold(strings.TrimSuffix(a.RegistryStr(), ":443"), strings.TrimSuffix(b.RegistryStr(), ":443"))
	return sameHost && strings.EqualFold(a.RepositoryStr(), b.RepositoryStr())
}
