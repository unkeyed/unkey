// Package githubapp holds the shared logic for connecting a GitHub repository
// to an app, used by both apps.createApp and apps.updateApp. It resolves which
// of a workspace's GitHub App installations can access a repository and returns
// enough to persist the connection; the callers own the writes so they stay in
// their own transactions.
package githubapp

import (
	"errors"
	"fmt"
	"strings"

	"github.com/oapi-codegen/nullable"
	"github.com/unkeyed/unkey/pkg/codes"
	"github.com/unkeyed/unkey/pkg/fault"
	"github.com/unkeyed/unkey/svc/api/openapi"
	github "github.com/unkeyed/unkey/svc/ctrl/worker/github"
)

// GitState builds the App.git response value. An empty repositoryFullName (no
// connection) yields JSON null; otherwise the repository and the branch the app
// tracks.
func GitState(repositoryFullName string, defaultBranch string) nullable.Nullable[openapi.AppGit] {
	var state nullable.Nullable[openapi.AppGit]
	if repositoryFullName == "" {
		state.SetNull()
		return state
	}
	branch := defaultBranch
	state.Set(openapi.AppGit{Repository: repositoryFullName, DefaultBranch: &branch})
	return state
}

// Resolved is the outcome of matching a repository to a workspace installation.
type Resolved struct {
	// InstallationID is the installation that grants access to the repository.
	InstallationID int64

	// Repository is the verified repository metadata straight from GitHub.
	Repository github.RepoInfo
}

// Resolve picks the installation (from the workspace's installations) that can
// access repository ("owner/repo") and verifies access against the GitHub API.
// It performs no database work; callers load the installations and own the
// writes, so this stays a pure, transaction-free access check.
//
// GitHub must be configured before calling (appName non-empty); callers gate on
// that to return a clean "not enabled" error. appName is used only to build the
// actionable install URL in user-facing errors.
func Resolve(
	client github.GitHubClient,
	appName string,
	installations []int64,
	repository string,
) (Resolved, error) {
	repository, err := normalizeRepository(repository)
	if err != nil {
		return Resolved{}, fault.Wrap(err,
			fault.Code(codes.App.Validation.InvalidInput.URN()),
			fault.Internal("could not normalize repository"),
			fault.Public("Provide the repository as \"owner/repo\"."),
		)
	}

	if len(installations) == 0 {
		return Resolved{}, fault.New("no github installation",
			fault.Code(codes.App.Precondition.PreconditionFailed.URN()),
			fault.Internal("workspace has no github app installation"),
			fault.Public(fmt.Sprintf(
				"The Unkey GitHub App is not installed for this workspace. Install it at https://github.com/apps/%s/installations/new with access to %q, then try again.",
				appName, repository,
			)),
		)
	}

	// An installation token is scoped to only the repositories the installation
	// was granted, so GetInstallationRepo doubles as the access check. A GitHub
	// "owner/repo" is globally unique, so at most one installation can grant it.
	var verifyErrs []error
	for _, installationID := range installations {
		info, gErr := client.GetInstallationRepo(installationID, repository)
		if gErr != nil {
			verifyErrs = append(verifyErrs, fmt.Errorf("installation %d: %w", installationID, gErr))
			continue
		}
		if info != nil {
			return Resolved{InstallationID: installationID, Repository: *info}, nil
		}
	}

	// A failed lookup makes "not accessible" unprovable (the errored installation
	// may have been the one that grants access), so collect the errors and, if
	// none resolved, report a verification failure rather than a hard "no access".
	if len(verifyErrs) > 0 {
		return Resolved{}, fault.Wrap(errors.Join(verifyErrs...),
			fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
			fault.Internal("failed to verify github repository access"),
			fault.Public("Failed to verify access to the GitHub repository."),
		)
	}

	return Resolved{}, fault.New("repository not accessible",
		fault.Code(codes.App.Precondition.PreconditionFailed.URN()),
		fault.Internal("no installation can access the repository"),
		fault.Public(fmt.Sprintf(
			"The Unkey GitHub App cannot access %q. Grant it access at https://github.com/apps/%s/installations/new, then try again.",
			repository, appName,
		)),
	)
}

// DefaultBranch returns the branch to track: the caller's override when it is
// set and non-empty, otherwise the given fallback (a fresh connect passes the
// repository's GitHub default; a replace passes the app's current branch).
func DefaultBranch(fallback string, override *string) string {
	if override != nil && *override != "" {
		return *override
	}
	return fallback
}

// normalizeRepository reduces a user-supplied repository reference to a clean
// "owner/repo". It accepts the bare form, an HTTPS URL, an SSH remote, and a
// browser URL with extra path segments or a query/fragment, e.g.:
//
//	unkeyed/unkey
//	https://github.com/unkeyed/unkey(.git)
//	git@github.com:unkeyed/unkey.git
//	https://github.com/unkeyed/unkey/tree/main?tab=readme
//
// It returns an error when the result is not a two-part "owner/repo". The
// github.com host is stripped only at a real host boundary (followed by "/" or
// ":") so an owner or repo name that merely contains "github.com" is left alone.
func normalizeRepository(repository string) (string, error) {
	repository = strings.TrimSpace(repository)

	if i := strings.LastIndex(repository, "github.com"); i >= 0 {
		rest := repository[i+len("github.com"):]
		if strings.HasPrefix(rest, "/") || strings.HasPrefix(rest, ":") {
			repository = strings.TrimLeft(rest, "/:")
		}
	}

	// Drop a query string or fragment copied from the browser address bar.
	if i := strings.IndexAny(repository, "?#"); i >= 0 {
		repository = repository[:i]
	}

	repository = strings.Trim(repository, "/")
	repository = strings.TrimSuffix(repository, ".git")

	parts := strings.Split(repository, "/")
	if len(parts) < 2 || parts[0] == "" || parts[1] == "" {
		return "", fmt.Errorf("invalid repository %q, expected \"owner/repo\"", repository)
	}
	return parts[0] + "/" + parts[1], nil
}
