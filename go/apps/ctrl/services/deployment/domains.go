package deployment

import (
	"fmt"
	"regexp"
	"strings"
)

// buildDomains looks at the deployment and returns a list of domains
// that should be assigned to the deployment.
//
// We want these domains per deployment
//   - `<projectslug>-git-<gitsha>-<workspaceslug>.unkey.app` (this never gets reassigned)
//   - `<projectslug>-git-<branchname>-<workspaceslug>.unkey.app` (this needs to point to the latest deployment of that branch, sluggify the branch name )
//   - `<projectslug>-<environmentslug>-<workspaceslug>.unkey.app` (this needs to point to the latest deployment of that environment and be rolled back)
func buildDomains(workspaceSlug, projectSlug, environmentSlug, gitSha, branchName, apex string) []string {

	var domains []string

	if gitSha != "" {
		short := gitSha
		if len(short) > 7 {
			short = short[:7]
		}
		domains = append(domains,
			fmt.Sprintf("%s-git-%s-%s.%s", projectSlug, short, workspaceSlug, apex),
		)
	}

	if branchName != "" {
		domains = append(
			domains,
			fmt.Sprintf("%s-git-%s-%s.%s", projectSlug, sluggify(branchName), workspaceSlug, apex),
		)
	}

	domains = append(
		domains,
		fmt.Sprintf("%s-%s-%s.%s", projectSlug, environmentSlug, workspaceSlug, apex),
	)
	return domains

}

var nonAlphanumericRegex = regexp.MustCompile(`[^a-zA-Z0-9\s]`)
var multipleSpacesRegex = regexp.MustCompile(`\s+`)

func sluggify(s string) string {
	// Trim whitespace
	s = strings.TrimSpace(s)

	// Remove all non-alphanumeric characters except spaces
	s = nonAlphanumericRegex.ReplaceAllString(s, "")

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
