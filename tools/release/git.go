package main

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// gitRun runs a git command, streaming output to the terminal.
func gitRun(args ...string) error {
	cmd := exec.Command("git", args...)
	cmd.Stdout = os.Stderr
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// gitOutput runs a git command and returns its trimmed stdout.
func gitOutput(args ...string) (string, error) {
	out, err := exec.Command("git", args...).Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

// versionCache memoizes parsed tags per service for a single run.
var versionCache = map[string][]semver{}

// listServiceVersions returns all semantic versions tagged for a service.
func listServiceVersions(service string) []semver {
	if cached, ok := versionCache[service]; ok {
		return cached
	}
	out, err := gitOutput("tag", "-l", service+"/v*")
	if err != nil || out == "" {
		versionCache[service] = nil
		return nil
	}

	var versions []semver
	for _, line := range strings.Split(out, "\n") {
		raw := strings.TrimPrefix(strings.TrimSpace(line), service+"/")
		if v, ok := parseSemver(raw); ok {
			versions = append(versions, v)
		}
	}
	versionCache[service] = versions
	return versions
}

// commitHashes returns the set of full commit hashes in baseline..HEAD.
func commitHashes(baseline string) map[string]bool {
	set := map[string]bool{}
	out, err := gitOutput("log", "--no-merges", "--pretty=format:%H", baseline+"..HEAD")
	if err != nil || strings.TrimSpace(out) == "" {
		return set
	}
	for _, h := range strings.Split(out, "\n") {
		if h = strings.TrimSpace(h); h != "" {
			set[h] = true
		}
	}
	return set
}

// verifyRepoState enforces CI's rules: stable tags must be on origin/main and
// no tag may already exist. Pre-release tags are allowed from any branch.
func verifyRepoState(tags []plannedTag) error {
	onMain := gitRun("merge-base", "--is-ancestor", "HEAD", "origin/main") == nil
	if !onMain {
		if hasStableTag(tags) {
			return fmt.Errorf("HEAD is not reachable from origin/main; stable release tags must be on main (use --rc/--pre to release a pre-release from a branch)")
		}
		branch, _ := gitOutput("rev-parse", "--abbrev-ref", "HEAD")
		fmt.Printf("note: tagging pre-release(s) from %q (not on main); CI will build but this is not a stable release.\n", branch)
	}

	for _, t := range tags {
		if tagExists(t.String()) {
			return fmt.Errorf("tag %s already exists; tags are immutable, bump the version instead", t)
		}
	}
	return nil
}

// tagExists checks for an existing tag locally (origin tags arrive via fetch).
func tagExists(tag string) bool {
	_, err := gitOutput("rev-parse", "--verify", "--quiet", "refs/tags/"+tag)
	return err == nil
}

// createAndPush creates a lightweight tag per entry and pushes each separately.
// GitHub drops tag push events when more than three are pushed at once, so a
// combined push would silently fail to trigger CI.
func createAndPush(tags []plannedTag) error {
	created := make([]plannedTag, 0, len(tags))
	for _, t := range tags {
		// tag.gpgsign=false avoids a signed annotated tag (and its editor
		// prompt) when the user has tag signing enabled globally.
		if err := gitRun("-c", "tag.gpgsign=false", "tag", t.String()); err != nil {
			rollback(created)
			return fmt.Errorf("creating tag %s: %w", t, err)
		}
		created = append(created, t)
	}

	// One push per tag; see the function doc for why.
	for _, t := range tags {
		if err := gitRun("push", "--no-progress", "origin", t.String()); err != nil {
			fmt.Fprintln(os.Stderr, "push failed; local tags were created but not all pushed.")
			fmt.Fprintln(os.Stderr, "fix the issue and re-run `git push origin "+strings.Join(tagNames(tags), " ")+"` (one at a time)")
			return fmt.Errorf("pushing tag %s: %w", t, err)
		}
	}

	fmt.Printf("\nPushed %d tag(s). Watch CI:\n", len(tags))
	fmt.Println("  depot ci run list --repo unkeyed/unkey --trigger push --status running")
	return nil
}

// rollback deletes locally created tags after a partial failure.
func rollback(created []plannedTag) {
	for _, t := range created {
		_ = gitRun("tag", "-d", t.String())
	}
	if len(created) > 0 {
		fmt.Fprintln(os.Stderr, "rolled back locally created tags.")
	}
}
