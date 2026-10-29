package logout

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/unkeyed/unkey/cmd/api/util"
	"github.com/unkeyed/unkey/internal/sdk"
	"github.com/unkeyed/unkey/internal/sdk/models/components"
	"github.com/unkeyed/unkey/pkg/cli"
	"github.com/unkeyed/unkey/pkg/cli/session"
)

func New() *cli.Command {
	return &cli.Command{
		Name:        "logout",
		RootAliases: []string{"logout"},
		Usage:       "Sign out of the active workspace",
		Description: "Revokes the active browser session on the server and removes it from this machine. Root keys are not removed.",
		Flags:       []cli.Flag{util.APIURLFlag()},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			current, err := session.Active()
			if errors.Is(err, session.ErrNotFound) {
				fmt.Println("No saved browser session.")
				return nil
			}
			if err != nil {
				return err
			}
			fallbackURL := cmd.String("api-url")
			used, revokeErr := revoke(ctx, fallbackURL)
			if used.WorkspaceID != "" {
				current = used
			}
			if _, err := session.Delete(current.WorkspaceID, current.SessionID); err != nil && !errors.Is(err, session.ErrNotFound) {
				return err
			}
			if revokeErr != nil {
				fmt.Fprintln(os.Stderr, "The session was removed from this machine but could not be revoked on the server. It stays valid until it expires.")
			} else {
				fmt.Println("Signed out.")
			}
			if next, err := session.Active(); err == nil {
				fmt.Fprintf(os.Stderr, "Switched to workspace %s.\n", next.WorkspaceID)
			}
			if revokeErr != nil {
				return fmt.Errorf("could not sign out: %w", revokeErr)
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
	api := sdk.NewClient(apiURL(current, fallbackURL), nil)
	res, err := api.Cli.RevokeSession(ctx, components.V2CliRevokeSessionRequestBody{SessionID: current.SessionID}, sdk.Bearer(current.AccessToken))
	if err != nil {
		var response *sdk.ResponseError
		if errors.As(err, &response) {
			return current, fmt.Errorf("server returned HTTP %d: %w", response.StatusCode, err)
		}
		return current, err
	}
	if res.V2CliRevokeSessionResponseBody == nil {
		return current, fmt.Errorf("server returned HTTP %d", res.HTTPMeta.Response.StatusCode)
	}
	return current, nil
}

func apiURL(current session.Session, fallbackURL string) string {
	if url := session.NormalizeAPIURL(current.APIURL); url != "" {
		return url
	}
	return session.NormalizeAPIURL(fallbackURL)
}
