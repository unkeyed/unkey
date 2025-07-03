package git

import (
	"os/exec"
	"strings"
)

// Info contains Git repository information
type Info struct {
	Branch    string // Current branch name (e.g., "main", "feature/auth")
	CommitSHA string // Full commit SHA (e.g., "abc123def456...")
	ShortSHA  string // Short commit SHA (e.g., "abc123d")
	IsDirty   bool   // Whether there are uncommitted changes
	IsRepo    bool   // Whether we're in a Git repository
}

// GetInfo safely extracts Git information from the current directory.
// It never fails - returns sensible defaults if Git is unavailable or we're not in a repo.
func GetInfo() Info {
	info := Info{
		Branch:    "main", // Default branch
		CommitSHA: "",     // Empty if not available
		ShortSHA:  "",     // Empty if not available
		IsDirty:   false,  // Assume clean if unknown
		IsRepo:    false,  // Assume not a repo until proven otherwise
	}

	// Check if we're in a Git repository
	if !isGitRepo() {
		return info
	}
	info.IsRepo = true

	// Get current branch
	if branch := getCurrentBranch(); branch != "" {
		info.Branch = branch
	}

	// Get commit SHA
	if sha := getCommitSHA(); sha != "" {
		info.CommitSHA = sha
		if len(sha) >= 7 {
			info.ShortSHA = sha[:7]
		}
	}

	// Check if working directory is dirty
	info.IsDirty = isWorkingDirDirty()

	return info
}

// isGitRepo checks if we're in a Git repository
func isGitRepo() bool {
	cmd := exec.Command("git", "rev-parse", "--git-dir")
	err := cmd.Run()
	return err == nil
}

// getCurrentBranch gets the current branch name
func getCurrentBranch() string {
	// Try to get branch name from HEAD
	cmd := exec.Command("git", "rev-parse", "--abbrev-ref", "HEAD")
	output, err := cmd.Output()
	if err != nil {
		return ""
	}

	branch := strings.TrimSpace(string(output))

	// If we're in detached HEAD state, try to get branch from describe
	if branch == "HEAD" {
		cmd = exec.Command("git", "describe", "--contains", "--all", "HEAD")
		output, err = cmd.Output()
		if err != nil {
			return ""
		}
		branch = strings.TrimSpace(string(output))

		// Clean up the output (remove refs/heads/ prefix if present)
		if strings.HasPrefix(branch, "heads/") {
			branch = strings.TrimPrefix(branch, "heads/")
		}
		if strings.HasPrefix(branch, "remotes/origin/") {
			branch = strings.TrimPrefix(branch, "remotes/origin/")
		}
	}

	return branch
}

// getCommitSHA gets the current commit SHA
func getCommitSHA() string {
	cmd := exec.Command("git", "rev-parse", "HEAD")
	output, err := cmd.Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(output))
}

// isWorkingDirDirty checks if there are uncommitted changes
func isWorkingDirDirty() bool {
	// Check for staged changes
	cmd := exec.Command("git", "diff-index", "--quiet", "--cached", "HEAD")
	if err := cmd.Run(); err != nil {
		return true // Has staged changes
	}

	// Check for unstaged changes
	cmd = exec.Command("git", "diff-files", "--quiet")
	if err := cmd.Run(); err != nil {
		return true // Has unstaged changes
	}

	// Check for untracked files
	cmd = exec.Command("git", "ls-files", "--others", "--exclude-standard")
	output, err := cmd.Output()
	if err != nil {
		return false // Assume clean if we can't check
	}

	return strings.TrimSpace(string(output)) != ""
}
