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
	sticky db.NullIngressRoutesSticky
}

// buildDomains generates the list of domains that should be assigned to a deployment.
//
// The function creates three types of domains:
//
// 1. Per-commit domain: `<project>-git-<sha>-<workspace>.<apex>`
//   - Never reassigned, provides stable URL for specific commit
//   - For CLI uploads, adds random suffix to prevent collisions
//
// 2. Per-branch domain: `<project>-git-<branch>-<workspace>.<apex>`
//   - Sticky to branch, always points to latest deployment of that branch
//   - Branch name is sluggified for URL safety
//
// 3. Per-environment domain: `<project>-<environment>-<workspace>.<apex>`
//   - Sticky to environment, points to latest deployment in that environment
//   - Can be rolled back to previous deployment
//
// The sticky behavior ensures branch and environment domains automatically update to point
// to new deployments, while commit domains remain immutable.
func buildDomains(workspaceSlug, projectSlug, environmentSlug, gitSha, branchName, apex string, source ctrlv1.SourceType) []newDomain {
	// Deploying via CLI often sends the same git sha, and we want to make them unique,
	// to prevent changes from overwriting each other.
	randomSuffix := ""
	if source == ctrlv1.SourceType_SOURCE_TYPE_CLI_UPLOAD {
		//nolint: gosec
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
				//nolint: exhaustruct
				sticky: db.NullIngressRoutesSticky{Valid: false},
			},
		)
	}

	if branchName != "" {
		domains = append(
			domains,
			newDomain{
				domain: fmt.Sprintf("%s-git-%s-%s.%s", projectSlug, sluggify(branchName), workspaceSlug, apex),
				sticky: db.NullIngressRoutesSticky{Valid: true, IngressRoutesSticky: db.IngressRoutesStickyBranch},
			},
		)
	}

	domains = append(
		domains,
		newDomain{
			domain: fmt.Sprintf("%s-%s-%s.%s", projectSlug, environmentSlug, workspaceSlug, apex),
			sticky: db.NullIngressRoutesSticky{Valid: true, IngressRoutesSticky: db.IngressRoutesStickyEnvironment},
		},
	)
	return domains
}

var (
	nonAlphanumericRegex = regexp.MustCompile(`[^a-zA-Z0-9\s]`)
	multipleSpacesRegex  = regexp.MustCompile(`\s+`)
)

// sluggify converts a string into a URL-safe slug.
//
// The function performs these transformations:
// - Removes all non-alphanumeric characters (except spaces, hyphens, underscores)
// - Converts hyphens and underscores to spaces
// - Collapses multiple spaces into single space
// - Replaces spaces with hyphens
// - Converts to lowercase
// - Limits to 80 characters
// - Removes trailing hyphens
//
// This is used to convert Git branch names into URL-safe domain components.
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
