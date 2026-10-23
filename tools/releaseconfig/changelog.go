package main

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"
)

func (analysis *reachability) changelog(cfg config, paths []string) (map[string]any, error) {
	if len(cfg.Changelog.Filters.Include) != 0 {
		return nil, errors.New("release changelog include filters are not supported")
	}
	result := map[string]any{"use": "git", "paths": paths}
	previous := os.Getenv("GORELEASER_PREVIOUS_TAG")
	if previous == "" {
		output, err := exec.Command("git", "describe", "--tags", "--abbrev=0", "--match", cfg.Monorepo.TagPrefix+"*", "HEAD^").Output()
		if err != nil {
			result["format"] = "{{ .SHA }} {{ .Message }} — dependency change; no previous release to analyze (conservative)"
			return result, nil
		}
		previous = strings.TrimSpace(string(output))
	}
	commits, err := gitOutput("rev-list", previous+"..HEAD")
	if err != nil {
		return nil, err
	}
	candidates := map[string]bool{}
	for commit := range strings.FieldsSeq(commits) {
		candidates[commit] = true
	}
	history, err := gitOutput(append([]string{"log", "--format=%H%x00%s", "HEAD", "--"}, paths...)...)
	if err != nil {
		return nil, err
	}
	type entry struct{ sha, subject, reason string }
	var entries []entry
	retained := map[string]bool{}
	for line := range strings.SplitSeq(strings.TrimSpace(history), "\n") {
		sha, subject, ok := strings.Cut(line, "\x00")
		if !ok {
			continue
		}
		if !candidates[sha] {
			retained[subject] = true
			continue
		}
		reason, err := analysis.commitReason(sha, paths)
		if err != nil {
			return nil, err
		}
		entries = append(entries, entry{sha, subject, reason})
		if reason != "" {
			retained[subject] = true
		}
	}
	excluded := slices.Clone(cfg.Changelog.Filters.Exclude)
	var format strings.Builder
	format.WriteString("{{ .SHA }} {{ .Message }} — ")
	for _, entry := range entries {
		if !retained[entry.subject] {
			excluded = append(excluded, "^"+regexp.QuoteMeta(entry.subject)+"$")
			continue
		}
		reason := entry.reason
		if reason == "" {
			reason = "duplicate commit subject (conservative)"
		}
		fmt.Fprintf(&format, "{{ if eq .SHA %s }}{{ %s }}{{ else }}", strconv.Quote(entry.sha), strconv.Quote(reason))
	}
	format.WriteString("dependency change outside analyzed range (conservative)")
	for _, entry := range entries {
		if retained[entry.subject] {
			format.WriteString("{{ end }}")
		}
	}
	slices.Sort(excluded)
	result["filters"] = map[string]any{"exclude": slices.Compact(excluded)}
	result["format"] = format.String()
	return result, nil
}

func (analysis *reachability) commitReason(sha string, paths []string) (string, error) {
	parents, err := gitOutput("rev-list", "--parents", "-n1", sha)
	if err != nil {
		return "", err
	}
	if len(strings.Fields(parents)) != 2 {
		return "merge or root commit (conservative)", nil
	}
	changes, err := gitOutput(append([]string{"diff-tree", "--no-commit-id", "--no-renames", "--name-status", "-z", "-r", sha, "--"}, paths...)...)
	if err != nil {
		return "", err
	}
	fields := strings.Split(strings.TrimSuffix(changes, "\x00"), "\x00")
	reason := ""
	var linked []string
	for index := 0; index+1 < len(fields); index += 2 {
		status, filename := fields[index], fields[index+1]
		candidate, err := analysis.fileReason(sha, status, filename)
		if err != nil {
			return "", err
		}
		if strings.HasPrefix(candidate, "linked ") {
			linked = append(linked, candidate)
		}
		if reason == "" && candidate != "" {
			reason = filename + ": " + candidate
		}
	}
	if len(linked) != 0 {
		return strings.Join(linked[:min(3, len(linked))], "; "), nil
	}
	return reason, nil
}

func (analysis *reachability) fileReason(sha, status, filename string) (string, error) {
	embedded := false
	assetPossible := status == "D"
	for dir, pkg := range analysis.packages {
		relative, err := filepath.Rel(dir, filename)
		if err != nil {
			return "", err
		}
		for _, pattern := range pkg.EmbedPatterns {
			for candidate := filepath.ToSlash(relative); candidate != "." && candidate != ".."; candidate = path.Dir(candidate) {
				matched, err := path.Match(strings.TrimPrefix(pattern, "all:"), candidate)
				if err != nil {
					return "", err
				}
				assetPossible = assetPossible || matched
			}
		}
		if slices.Contains(pkg.EmbedFiles, filepath.ToSlash(relative)) {
			embedded = true
			linked, err := analysis.embedded(dir, pkg)
			if err != nil {
				return "", err
			}
			if linked {
				return "embedded asset in linked package (conservative)", nil
			}
		}
	}
	if embedded {
		return "", nil
	}
	if strings.HasSuffix(filename, "_test.go") {
		return "", nil
	}
	pkg, ok := analysis.packages[filepath.ToSlash(filepath.Dir(filename))]
	if !strings.HasSuffix(filename, ".go") {
		switch filepath.Ext(filename) {
		case ".md", ".mdx":
			if !assetPossible {
				return "", nil
			}
		}
		return "build input or asset (conservative)", nil
	}
	if !ok {
		return "removed dependency (conservative)", nil
	}
	if !analysis.files[filename] {
		return "unselected or removed Go source (conservative)", nil
	}
	var before, after string
	var err error
	if status != "A" {
		before, err = gitOutput("show", sha+"^:"+filename)
		if err != nil {
			return "", err
		}
	}
	if status != "D" {
		after, err = gitOutput("show", sha+":"+filename)
		if err != nil {
			return "", err
		}
	}
	return analysis.changedGo([]byte(before), []byte(after), pkg), nil
}
