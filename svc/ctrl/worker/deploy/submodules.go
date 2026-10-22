package deploy

import (
	"fmt"
	"net/url"
	"path"
	"strings"

	"github.com/unkeyed/unkey/pkg/logger"
)

// privateSubmoduleRepos returns the private repositories that repo declares as
// submodules at ref and that the GitHub App installation can read. The clone
// token must cover them, because BuildKit sends the same token to every
// https://github.com/ remote when it runs `git submodule update`.
//
// Public submodules need no scope. Submodules that the installation cannot
// read are skipped, because GitHub rejects a token request that lists them;
// the clone of such a submodule then fails with a submodule error.
func (w *Workflow) privateSubmoduleRepos(installationID int64, repo, ref string) ([]string, error) {
	gitmodules, found, err := w.github.GetFileContents(installationID, repo, ref, ".gitmodules")
	if err != nil {
		return nil, fmt.Errorf("failed to read .gitmodules of %s: %w", repo, err)
	}
	if !found {
		return nil, nil
	}

	var private []string
	for _, submodule := range parseSubmoduleRepos(string(gitmodules), repo) {
		if strings.EqualFold(submodule, repo) {
			continue
		}
		// An installation covers one account, so a token cannot reach a
		// repository of another owner.
		if !strings.EqualFold(repoOwner(submodule), repoOwner(repo)) {
			continue
		}

		info, err := w.github.GetInstallationRepo(installationID, submodule)
		if err != nil {
			return nil, fmt.Errorf("failed to look up submodule repository %s: %w", submodule, err)
		}
		if info == nil {
			logger.Warn("submodule repository is not accessible to the GitHub App installation",
				"repository", repo, "submodule", submodule)
			continue
		}
		if info.Private {
			private = append(private, info.FullName)
		}
	}
	return private, nil
}

// parseSubmoduleRepos returns the GitHub repositories ("owner/repo") that the
// url entries of a .gitmodules file point to. Relative URLs resolve against
// superRepo, like git resolves them against the superproject's remote. Only
// https://github.com/ and relative URLs are returned, because BuildKit sends
// the clone token only to https://github.com/ remotes.
func parseSubmoduleRepos(gitmodules, superRepo string) []string {
	var repos []string
	for line := range strings.Lines(gitmodules) {
		key, value, ok := strings.Cut(line, "=")
		if !ok || !strings.EqualFold(strings.TrimSpace(key), "url") {
			continue
		}
		value = strings.Trim(strings.TrimSpace(value), `"`)

		var repoPath string
		if strings.HasPrefix(value, "./") || strings.HasPrefix(value, "../") {
			repoPath = path.Join("/"+superRepo+".git", value)
		} else {
			u, err := url.Parse(value)
			if err != nil || u.Scheme != "https" || (u.Host != "github.com" && u.Host != "www.github.com") {
				continue
			}
			repoPath = u.Path
		}

		fullName := strings.TrimSuffix(strings.Trim(repoPath, "/"), ".git")
		if isValidRepoFullName(fullName) {
			repos = append(repos, fullName)
		}
	}
	return repos
}
