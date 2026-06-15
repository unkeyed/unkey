package main

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"strings"
)

// options is the fully-resolved input to a release run.
type options struct {
	version  string   // explicit version pinned for all bare service args (optional)
	bump     bumpKind // stable increment used when auto-numbering
	pre      string   // pre-release label (e.g. "rc"); empty for a stable release
	services []string // service args (bare name, service@version, or service/vX.Y.Z)
	dryRun   bool
	yes      bool
	noFetch  bool
	noLog    bool
	noVerify bool
}

// release plans the tags, previews them, and (unless dry-run) creates and
// pushes them.
func release(opts options) error {
	if _, err := gitOutput("rev-parse", "--git-dir"); err != nil {
		return errors.New("not inside a git repository")
	}

	// Fetch tags up front so auto-numbering and existence checks see the
	// authoritative state on origin. The explicit "+refs/tags/*" refspec
	// force-updates any stale local tags that have diverged from origin; a
	// plain "--tags" (even with --force, which only covers the named refspec)
	// is rejected with "would clobber existing tag" and numbering would be
	// wrong anyway. origin is the source of truth and its tags are immutable.
	if !opts.noFetch {
		if err := gitRun("fetch", "--no-progress", "origin", "main", "+refs/tags/*:refs/tags/*"); err != nil {
			return fmt.Errorf("fetching origin/main and tags: %w", err)
		}
	}

	tags, err := buildTags(opts)
	if err != nil {
		return err
	}

	if !opts.noVerify {
		if err := verifyRepoState(tags); err != nil {
			return err
		}
	}

	head, err := gitOutput("rev-parse", "--short", "HEAD")
	if err != nil {
		return err
	}

	fmt.Printf("About to tag commit %s with:\n", head)
	for _, t := range tags {
		fmt.Printf("  - %s\n", t)
	}
	if !opts.noLog {
		printChangelog(tags)
	}

	if opts.dryRun {
		fmt.Println("\ndry-run: no tags created or pushed.")
		return nil
	}
	if !opts.yes && !confirm() {
		fmt.Println("aborted.")
		return nil
	}

	return createAndPush(tags)
}

// confirm prompts the user to proceed and returns true only on an affirmative
// answer.
func confirm() bool {
	fmt.Print("\nProceed? [y/N] ")
	line, err := bufio.NewReader(os.Stdin).ReadString('\n')
	if err != nil {
		return false
	}
	answer := strings.ToLower(strings.TrimSpace(line))
	return answer == "y" || answer == "yes"
}
