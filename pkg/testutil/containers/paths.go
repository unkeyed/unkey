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
