// Package store is the Postgres persistence layer: channel-id cache, the
// GitHub->Slack user map, reminder bookkeeping and webhook idempotency.
//
// SQL lives in queries/ and is compiled to typed Go by sqlc (see generate.go);
// the methods here are thin domain wrappers over the generated Queries.
package store

import (
	"context"
	"embed"
	"errors"
	"fmt"
	"sort"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

//go:embed migrations/*.sql
var migrationsFS embed.FS

// Store wraps a pgx connection pool and the sqlc-generated query set.
type Store struct {
	pool *pgxpool.Pool
	q    *Queries
}

// Open connects to Postgres and verifies the connection.
func Open(ctx context.Context, url string) (*Store, error) {
	pool, err := pgxpool.New(ctx, url)
	if err != nil {
		return nil, fmt.Errorf("connect: %w", err)
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("ping: %w", err)
	}
	return &Store{pool: pool, q: New(pool)}, nil
}

// Close releases the connection pool.
func (s *Store) Close() { s.pool.Close() }

// Migrate applies any embedded migrations that have not run yet. Migrations are
// plain SQL files applied in filename order and tracked in schema_migrations.
func (s *Store) Migrate(ctx context.Context) error {
	if _, err := s.pool.Exec(ctx, `CREATE TABLE IF NOT EXISTS schema_migrations (
		version TEXT PRIMARY KEY,
		applied_at TIMESTAMPTZ NOT NULL DEFAULT now()
	)`); err != nil {
		return fmt.Errorf("create schema_migrations: %w", err)
	}

	entries, err := migrationsFS.ReadDir("migrations")
	if err != nil {
		return err
	}
	names := make([]string, 0, len(entries))
	for _, e := range entries {
		names = append(names, e.Name())
	}
	sort.Strings(names)

	for _, name := range names {
		var exists bool
		if err := s.pool.QueryRow(ctx,
			`SELECT EXISTS(SELECT 1 FROM schema_migrations WHERE version = $1)`, name,
		).Scan(&exists); err != nil {
			return err
		}
		if exists {
			continue
		}
		body, err := migrationsFS.ReadFile("migrations/" + name)
		if err != nil {
			return err
		}
		tx, err := s.pool.Begin(ctx)
		if err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, string(body)); err != nil {
			_ = tx.Rollback(ctx)
			return fmt.Errorf("apply %s: %w", name, err)
		}
		if _, err := tx.Exec(ctx, `INSERT INTO schema_migrations (version) VALUES ($1)`, name); err != nil {
			_ = tx.Rollback(ctx)
			return err
		}
		if err := tx.Commit(ctx); err != nil {
			return err
		}
	}
	return nil
}

// SeenDelivery records a webhook delivery id and reports whether it was already
// processed. The first call for an id returns false; repeats return true.
func (s *Store) SeenDelivery(ctx context.Context, id string) (bool, error) {
	if id == "" {
		return false, nil
	}
	affected, err := s.q.InsertDelivery(ctx, id)
	if err != nil {
		return false, err
	}
	return affected == 0, nil
}

// TryAcquireCronLock attempts to take a named lease lock for the given
// duration. It returns true only for the caller that wins the current window;
// other replicas get false. The lease is time-bounded, so a crashed holder is
// released automatically when locked_until passes.
func (s *Store) TryAcquireCronLock(ctx context.Context, name, holder string, lease time.Duration) (bool, error) {
	affected, err := s.q.TryAcquireCronLock(ctx, TryAcquireCronLockParams{
		Name:        name,
		Holder:      holder,
		LockedUntil: time.Now().Add(lease),
	})
	if err != nil {
		return false, err
	}
	return affected == 1, nil
}

// PR is a tracked pull request.
type PR struct {
	Repo        string
	Number      int
	ChannelID   string
	ChannelName string
	State       string
	Author      string
	Reviewers   []string
	URL         string
	Title       string
}

