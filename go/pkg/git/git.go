package git

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

// Info contains comprehensive Git repo and commit information
type Info struct {
	// Basic repo status
	Branch  string // Current branch name (e.g., "main", "feature/auth")
	IsDirty bool   // Whether there are uncommitted changes
	IsRepo  bool   // Whether we're in a Git repo

	// Commit identification
	CommitSHA string // Full commit SHA (e.g., "abc123def456...")
	ShortSHA  string // Short commit SHA (e.g., "abc123d")

	// Commit details
	Message         string // Commit message (first line only)
	AuthorHandle    string // Author's GitHub handle
	AuthorAvatarURL string // URL to author's avatar image
	CommitTimestamp int64  // Unix timestamp of the commit in milliseconds
}

// githubCommitResponse represents the GitHub API commit response
type githubCommitResponse struct {
	Author struct {
		Login     string `json:"login"`
		AvatarURL string `json:"avatar_url"`
	} `json:"author"`
}

// GetInfo safely extracts Git information from the current directory.
// It never fails - returns sensible defaults if Git is unavailable or we're not in a repo.
// Extended commit details (message, author, timestamp) are populated when available.
func GetInfo() Info {
	info := Info{
		Branch:  "main", // Default branch
		IsDirty: false,  // Assume clean if unknown
		IsRepo:  false,  // Assume not a repo until proven otherwise
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

	// Get extended commit details (best effort - ignore errors)
	if info.CommitSHA != "" {
		// Get commit message (first line only)
		if message, err := execGitCommand("git", "log", "-1", "--pretty=%s"); err == nil {
			info.Message = message
		}

		// Get commit timestamp
		if timestampStr, err := execGitCommand("git", "log", "-1", "--pretty=%ct"); err == nil {
			if timestamp, err := strconv.ParseInt(timestampStr, 10, 64); err == nil {
				info.CommitTimestamp = timestamp * 1000 // Convert to milliseconds
			}
		}

		// Get remote URL to determine if it's a GitHub repo
		if remoteURL, err := execGitCommand("git", "config", "--get", "remote.origin.url"); err == nil && isGitHubURL(remoteURL) {
			// Extract owner and repo from GitHub URL
			owner, repo := parseGitHubURL(remoteURL)
			if owner != "" && repo != "" {
				// Fetch author info from GitHub API (best effort)
				handle, avatarURL := fetchGitHubAuthorInfo(owner, repo, info.CommitSHA)
				info.AuthorHandle = handle
				info.AuthorAvatarURL = avatarURL
			}
		}
	}

	return info
}

// isGitHubURL checks if the URL is a GitHub repository URL
func isGitHubURL(url string) bool {
	return strings.Contains(url, "github.com")
}

// parseGitHubURL extracts owner and repo name from GitHub URL
// Supports both HTTPS and SSH formats:
// - https://github.com/owner/repo.git
// - git@github.com:owner/repo.git
func parseGitHubURL(url string) (owner, repo string) {
	url = strings.TrimSpace(url)

	// Remove .git suffix
	url = strings.TrimSuffix(url, ".git")

	// Handle SSH format: git@github.com:owner/repo
	if after, ok := strings.CutPrefix(url, "git@github.com:"); ok {
		path := after
		parts := strings.Split(path, "/")
		if len(parts) == 2 {
			return parts[0], parts[1]
		}
	}

	// Handle HTTPS format: https://github.com/owner/repo
	if strings.Contains(url, "github.com/") {
		parts := strings.Split(url, "github.com/")
		if len(parts) == 2 {
			pathParts := strings.Split(parts[1], "/")
			if len(pathParts) >= 2 {
				return pathParts[0], pathParts[1]
			}
		}
	}

	return "", ""
}

// TODO: We'll have something smarter after demo. As long as we are demoing in a pushed repo we are good.
// fetchGitHubAuthorInfo fetches the commit author's GitHub handle and avatar from GitHub API
func fetchGitHubAuthorInfo(owner, repo, sha string) (string, string) {
	url := fmt.Sprintf("https://api.github.com/repos/%s/%s/commits/%s", owner, repo, sha)

	client := &http.Client{
		Timeout: 5 * time.Second,
	}

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return "", ""
	}

	// Set User-Agent header (required by GitHub API)
	req.Header.Set("User-Agent", "unkey-cli")

	resp, err := client.Do(req)
	if err != nil {
		return "", ""
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", ""
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", ""
	}

	var commitData githubCommitResponse
	if err := json.Unmarshal(body, &commitData); err != nil {
		return "", ""
	}

	return commitData.Author.Login, commitData.Author.AvatarURL
}

// isGitRepo checks if we're in a Git repository
func isGitRepo() bool {
	_, err := execGitCommand("git", "rev-parse", "--git-dir")
	return err == nil
}

// getCurrentBranch gets the current branch name
func getCurrentBranch() string {
	// Try to get branch name from HEAD
	branch, err := execGitCommand("git", "rev-parse", "--abbrev-ref", "HEAD")
	if err != nil {
		return ""
	}
	// If we're in detached HEAD state, try to get branch from describe
	if branch == "HEAD" {
		describeBranch, describeErr := execGitCommand("git", "describe", "--contains", "--all", "HEAD")
		if describeErr != nil {
			return ""
		}
		branch = describeBranch
		branch = strings.TrimPrefix(branch, "heads/")
		branch = strings.TrimPrefix(branch, "remotes/origin/")
	}

	return branch
}

// getCommitSHA gets the current commit SHA
func getCommitSHA() string {
	sha, err := execGitCommand("git", "rev-parse", "HEAD")
	if err != nil {
		return ""
	}
	return sha
}

// isWorkingDirDirty checks if there are uncommitted changes
func isWorkingDirDirty() bool {
	// Check for staged changes
	_, err := execGitCommand("git", "diff-index", "--quiet", "--cached", "HEAD")
	if err != nil {
		return true
	}
	// Check for unstaged changes
	_, err = execGitCommand("git", "diff-files", "--quiet")
	if err != nil {
		return true
	}
	// Check for untracked files
	untrackedOutput, untrackedErr := execGitCommand("git", "ls-files", "--others", "--exclude-standard")
	if untrackedErr != nil {
		return false
	}

	return untrackedOutput != ""
}

// execGitCommand executes a git command and returns trimmed output
func execGitCommand(args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	output, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return "", fmt.Errorf("command failed: %s, stderr: %s", err, string(exitErr.Stderr))
		}
		return "", err
	}
	return strings.TrimSpace(string(output)), nil
}
