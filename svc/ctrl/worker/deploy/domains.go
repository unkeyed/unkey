package deploy

import (
	"fmt"
	"math/rand/v2"
	"regexp"
	"strings"

	ctrlv1 "github.com/unkeyed/unkey/gen/proto/ctrl/v1"
	"github.com/unkeyed/unkey/pkg/db"
)

// newDomain represents a domain to be created for a deployment, including its
// stickiness behavior for routing updates.
type newDomain struct {
	// domain is the fully qualified domain name (e.g., "myapp-production-acme.unkey.app").
	domain string

	// sticky determines how this domain behaves across deployments. Non-sticky domains
	// remain pinned to their original deployment, while sticky domains automatically
	// update to point to new deployments matching their criteria.
	sticky db.FrontlineRoutesSticky
}

// buildDomains generates the list of domains that should be assigned to a deployment.
//
// The function creates three or four types of domains:
//
// 1. Per-commit domain: `<prefix>-git-<sha>-<workspace>.<apex>`
//   - Never reassigned, provides stable URL for specific commit
//   - For CLI uploads, adds random suffix to prevent collisions
//
// 2. Per-branch domain: `<prefix>-git-<branch>-<workspace>.<apex>`
//   - Sticky to branch, always points to latest deployment of that branch
//   - Branch name is sluggified for URL safety
//
// 3. Per-environment domain: `<prefix>-<environment>-<workspace>.<apex>`
//   - Sticky to environment, points to latest deployment in that environment
//   - Can be rolled back to previous deployment
//
// 4. Per-live domain (production only): `<prefix>-<workspace>.<apex>`
//   - Sticky to live, points to the active production deployment
//
// 5. Per-deployment domain: `<prefix>-dep-<deploymentID>-<workspace>.<apex>`
//   - Never reassigned, provides a stable deployment-specific URL
//
// Where <prefix> is `<project>-<app>` for custom app slugs, or just `<project>`
// when the app slug is "default" (see TODO above).
// TODO: Once users can configure custom app slugs, include the app slug in all
// generated domains. Currently the only app slug is "default" which adds no
// useful information to the domain and just makes URLs longer. Remove this
// exclusion and always include appSlug once the dashboard supports renaming apps.
func buildDomains(workspaceSlug, projectSlug, appSlug, environmentSlug, gitSha, branchName, apex string, source ctrlv1.SourceType, deploymentID string) []newDomain {
	// Build the project-app prefix for domain names.
	// Skip "default" app slug since it's not configurable yet and would just
	// add noise to URLs (e.g. "myproject-default-..." vs "myproject-...").
	prefix := projectSlug
	if appSlug != "default" {
		prefix = projectSlug + "-" + appSlug
	}

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
				domain: fmt.Sprintf("%s-git-%s-%s.%s", prefix, short, workspaceSlug, apex),
				//nolint: exhaustruct
				sticky: db.FrontlineRoutesStickyNone,
			},
		)
	}

	if branchName != "" {
		domains = append(
			domains,
			newDomain{
				domain: fmt.Sprintf("%s-git-%s-%s.%s", prefix, sluggify(branchName), workspaceSlug, apex),
				sticky: db.FrontlineRoutesStickyBranch,
			},
		)
	}

	domains = append(
		domains,
		newDomain{
			domain: fmt.Sprintf("%s-%s-%s.%s", prefix, environmentSlug, workspaceSlug, apex),
			sticky: db.FrontlineRoutesStickyEnvironment,
		},
	)

	if environmentSlug == "production" {
		domains = append(domains,
			newDomain{
				domain: fmt.Sprintf("%s-%s.%s", prefix, workspaceSlug, apex),
				sticky: db.FrontlineRoutesStickyLive,
			})
	}

	// deployment-specific domain for stable public access.
	sanitizedDeploymentID := strings.ReplaceAll(deploymentID, "_", "-")
	domains = append(domains, newDomain{
		domain: fmt.Sprintf("%s-dep-%s-%s.%s", prefix, sanitizedDeploymentID, workspaceSlug, apex),
		//nolint: exhaustruct
		sticky: db.FrontlineRoutesStickyDeployment,
	})

	return domains
}

var (
	// nonAlphanumericRegex removes characters that are unsafe for domain slugs and
	// avoids double hyphens when combined with whitespace normalization.
	nonAlphanumericRegex = regexp.MustCompile(`[^a-zA-Z0-9\s]`)

	// multipleSpacesRegex collapses consecutive whitespace before hyphen conversion.
	multipleSpacesRegex = regexp.MustCompile(`\s+`)
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
