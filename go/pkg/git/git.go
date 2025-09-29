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

// Info contains Git repository information
type Info struct {
	Branch    string // Current branch name (e.g., "main", "feature/auth")
	CommitSHA string // Full commit SHA (e.g., "abc123def456...")
	ShortSHA  string // Short commit SHA (e.g., "abc123d")
	IsDirty   bool   // Whether there are uncommitted changes
	IsRepo    bool   // Whether we're in a Git repository
}

// CommitInfo contains detailed information about the current commit
type CommitInfo struct {
	SHA             string // Full commit SHA
	Branch          string // Current branch name
	Message         string // Commit message (first line only)
	AuthorHandle    string // Author's GitHub handle
	AuthorAvatarURL string // URL to author's avatar image
	CommitTimestamp int64  // Unix timestamp of the commit
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

// GetCommitInfo retrieves detailed information about the current commit.
// Returns error if not in a git repository or if git commands fail.
func GetCommitInfo() (*CommitInfo, error) {
	info := &CommitInfo{}

	// Get commit SHA
	sha, err := execGitCommand("git", "rev-parse", "HEAD")
	if err != nil {
		return nil, fmt.Errorf("failed to get commit SHA: %w", err)
	}
	info.SHA = sha

	// Get current branch
	branch, err := execGitCommand("git", "rev-parse", "--abbrev-ref", "HEAD")
	if err != nil {
		return nil, fmt.Errorf("failed to get branch: %w", err)
	}
	info.Branch = branch

	// Get commit message (first line only)
	message, err := execGitCommand("git", "log", "-1", "--pretty=%s")
	if err != nil {
		return nil, fmt.Errorf("failed to get commit message: %w", err)
	}
	info.Message = message

	// Get commit timestamp
	timestampStr, err := execGitCommand("git", "log", "-1", "--pretty=%ct")
	if err != nil {
		return nil, fmt.Errorf("failed to get commit timestamp: %w", err)
	}
	info.CommitTimestamp, err = strconv.ParseInt(timestampStr, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("failed to parse timestamp: %w", err)
	}
	info.CommitTimestamp = info.CommitTimestamp * 1000

	// Get remote URL to determine if it's a GitHub repo
	remoteURL, err := execGitCommand("git", "config", "--get", "remote.origin.url")
	if err == nil && isGitHubURL(remoteURL) {
		// Extract owner and repo from GitHub URL
		owner, repo := parseGitHubURL(remoteURL)
		if owner != "" && repo != "" {
			// Fetch author info from GitHub API
			handle, avatarURL := fetchGitHubAuthorInfo(owner, repo, sha)
			info.AuthorHandle = handle
			info.AuthorAvatarURL = avatarURL
		}
	}

	return info, nil
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

	req, err := http.NewRequest("GET", url, nil)
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
	branch, err := execGitCommand("git", "rev-parse", "--abbrev-ref", "HEAD")
	if err != nil {
		return ""
	}

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
	_, err := execGitCommand("git", "diff-index", "--quiet", "--cached", "HEAD")
	if err != nil {
		return true
	}

	_, err = execGitCommand("git", "diff-files", "--quiet")
	if err != nil {
		return true
	}

	untrackedOutput, untrackedErr := execGitCommand("git", "ls-files", "--others", "--exclude-standard")
	if untrackedErr != nil {
		return false
	}

	return untrackedOutput != ""
}

// execGitCommand executes a git command and returns trimmed output
func execGitCommand(name string, args ...string) (string, error) {
	cmd := exec.Command(name, args...)
	output, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return "", fmt.Errorf("command failed: %s, stderr: %s", err, string(exitErr.Stderr))
		}
		return "", err
	}
	return strings.TrimSpace(string(output)), nil
}
