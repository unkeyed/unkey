package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/google/go-github/v63/github"
	"github.com/spf13/cobra"
)

var (
	tagName   string
	tokenFile string
	format    string
	version   string = "dev"
)

var rootCmd = &cobra.Command{
	Use:   "gha-fetch-digest [owner/repo or repo]",
	Short: "Fetch GitHub Action tag and the commit SHAs",
	Long: `Fetch GitHub Action tag and the corresponding commit SHAs for use in GitHub Actions workflows.

By default, fetches the latest release. You can specify a particular tag with --tag.
If no owner is specified, defaults to 'actions' organization.`,
	Example: `  # Get latest release of actions/checkout
  gha-fetch-digest checkout

  # Get specific tag
  gha-fetch-digest --tag v4 actions/checkout

  # Output just the SHA
  gha-fetch-digest --format sha hashicorp/setup-terraform

  # Output as JSON
  gha-fetch-digest --format json actions/setup-go`,
	Args: cobra.ExactArgs(1),
	RunE: runCommand,
}

func init() {
	rootCmd.PersistentFlags().StringVarP(&tagName, "tag", "t", "", "Specific tag to fetch (if empty, uses GitHub's latest release)")
	rootCmd.PersistentFlags().StringVar(&tokenFile, "token-file", "", "Path to file containing GitHub token")
	rootCmd.PersistentFlags().StringVarP(&format, "format", "f", "uses", "Output format: uses, sha, tag, or json")
	rootCmd.AddCommand(versionCmd)
}

func runCommand(cmd *cobra.Command, args []string) error {
	repoArg := args[0]
	parts := strings.Split(repoArg, "/")

	var owner, repo string
	switch len(parts) {
	case 1:
		owner = "actions"
		repo = parts[0]
	case 2:
		owner = parts[0]
		repo = parts[1]
	default:
		return fmt.Errorf("invalid repository format: %s (use 'owner/repo' or 'repo')", repoArg)
	}

	token := loadToken()

	ctx := context.Background()
	client := github.NewClient(nil)
	if token != "" {
		client = client.WithAuthToken(token)
	}

	selectedTag, err := fetchTag(ctx, client, owner, repo)
	if err != nil {
		return err
	}

	sha, err := fetchCommitSHA(ctx, client, owner, repo, selectedTag)
	if err != nil {
		return err
	}

	return outputResult(owner, repo, selectedTag, sha)
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the version number of gha-fetch-digest",
	Long:  `Print the version number of gha-fetch digest longer`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println(version)
	},
}

func loadToken() string {
	if tokenFile != "" {
		data, err := os.ReadFile(tokenFile)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: could not read token file: %v\n", err)
			return ""
		}
		return strings.TrimSpace(string(data))
	}

	return os.Getenv("GITHUB_TOKEN")
}

func fetchTag(ctx context.Context, client *github.Client, owner, repo string) (string, error) {
	if tagName != "" {
		return tagName, nil
	}

	release, _, err := client.Repositories.GetLatestRelease(ctx, owner, repo)
	if err != nil {
		var errResp *github.ErrorResponse
		if errors.As(err, &errResp) {
			if errResp.Response != nil && errResp.Response.StatusCode == 404 {
				return "", fmt.Errorf("no latest release found. Check available releases at: https://github.com/%s/%s/releases", owner, repo)
			}
			if errResp.Response != nil && errResp.Response.StatusCode == 401 {
				return "", fmt.Errorf("authentication failed. Check your GitHub token: %w", err)
			}
		}

		var rateLimitErr *github.RateLimitError
		if errors.As(err, &rateLimitErr) {
			return "", fmt.Errorf("GitHub API rate limit exceeded. Resets at %v. Set GITHUB_TOKEN environment variable",
				rateLimitErr.Rate.Reset.Time)
		}

		return "", fmt.Errorf("error fetching latest release: %w", err)
	}

	if release.TagName == nil {
		return "", fmt.Errorf("latest release has no tag name")
	}

	return *release.TagName, nil
}

func fetchCommitSHA(ctx context.Context, client *github.Client, owner, repo, tag string) (string, error) {
	commit, _, err := client.Repositories.GetCommit(ctx, owner, repo, tag, nil)
	if err != nil {
		var errResp *github.ErrorResponse
		if errors.As(err, &errResp) && errResp.Response != nil && errResp.Response.StatusCode == 404 {
			return "", fmt.Errorf("tag '%s' not found in repository %s/%s", tag, owner, repo)
		}
		return "", fmt.Errorf("error fetching commit for tag %s: %w", tag, err)
	}

	if commit.SHA == nil {
		return "", fmt.Errorf("commit has no SHA")
	}

	return *commit.SHA, nil
}

func outputResult(owner, repo, tag, sha string) error {
	switch format {
	case "uses":
		fmt.Printf("- uses: %s/%s@%s # %s\n", owner, repo, sha, tag)
	case "sha":
		fmt.Println(sha)
	case "tag":
		fmt.Println(tag)
	case "json":
		output := map[string]string{
			"owner": owner,
			"repo":  repo,
			"tag":   tag,
			"sha":   sha,
		}
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(output)
	default:
		return fmt.Errorf("unknown format: %s (use: uses, sha, tag, or json)", format)
	}
	return nil
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
