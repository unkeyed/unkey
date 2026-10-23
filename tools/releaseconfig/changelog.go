package main

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"
)

type changelogEntry struct {
	sha     string
	subject string
	reason  string
}

type changeImpact struct {
	symbol string
	reason string
}

func (impact changeImpact) String() string {
	if impact.symbol != "" {
		return "linked `" + impact.symbol + "`"
	}
	return impact.reason
}

func (analysis *reachability) changelog(cfg config, paths []string) (changelogConfig, error) {
	result := changelogConfig{Use: "git", Paths: paths, Format: "", Filters: nil}
	if len(cfg.Changelog.Filters.Include) != 0 {
		return result, errors.New("release changelog include filters are not supported")
	}
	previous := os.Getenv("GORELEASER_PREVIOUS_TAG")
	if previous == "" {
		output, err := exec.Command("git", "describe", "--tags", "--abbrev=0", "--match", cfg.Monorepo.TagPrefix+"*", "HEAD^").Output()
		if err != nil {
			result.Format = "{{ .SHA }} {{ .Message }} — dependency change; no previous release to analyze (conservative)"
			return result, nil
		}
		previous = strings.TrimSpace(string(output))
	}

	entries, err := analysis.analyzeHistory(previous, paths)
	if err != nil {
		return result, err
	}
	excluded := slices.Clone(cfg.Changelog.Filters.Exclude)
	for _, entry := range entries {
		if entry.reason == "" {
			excluded = append(excluded, "^"+regexp.QuoteMeta(entry.subject)+"$")
		}
	}
	slices.Sort(excluded)
	result.Filters = &changelogFilters{Exclude: slices.Compact(excluded), Include: nil}
	result.Format = changelogFormat(entries)
	return result, nil
}

func (analysis *reachability) analyzeHistory(previous string, paths []string) ([]changelogEntry, error) {
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
	var entries []changelogEntry
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
		entries = append(entries, changelogEntry{sha: sha, subject: subject, reason: reason})
		if reason != "" {
			retained[subject] = true
		}
	}
	for index := range entries {
		entry := &entries[index]
		if entry.reason == "" && retained[entry.subject] {
			entry.reason = "duplicate commit subject (conservative)"
		}
	}
	return entries, nil
}

func changelogFormat(entries []changelogEntry) string {
	var format strings.Builder
	format.WriteString("{{ .SHA }} {{ .Message }} — ")
	branches := 0
	for _, entry := range entries {
		if entry.reason == "" {
			continue
		}
		fmt.Fprintf(&format, "{{ if eq .SHA %s }}{{ %s }}{{ else }}", strconv.Quote(entry.sha), strconv.Quote(entry.reason))
		branches++
	}
	format.WriteString("dependency change outside analyzed range (conservative)")
	format.WriteString(strings.Repeat("{{ end }}", branches))
	return format.String()
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
		impact, err := analysis.fileImpact(sha, status, filename)
		if err != nil {
			return "", err
		}
		if impact.symbol != "" {
			linked = append(linked, impact.String())
		}
		if reason == "" && impact.reason != "" {
			reason = filename + ": " + impact.reason
		}
	}
	if len(linked) != 0 {
		return strings.Join(linked[:min(3, len(linked))], "; "), nil
	}
	return reason, nil
}

func (analysis *reachability) fileImpact(sha, status, filename string) (changeImpact, error) {
	asset, err := analysis.assetUsage(filename)
	if err != nil {
		return changeImpact{}, err
	}
	if asset.linked {
		return changeImpact{symbol: "", reason: "embedded asset in linked package (conservative)"}, nil
	}
	if asset.embedded || strings.HasSuffix(filename, "_test.go") {
		return changeImpact{symbol: "", reason: ""}, nil
	}
	if !strings.HasSuffix(filename, ".go") {
		switch filepath.Ext(filename) {
		case ".md", ".mdx":
			if !asset.possible && status != "D" {
				return changeImpact{symbol: "", reason: ""}, nil
			}
		}
		return changeImpact{symbol: "", reason: "build input or asset (conservative)"}, nil
	}

	pkg, ok := analysis.packages[filepath.ToSlash(filepath.Dir(filename))]
	if !ok {
		return changeImpact{symbol: "", reason: "removed dependency (conservative)"}, nil
	}
	if !analysis.files[filename] {
		return changeImpact{symbol: "", reason: "unselected or removed Go source (conservative)"}, nil
	}
	before, after, err := gitFileVersions(sha, status, filename)
	if err != nil {
		return changeImpact{}, err
	}
	return analysis.changedGo([]byte(before), []byte(after), pkg), nil
}

func gitFileVersions(sha, status, filename string) (string, string, error) {
	var before, after string
	var err error
	if status != "A" {
		before, err = gitOutput("show", sha+"^:"+filename)
		if err != nil {
			return "", "", err
		}
	}
	if status != "D" {
		after, err = gitOutput("show", sha+":"+filename)
		if err != nil {
			return "", "", err
		}
	}
	return before, after, nil
}

func gitOutput(args ...string) (string, error) {
	output, err := exec.Command("git", args...).CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("git %s: %s: %w", strings.Join(args, " "), output, err)
	}
	return string(output), nil
}
