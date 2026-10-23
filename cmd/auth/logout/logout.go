package logout

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/unkeyed/unkey/cmd/api/util"
	"github.com/unkeyed/unkey/pkg/cli"
	"github.com/unkeyed/unkey/pkg/cli/session"
)

func New() *cli.Command {
	return &cli.Command{
		Name:        "logout",
		Usage:       "Revoke the active CLI session",
		Description: "Revokes the active CLI session on the server and deletes it from this machine.",
		Flags:       []cli.Flag{util.APIURLFlag()},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			current, err := session.Active()
			if errors.Is(err, session.ErrNotFound) {
				fmt.Println("No CLI session is stored.")
				return nil
			}
			if err != nil {
				return err
			}
			fallbackURL := cmd.String("api-url")
			used, revokeErr := revoke(ctx, fallbackURL)
			workspaceID := current.WorkspaceID
			if used.WorkspaceID != "" {
				workspaceID = used.WorkspaceID
			}
			if _, err := session.Delete(workspaceID); err != nil && !errors.Is(err, session.ErrNotFound) {
				return err
			}
			fmt.Printf("Signed out of workspace %s.\n", workspaceID)
			if next, err := session.Active(); err == nil {
				fmt.Printf("Active workspace is %s.\n", next.WorkspaceID)
			}
			if revokeErr != nil {
				fmt.Fprintln(os.Stderr, "The session was removed from this machine but could not be revoked on the server. It stays valid until it expires.")
				return fmt.Errorf("revoke CLI session: %w", revokeErr)
			}
			return nil
		},
	}
}

func revoke(ctx context.Context, fallbackURL string) (session.Session, error) {
	current, err := session.AccessToken(ctx, func(ctx context.Context, current session.Session) (session.Session, error) {
		return session.Refresh(ctx, apiURL(current, fallbackURL), current.RefreshToken)
	})
	if err != nil {
		return session.Session{}, err
	}
	body, err := json.Marshal(map[string]string{"sessionId": current.SessionID})
	if err != nil {
		return current, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, apiURL(current, fallbackURL)+"/v2/cli.revokeSession", bytes.NewReader(body))
	if err != nil {
		return current, err
	}
	req.Header.Set("content-type", "application/json")
	req.Header.Set("authorization", "Bearer "+current.AccessToken)
	res, err := (&http.Client{Timeout: 30 * time.Second}).Do(req)
	if err != nil {
		return current, err
	}
	defer func() { _ = res.Body.Close() }()
	payload, _ := io.ReadAll(io.LimitReader(res.Body, 1<<20))
	if res.StatusCode >= 300 {
		return current, fmt.Errorf("revoke returned %d: %s", res.StatusCode, strings.TrimSpace(string(payload)))
	}
	return current, nil
}

func apiURL(current session.Session, fallbackURL string) string {
	if url := session.NormalizeAPIURL(current.APIURL); url != "" {
		return url
	}
	return session.NormalizeAPIURL(fallbackURL)
}
