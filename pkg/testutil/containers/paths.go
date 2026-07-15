package containers

import (
	"os"
	"path/filepath"
	"runtime"
)

func sourceRepoRoot() string {
	_, currentFile, _, ok := runtime.Caller(0)
	if ok {
		if root := findRepoRoot(filepath.Dir(currentFile)); root != "" {
			return root
		}
	}

	wd, err := os.Getwd()
	if err != nil {
		return "."
	}
	if root := findRepoRoot(wd); root != "" {
		return root
	}
	return wd
}

// dataPath locates a repository-relative data file, preferring the Bazel test
// runfiles tree before falling back to the source tree. All test data files
// (schemas, compose config) resolve through this single rule.
func dataPath(rel ...string) string {
	if runfiles := os.Getenv("TEST_SRCDIR"); runfiles != "" {
		if workspace := os.Getenv("TEST_WORKSPACE"); workspace != "" {
			candidate := filepath.Join(append([]string{runfiles, workspace}, rel...)...)
			if _, err := os.Stat(candidate); err == nil {
				return candidate
			}
		}
		candidate := filepath.Join(append([]string{runfiles, "_main"}, rel...)...)
		if _, err := os.Stat(candidate); err == nil {
			return candidate
		}
	}

	return filepath.Join(append([]string{sourceRepoRoot()}, rel...)...)
}

func findRepoRoot(start string) string {
	dir := filepath.Clean(start)
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return ""
		}
		dir = parent
	}
}
