package logout

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
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
			apiURL := session.NormalizeAPIURL(current.APIURL)
			if apiURL == "" {
				apiURL = cmd.String("api-url")
			}
			if err := revoke(ctx, apiURL, current); err != nil {
				fmt.Printf("Could not revoke the session on the server (%s). It was removed from this machine.\n", err.Error())
			}
			if _, err := session.Delete(current.WorkspaceID); err != nil && !errors.Is(err, session.ErrNotFound) {
				return err
			}
			fmt.Printf("Signed out of workspace %s.\n", current.WorkspaceID)
			return nil
		},
	}
}

func revoke(ctx context.Context, apiURL string, current session.Session) error {
	token := current.AccessToken
	if current.ExpiresAt <= time.Now().Add(time.Minute).Unix() {
		refreshed, err := session.Refresh(ctx, apiURL, current.RefreshToken)
		if err != nil {
			return err
		}
		if refreshed.WorkspaceID == "" {
			refreshed.WorkspaceID = current.WorkspaceID
		}
		if refreshed.SessionID == "" {
			refreshed.SessionID = current.SessionID
		}
		if refreshed.APIURL == "" {
			refreshed.APIURL = current.APIURL
		}
		if err := session.Save(refreshed); err != nil {
			return err
		}
		token = refreshed.AccessToken
	}
	body, _ := json.Marshal(map[string]string{"sessionId": current.SessionID})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, strings.TrimRight(apiURL, "/")+"/v2/cli.revokeSession", bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("content-type", "application/json")
	req.Header.Set("authorization", "Bearer "+token)
	res, err := (&http.Client{Timeout: 30 * time.Second}).Do(req)
	if err != nil {
		return err
	}
	defer func() { _ = res.Body.Close() }()
	payload, _ := io.ReadAll(io.LimitReader(res.Body, 1<<20))
	if res.StatusCode >= 300 {
		return fmt.Errorf("revoke returned %d: %s", res.StatusCode, strings.TrimSpace(string(payload)))
	}
	return nil
}
