// Command pr-channels runs the GitHub -> Slack pull-request channel bot: a
// single long-running server. It receives GitHub webhooks and, on an internal
// timer, nudges stale PRs. The reminder loop is guarded by a Postgres lease
// lock so only one replica fires it per interval.
package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/unkeyed/unkey/tools/pr-channels/internal/app"
	"github.com/unkeyed/unkey/tools/pr-channels/internal/config"
	"github.com/unkeyed/unkey/tools/pr-channels/internal/server"
	"github.com/unkeyed/unkey/tools/pr-channels/internal/slackclient"
	"github.com/unkeyed/unkey/tools/pr-channels/internal/store"
)

func main() {
	log := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	if err := run(log); err != nil {
		log.Error("fatal", "err", err)
		os.Exit(1)
	}
}

func run(log *slog.Logger) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	// Cancel everything on SIGINT/SIGTERM for a clean shutdown.
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	st, err := store.Open(ctx, cfg.DatabaseURL)
	if err != nil {
		return err
	}
	defer st.Close()
	if err := st.Migrate(ctx); err != nil {
		return err
	}

	a := app.New(cfg, st, slackclient.New(cfg.SlackBotToken), log)

	// Reminder loop: one replica wins each interval via the lease lock.
	go reminderLoop(ctx, cfg, st, a, log)

	srv := &http.Server{
		Addr:              cfg.Addr,
		Handler:           server.New(a, st, cfg.GitHubWebhookSecret, log).Handler(),
		ReadHeaderTimeout: 10 * time.Second,
	}

	// Shut the HTTP server down when the context is cancelled.
	go func() {
		<-ctx.Done()
		sctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		_ = srv.Shutdown(sctx)
	}()

	log.Info("listening", "addr", cfg.Addr)
	if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	return nil
}

func reminderLoop(ctx context.Context, cfg config.Config, st *store.Store, a *app.App, log *slog.Logger) {
	holder := randomHolder()
	t := time.NewTicker(cfg.ReminderInterval)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			ok, err := st.TryAcquireCronLock(ctx, "reminders", holder, cfg.ReminderInterval)
			if err != nil {
				log.Error("reminder lock failed", "err", err)
				continue
			}
			if !ok {
				continue // another replica owns this window
			}
			if err := a.Remind(ctx); err != nil {
				log.Error("reminders failed", "err", err)
			}
		}
	}
}

func randomHolder() string {
	b := make([]byte, 8)
	if _, err := rand.Read(b); err != nil {
		return "unknown"
	}
	return hex.EncodeToString(b)
}