// UpsertPR records or refreshes a tracked PR, preserving created_at.
func (s *Store) UpsertPR(ctx context.Context, pr PR) error {
	return s.q.UpsertPR(ctx, UpsertPRParams{
		Repo:           pr.Repo,
		Number:         int32(pr.Number),
		ChannelID:      pr.ChannelID,
		ChannelName:    pr.ChannelName,
		State:          pr.State,
		AuthorLogin:    pr.Author,
		ReviewerLogins: pr.Reviewers,
		Url:            pr.URL,
		Title:          pr.Title,
	})
}

// AddReviewer appends a reviewer to a tracked PR (deduplicated) and bumps
// activity. Used when a reviewer is requested on a PR that already has a
// channel.
func (s *Store) AddReviewer(ctx context.Context, repo string, number int, login string) error {
	return s.q.AddReviewer(ctx, AddReviewerParams{
		Reviewers: []string{login},
		Repo:      repo,
		Number:    int32(number),
	})
}

// GetPR returns the tracked PR, or (nil, nil) if it is not tracked.
func (s *Store) GetPR(ctx context.Context, repo string, number int) (*PR, error) {
	row, err := s.q.GetPR(ctx, GetPRParams{Repo: repo, Number: int32(number)})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &PR{
		Repo:        row.Repo,
		Number:      int(row.Number),
		ChannelID:   row.ChannelID,
		ChannelName: row.ChannelName,
		State:       row.State,
		Author:      row.AuthorLogin,
		Reviewers:   row.ReviewerLogins,
		URL:         row.Url,
		Title:       row.Title,
	}, nil
}

// Touch bumps last_activity_at so the PR is not flagged as stale.
func (s *Store) Touch(ctx context.Context, repo string, number int) error {
	return s.q.TouchPR(ctx, TouchPRParams{Repo: repo, Number: int32(number)})
}

// SetClosed marks a PR merged or closed and records the archive deadline.
func (s *Store) SetClosed(ctx context.Context, repo string, number int, merged bool) error {
	state := "closed"
	if merged {
		state = "merged"
	}
	return s.q.SetPRClosed(ctx, SetPRClosedParams{State: state, Repo: repo, Number: int32(number)})
}

// MarkArchived records that the Slack channel has been archived.
func (s *Store) MarkArchived(ctx context.Context, repo string, number int) error {
	return s.q.MarkPRArchived(ctx, MarkPRArchivedParams{Repo: repo, Number: int32(number)})
}

// StalePR is the minimal projection the reminder loop needs.
type StalePR struct {
	Repo      string
	Number    int
	ChannelID string
	Author    string
	Reviewers []string
}

// StaleOpenPRs returns open, un-archived PRs whose last activity is older than
// idle and that have not been reminded since then.
func (s *Store) StaleOpenPRs(ctx context.Context, idle time.Duration) ([]StalePR, error) {
	rows, err := s.q.StaleOpenPRs(ctx, time.Now().Add(-idle))
	if err != nil {
		return nil, err
	}
	out := make([]StalePR, 0, len(rows))
	for _, r := range rows {
		out = append(out, StalePR{
			Repo:      r.Repo,
			Number:    int(r.Number),
			ChannelID: r.ChannelID,
			Author:    r.AuthorLogin,
			Reviewers: r.ReviewerLogins,
		})
	}
	return out, nil
}

// MarkReminded records that a reminder was sent for a PR.
func (s *Store) MarkReminded(ctx context.Context, repo string, number int) error {
	return s.q.MarkPRReminded(ctx, MarkPRRemindedParams{Repo: repo, Number: int32(number)})
}

// SlackIDs resolves GitHub logins to Slack user ids, skipping unmapped logins.
func (s *Store) SlackIDs(ctx context.Context, logins []string) (map[string]string, error) {
	out := make(map[string]string, len(logins))
	if len(logins) == 0 {
		return out, nil
	}
	rows, err := s.q.SlackIDs(ctx, logins)
	if err != nil {
		return nil, err
	}
	for _, r := range rows {
		out[r.GithubLogin] = r.SlackUserID
	}
	return out, nil
}
