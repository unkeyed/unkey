// Package config loads the service configuration from the environment.
package config

import (
	"fmt"
	"os"
	"strings"
	"time"
)

// Config is the fully resolved runtime configuration.
type Config struct {
	// Addr is the host:port the HTTP server listens on. Unkey Deploy sets PORT.
	Addr string

	// DatabaseURL is the PlanetScale Postgres connection string (pgx format).
	DatabaseURL string

	// GitHubWebhookSecret verifies incoming webhook signatures.
	GitHubWebhookSecret string

	// SlackBotToken is the Slack bot user OAuth token (xoxb-...).
	SlackBotToken string

	// OverviewChannelID is the shared overview feed channel. Opened and merged
	// PRs are posted here. Empty disables the feed. The bot must be a member of
	// this channel.
	OverviewChannelID string

	// RepoAllowlist limits which repos get channels. Empty or {"*"} means the
	// whole org. Entries match the repository name (without owner).
	RepoAllowlist []string

	// ArchiveDelay is how long to wait after a PR closes before archiving its
	// channel, so people can still read the outcome.
	ArchiveDelay time.Duration

	// ReminderIdle is how long a PR can sit without activity before its
	// reviewers get nudged.
	ReminderIdle time.Duration

	// ReminderInterval is how often the reminder loop checks for stale PRs. One
	// replica wins each interval via a lease lock; the rest skip.
	ReminderInterval time.Duration
}

// Load reads configuration from the environment, applying defaults and
// returning an error if a required value is missing.
func Load() (Config, error) {
	c := Config{
		Addr:                ":" + envOr("PORT", "8080"),
		DatabaseURL:         os.Getenv("DATABASE_URL"),
		GitHubWebhookSecret: os.Getenv("GITHUB_WEBHOOK_SECRET"),
		SlackBotToken:       os.Getenv("SLACK_BOT_TOKEN"),
		OverviewChannelID:   os.Getenv("OVERVIEW_CHANNEL_ID"),
		RepoAllowlist:       splitList(os.Getenv("REPO_ALLOWLIST")),
		ArchiveDelay:        durationOr("ARCHIVE_DELAY", time.Minute),
		ReminderIdle:        durationOr("REMINDER_IDLE", 18*time.Hour),
		ReminderInterval:    durationOr("REMINDER_INTERVAL", 15*time.Minute),
	}

	var missing []string
	if c.DatabaseURL == "" {
		missing = append(missing, "DATABASE_URL")
	}
	if c.GitHubWebhookSecret == "" {
		missing = append(missing, "GITHUB_WEBHOOK_SECRET")
	}
	if c.SlackBotToken == "" {
		missing = append(missing, "SLACK_BOT_TOKEN")
	}
	if len(missing) > 0 {
		return Config{}, fmt.Errorf("missing required env: %s", strings.Join(missing, ", "))
	}
	return c, nil
}

// ManagesAll reports whether every repo in the org should be managed.
func (c Config) ManagesAll() bool {
	if len(c.RepoAllowlist) == 0 {
		return true
	}
	for _, r := range c.RepoAllowlist {
		if r == "*" {
			return true
		}
	}
	return false
}

// ManagesRepo reports whether the given repository name is in scope.
func (c Config) ManagesRepo(name string) bool {
	if c.ManagesAll() {
		return true
	}
	for _, r := range c.RepoAllowlist {
		if strings.EqualFold(r, name) {
			return true
		}
	}
	return false
}

func envOr(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func durationOr(key string, def time.Duration) time.Duration {
	if v := os.Getenv(key); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			return d
		}
	}
	return def
}

func splitList(s string) []string {
	if strings.TrimSpace(s) == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
}
