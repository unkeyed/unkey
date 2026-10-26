package deploy

import (
	"crypto/sha256"
	"fmt"
	"regexp"
	"strings"

	"github.com/unkeyed/unkey/svc/ctrl/internal/db"
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
// 1. Per-commit domain: `<prefix>-git-<sha>-<workspace>-<code>.<apex>`
//   - Never reassigned, provides stable URL for specific commit
//   - For CLI uploads, appends a deployment-ID-derived suffix to prevent collisions
//
// 2. Per-branch domain: `<prefix>-git-<branch>-<workspace>-<code>.<apex>`
//   - Sticky to branch, always points to latest deployment of that branch
//   - Branch name is sluggified for URL safety
//
// 3. Per-environment domain: `<prefix>-<environment>-<workspace>-<code>.<apex>`
//   - Sticky to environment, points to latest deployment in that environment
//   - Can be rolled back to previous deployment
//
// 4. Per-live domain (production only): `<prefix>-<workspace>-<code>.<apex>`
//   - Sticky to live, points to the active production deployment
//
// 5. Per-deployment domain: `<prefix>-<deploymentID>-<workspace>-<code>.<apex>`
//   - Never reassigned, provides a stable deployment-specific URL
//
// Where <prefix> is `<project>-<app>` for custom app slugs, or just `<project>`
// when the app slug is "default" (see TODO above).
// TODO: Once users can configure custom app slugs, include the app slug in all
// generated domains. Currently the only app slug is "default" which adds no
// useful information to the domain and just makes URLs longer. Remove this
// exclusion and always include appSlug once the dashboard supports renaming apps.
//
// `uniquifyCommitDomain` controls whether the per-commit domain gets an
// additional suffix derived from the deployment ID. CLI uploads send the same
// git SHA repeatedly across iterative `unkey deploy` runs, so without a suffix
// successive deploys collide on the per-commit domain. Git-driven deploys never
// repeat a SHA so they don't need it. Callers derive this from the deployment's
// trigger column.
//
// Every domain goes through [cappedDomain], so no label is longer than
// [dnsLabelLengthMax]
func buildDomains(
	workspaceSlug, projectSlug, appSlug, environmentSlug,
	gitSha, branchName, forkOwner, apex string,
	isProduction bool,
	uniquifyCommitDomain bool,
	deploymentID, appID string,
) []newDomain {
	code := appCode(appID)

	// Skip the "default" app slug since it's not configurable yet and would
	// just add noise to URLs (e.g. "myproject-default-..." vs "myproject-...").
	prefix := domainPrefix(projectSlug, appSlug)

	// Fork PRs include the fork owner in the domain to distinguish from same-repo
	// deployments (e.g. "myproject-fork-contributor-git-abc1234-acme.unkey.app").
	if forkOwner != "" {
		prefix = prefix + "-fork-" + sluggify(forkOwner)
	}

	// Deploying via CLI often sends the same git sha across deployments, so the
	// per-commit domain needs disambiguation. The suffix must be deterministic:
	// this function runs inside a Restate workflow, so re-rolling a random
	// value on replay would change the per-domain Run name and trigger a
	// journal mismatch. We take the tail of the deployment ID since each
	// deployment has a unique slug already.
	disambiguator := ""
	if uniquifyCommitDomain {
		tail := sluggify(deploymentID)
		if len(tail) > 4 {
			tail = tail[len(tail)-4:]
		}
		if tail != "" {
			disambiguator = "-" + tail
		}
	}

	var domains []newDomain

	if gitSha != "" {
		short := gitSha
		if len(short) > 7 {
			short = short[:7]
		}
		short += disambiguator
		domains = append(domains,
			newDomain{
				domain: cappedDomain(prefix+"-git-"+short, workspaceSlug, code, apex),
				sticky: db.FrontlineRoutesStickyNone,
			},
		)
	}

	if branchName != "" {
		domains = append(
			domains,
			newDomain{
				domain: cappedDomain(prefix+"-git-"+sluggify(branchName), workspaceSlug, code, apex),
				sticky: db.FrontlineRoutesStickyBranch,
			},
		)
	}

	domains = append(
		domains,
		newDomain{
			domain: cappedDomain(prefix+"-"+environmentSlug, workspaceSlug, code, apex),
			sticky: db.FrontlineRoutesStickyEnvironment,
		},
	)

	if isProduction {
		domains = append(domains,
			newDomain{
				domain: cappedDomain(prefix, workspaceSlug, code, apex),
				sticky: db.FrontlineRoutesStickyLive,
			})
	}

	// deployment-specific domain for stable public access.
	domains = append(domains, newDomain{
		domain: cappedDomain(prefix+"-"+sluggify(deploymentID), workspaceSlug, code, apex),
		sticky: db.FrontlineRoutesStickyDeployment,
	})

	return domains
}

// dnsLabelLengthMax is the length limit for one label of a domain name, from
// [RFC 1035 section 2.3.4]. DNS clients reject a longer label, so a domain
// with one never resolves
//
// [RFC 1035 section 2.3.4]: https://www.rfc-editor.org/rfc/rfc1035.html#section-2.3.4
const dnsLabelLengthMax = 63

// labelHashLength is the number of hex characters of the hash that
// [cappedDomain] puts in a cut label
const labelHashLength = 8

// cutPartsLength is what a cut label spends on everything except the head and
// the workspace slug: the hash, the app code, the three hyphens between them,
// and one character of head
const cutPartsLength = labelHashLength + appCodeLength + 4

// workspaceSlugLengthMax is the longest workspace slug a cut label keeps
const workspaceSlugLengthMax = dnsLabelLengthMax - cutPartsLength

// cappedDomain returns `<head>-<workspaceSlug>-<appCode>.<apex>`. A label
// longer than [dnsLabelLengthMax] becomes
// `<cut head>-<hash of full label>-<workspaceSlug>-<appCode>`.
// The hash keeps long labels apart and is stable, so a branch keeps its sticky
// domain across deploys.
//
// The app code is what makes a label unique to one app. The head is built from
// user-controlled text joined by hyphens, so two apps can build the same head:
// workspace "testing" on branch "main-x" and workspace "x-testing" on branch
// "main" both produce "git-main-x-testing". A label is also the unique key of a
// frontline route, so a shared label is a shared route, and the last deploy
// wins. The code comes from the app id, which no user chooses and no two apps
// share, so two apps cannot produce the same label.
func cappedDomain(head, workspaceSlug, code, apex string) string {
	label := head + "-" + workspaceSlug + "-" + code
	if len(label) <= dnsLabelLengthMax {
		return label + "." + apex
	}

	hash := fmt.Sprintf("%x", sha256.Sum256([]byte(label)))[:labelHashLength]
	tail := "-" + hash + "-" + cutLabel(workspaceSlug, workspaceSlugLengthMax) + "-" + code
	return cutLabel(head, dnsLabelLengthMax-len(tail)) + tail + "." + apex
}

// appCodeLength is the number of characters [appCode] returns. Six
// characters of a stable hash keep unrelated apps apart with room to spare
// inside a 63-character label, and leave the readable head long enough to stay
// recognizable.
const appCodeLength = 6

// appCode returns the fixed-length code that makes a domain label unique to one
// app. The value is stable for the life of the app, so a branch keeps the same
// sticky domain across deploys.
func appCode(appID string) string {
	return fmt.Sprintf("%x", sha256.Sum256([]byte("app:"+appID)))[:appCodeLength]
}

// cutLabel shortens s to at most length characters and drops the hyphens
// left at the cut
func cutLabel(s string, length int) string {
	return strings.TrimRight(s[:min(len(s), length)], "-")
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

	// Remove trailing hyphen if present
	s = strings.TrimSuffix(s, "-")

	return s
}
