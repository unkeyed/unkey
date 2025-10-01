package deploy

import (
	"fmt"
	"math/rand/v2"
	"regexp"
	"strings"

	ctrlv1 "github.com/unkeyed/unkey/go/gen/proto/ctrl/v1"
	"github.com/unkeyed/unkey/go/pkg/db"
)

type newDomain struct {
	domain string
	sticky db.NullDomainsSticky
}

// buildDomains looks at the deployment and returns a list of domains
// that should be assigned to the deployment.
//
// We want these domains per deployment
//   - `<projectslug>-git-<gitsha>-<workspaceslug>.unkey.app` (this never gets reassigned)
//   - `<projectslug>-git-<branchname>-<workspaceslug>.unkey.app` (this needs to point to the latest deployment of that branch, sluggify the branch name )
//   - `<projectslug>-<environmentslug>-<workspaceslug>.unkey.app` (this needs to point to the latest deployment of that environment and be rolled back)
func buildDomains(workspaceSlug, projectSlug, environmentSlug, gitSha, branchName, apex string, source ctrlv1.SourceType) []newDomain {

	// Deploying via CLI often sends the same git sha, and we want to make them unique,
	// to prevent changes from overwriting each other.
	randomSuffix := ""
	if source == ctrlv1.SourceType_SOURCE_TYPE_CLI_UPLOAD {
		randomSuffix = fmt.Sprintf("-%d", 1000+rand.IntN(9000))
	}

	var domains []newDomain

	if gitSha != "" {
		short := gitSha
		if len(short) > 7 {
			short = short[:7]
		}
		short += randomSuffix
		domains = append(domains,
			newDomain{
				domain: fmt.Sprintf("%s-git-%s-%s.%s", projectSlug, short, workspaceSlug, apex),
				sticky: db.NullDomainsSticky{Valid: false},
			},
		)
	}

	if branchName != "" {
		domains = append(
			domains,
			newDomain{
				domain: fmt.Sprintf("%s-git-%s-%s.%s", projectSlug, sluggify(branchName), workspaceSlug, apex),
				sticky: db.NullDomainsSticky{Valid: true, DomainsSticky: db.DomainsStickyBranch},
			},
		)
	}

	domains = append(
		domains,
		newDomain{
			domain: fmt.Sprintf("%s-%s-%s.%s", projectSlug, environmentSlug, workspaceSlug, apex),
			sticky: db.NullDomainsSticky{Valid: true, DomainsSticky: db.DomainsStickyEnvironment},
		},
	)
	return domains

}

var nonAlphanumericRegex = regexp.MustCompile(`[^a-zA-Z0-9\s]`)
var multipleSpacesRegex = regexp.MustCompile(`\s+`)

func sluggify(s string) string {
	// Trim whitespace
	s = strings.TrimSpace(s)

	// Remove all non-alphanumeric characters except spaces
	s = nonAlphanumericRegex.ReplaceAllString(s, " ")
	s = strings.ReplaceAll(s, "-", " ")
	s = strings.ReplaceAll(s, "_", " ")

	// Replace multiple spaces with single space
	s = multipleSpacesRegex.ReplaceAllString(s, " ")

	// Replace spaces with hyphens
	s = strings.ReplaceAll(s, " ", "-")

	// Convert to lowercase
	s = strings.ToLower(s)

	// Limit to 80 characters
	if len(s) > 80 {
		s = s[:80]
	}

	// Remove trailing hyphen if present
	s = strings.TrimSuffix(s, "-")

	return s
}
