// The lintsqlwildcards command rejects production queries that select every column.
package main

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
)

var (
	relevantSourcePattern = regexp.MustCompile(`(?i)\bselect\b|\bsqlc\s*\.|\.\s*query\s*\.|\.\s*select(?:All|Distinct)?\s*\(`)
	exemptTestPathPattern = regexp.MustCompile(`(?:^|/)(?:testdata|fixtures|__tests__)(?:/|$)|(?:_test\.go|\.(?:test|spec)\.[cm]?[jt]sx?|(?:_test|\.test|\.fixture)\.sql)$`)

	errViolations = errors.New("forbidden all-column SQL selections")
)

type reporter struct {
	stderr     io.Writer
	violations int
}

func main() {
	if err := run(".", os.Stderr); err != nil {
		if !errors.Is(err, errViolations) {
			fmt.Fprintln(os.Stderr, err)
		}
		os.Exit(1)
	}
}

// TODO: Replace this scanner with sqlc vet once the dashboard uses only the API,
// all production SQL is managed by sqlc, and vet exposes the original source.
// sqlc 1.30 expands wildcards before CEL rules evaluate query.sql.
func run(root string, stderr io.Writer) error {
	paths, err := sourceFiles(root)
	if err != nil {
		return err
	}

	r := reporter{stderr: stderr, violations: 0}
	for _, path := range paths {
		if isExemptTestPath(path) {
			continue
		}
		original, readErr := os.ReadFile(filepath.Join(root, filepath.FromSlash(path)))
		if readErr != nil {
			return fmt.Errorf("read %s: %w", path, readErr)
		}
		r.scanFile(path, original)
	}
	if r.violations == 0 {
		return nil
	}

	fmt.Fprintf(stderr, "Found %d all-column SQL selection(s). List columns explicitly.\n", r.violations)
	return errViolations
}

func sourceFiles(root string) ([]string, error) {
	cmd := exec.Command("git", "ls-files", "-z", "--cached", "--others", "--exclude-standard")
	cmd.Dir = root
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("list source files: %w", err)
	}

	parts := bytes.Split(output, []byte{0})
	paths := make([]string, 0, len(parts))
	for _, part := range parts {
		if len(part) > 0 {
			paths = append(paths, string(part))
		}
	}
	return paths, nil
}

func isExemptTestPath(path string) bool {
	return exemptTestPathPattern.MatchString(strings.ToLower(filepath.ToSlash(path)))
}

func (r *reporter) scanFile(path string, original []byte) {
	lower := strings.ToLower(path)
	if strings.HasSuffix(lower, ".sql") {
		source := maskSQL(original)
		for _, pattern := range migrationMasks[path] {
			source = pattern.ReplaceAllFunc(source, blank)
		}
		r.reportSQLWildcards(path, original, source)
		return
	}
	if !isSourceFile(lower) || !relevantSourcePattern.Match(original) {
		return
	}

	source := maskSourceComments(original, strings.HasSuffix(lower, ".go"))
	r.reportSQLWildcards(path, original, source)
	if strings.HasSuffix(lower, ".go") {
		return
	}

	if strings.HasPrefix(filepath.ToSlash(path), "web/apps/dashboard/") {
		r.reportDrizzleQueries(path, original)
	}
	r.reportQueryBuilders(path, original, source)
}

func isSourceFile(path string) bool {
	switch filepath.Ext(path) {
	case ".go", ".ts", ".tsx", ".js", ".jsx",
		".mts", ".mtsx", ".mjs", ".mjsx",
		".cts", ".ctsx", ".cjs", ".cjsx":
		return true
	default:
		return false
	}
}

func (r *reporter) report(path string, original []byte, offset int, kind string) {
	offset = min(max(offset, 0), len(original))
	line := 1 + bytes.Count(original[:offset], []byte{'\n'})
	lineEnd := bytes.IndexByte(original[offset:], '\n')
	if lineEnd < 0 {
		lineEnd = len(original) - offset
	}
	sourceLine := strings.TrimSpace(string(original[offset : offset+lineEnd]))
	fmt.Fprintf(r.stderr, "%s:%d: forbidden %s", path, line, kind)
	if sourceLine != "" {
		fmt.Fprintf(r.stderr, ": %s", sourceLine)
	}
	fmt.Fprintln(r.stderr)
	r.violations++
}
