# gha-fetch-digest

This utility is to help secure GitHub Actions by fetching a commit hash that correlates to a tag. The default output is a string you can drop into your GitHub Actions file, but supports a bare sha, tag, and json.

## Usage

### Quickstart

```
$ gha-fetch-digest checkout
- uses: actions/checkout@08c6903 # v5.0.0
```

### Help

```
Fetch GitHub Action tag and the corresponding commit SHAs for use in GitHub Actions workflows.

By default, fetches the latest release. You can specify a particular tag with --tag.
If no owner is specified, defaults to 'actions' organization.

Usage:
  gha-fetch-digest [owner/repo or repo] [flags]

Examples:
  # Get latest release of actions/checkout
  gha-fetch-digest checkout

  # Get specific tag
  gha-fetch-digest --tag v4 actions/checkout

  # Output just the SHA
  gha-fetch-digest --format sha hashicorp/setup-terraform

  # Output as JSON
  gha-fetch-digest --format json actions/setup-go

Flags:
  -f, --format string       Output format: uses, sha, tag, or json (default "uses")
  -h, --help                help for gha-fetch-digest
  -t, --tag string          Specific tag to fetch (if empty, uses GitHub's latest release)
      --token-file string   Path to file containing GitHub token
```
