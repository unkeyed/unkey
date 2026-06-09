package main

import (
	"fmt"
	"regexp"
	"strings"
)

// prRefRe matches the `(#NNNN)` PR reference squash-merges leave in the subject.
var prRefRe = regexp.MustCompile(`\(#(\d+)\)`)

// changelogLimit caps how many commits are listed in the preview.
const changelogLimit = 20

// serviceRange is the set of commits a service's release will newly ship,
// i.e. everything since its last stable tag.
type serviceRange struct {
	service  string
	baseline string
	hashes   map[string]bool
}

// printChangelog prints a deduplicated preview of the commits merged since each
// service's last stable release, annotating each with the services it ships in.
func printChangelog(tags []plannedTag) {
	var ranges []serviceRange
	widest := serviceRange{service: "", baseline: "", hashes: nil}
	for _, service := range uniqueServices(tags) {
		latest, found := maxStable(listServiceVersions(service))
		if !found {
			fmt.Printf("\n%s: no previous stable release; full history will ship.\n", service)
			continue
		}
		r := serviceRange{
			service:  service,
			baseline: service + "/" + formatVersion(latest),
			hashes:   commitHashes(service + "/" + formatVersion(latest)),
		}
		ranges = append(ranges, r)
		if len(r.hashes) > len(widest.hashes) {
			widest = r
		}
	}
	if len(ranges) == 0 {
		return
	}

	// The widest range (oldest baseline) is a superset on linear history, so
	// its ordered log gives a stable order covering every commit.
	out, err := gitOutput("log", "--no-merges", "--pretty=format:%H%x1f%h %s", widest.baseline+"..HEAD")
	if err != nil || strings.TrimSpace(out) == "" {
		fmt.Println("\nNo new commits since the last release.")
		return
	}

	lines := strings.Split(out, "\n")
	fmt.Printf("\nChanges since last release (%d unique commit(s)):\n", len(lines))
	for i, line := range lines {
		if i == changelogLimit {
			fmt.Printf("  ... and %d more\n", len(lines)-changelogLimit)
			break
		}
		full, disp, ok := strings.Cut(line, "\x1f")
		if !ok {
			continue
		}
		svcs := strings.Join(servicesFor(full, ranges), ", ")
		if url := prURL(disp); url != "" {
			fmt.Printf("  %s  %s  [%s]\n", disp, url, svcs)
		} else {
			fmt.Printf("  %s  [%s]\n", disp, svcs)
		}
	}
}

// prURL returns the GitHub PR URL referenced in a commit subject, if any.
func prURL(subject string) string {
	m := prRefRe.FindStringSubmatch(subject)
	if m == nil {
		return ""
	}
	return "https://github.com/unkeyed/unkey/pull/" + m[1]
}

// servicesFor returns the services whose release range includes the commit.
func servicesFor(hash string, ranges []serviceRange) []string {
	var out []string
	for _, r := range ranges {
		if r.hashes[hash] {
			out = append(out, r.service)
		}
	}
	return out
}
