package status

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/unkeyed/unkey/cmd/api/util"
	"github.com/unkeyed/unkey/internal/sdk"
	"github.com/unkeyed/unkey/pkg/cli"
	"github.com/unkeyed/unkey/pkg/cli/session"
)

func New() *cli.Command {
	return &cli.Command{
		Name:        "status",
		RootAliases: []string{"whoami"},
		Usage:       "Show which credential the CLI uses and whether it works",
		Description: `Show the credential that resource commands would use, chosen in the same order: --root-key, UNKEY_ROOT_KEY, an explicit --config file, the browser session for this API, then a saved root key.

A browser session is checked with the API, and the output shows its workspace and user. The API cannot confirm a root key or report its workspace without a resource permission, so a root key is reported as unchecked. Credentials are never printed.

The command exits with a nonzero status when no credential is found or the check fails. This is not 'unkey keys whoami', which inspects an API key.`,
		Examples: []string{
			"unkey auth status",
			"unkey whoami --json",
		},
		Flags: []cli.Flag{
			util.RootKeyFlag(),
			util.APIURLFlag(),
			util.ConfigFlag(),
			util.OutputFlag(),
			cli.Bool("json", "Print the status as JSON"),
		},
		Action: run,
	}
}

type report struct {
	Credential     string `json:"credential"`
	Authentication string `json:"authentication"`
	APIURL         string `json:"apiUrl"`
	WorkspaceID    string `json:"workspaceId,omitempty"`
	SessionID      string `json:"sessionId,omitempty"`
	UserID         string `json:"userId,omitempty"`
	Detail         string `json:"detail,omitempty"`
}

func run(ctx context.Context, cmd *cli.Command) error {
	cred, err := util.Credential(ctx, cmd)
	status := report{
		Credential:     string(cred.Source),
		Authentication: "",
		APIURL:         cred.APIURL,
		WorkspaceID:    cred.WorkspaceID,
		SessionID:      cred.SessionID,
		UserID:         "",
		Detail:         "",
	}
	if status.APIURL == "" {
		status.APIURL = session.NormalizeAPIURL(cmd.String("api-url"))
	}

	switch {
	case errors.Is(err, util.ErrNoCredentials):
		status.Credential = "none"
		status.Authentication = "missing"
		return errors.Join(write(cmd, status), err)
	case err != nil && cred.Source == util.SourceSession:
		err = fmt.Errorf("could not refresh the browser session: %w", err)
	case err != nil:
		return err
	case cred.Source != util.SourceSession:
		status.Authentication = "unchecked"
		status.Detail = "The API cannot confirm a root key or report its workspace without a resource permission."
		return write(cmd, status)
	default:
		status.UserID, err = verifySession(ctx, cred)
	}

	if err != nil {
		status.Authentication = "failed"
		status.Detail = err.Error()
		var response *sdk.ResponseError
		if errors.As(err, &response) && response.StatusCode == http.StatusUnauthorized {
			err = errors.New("the browser session no longer works. Run 'unkey login' to sign in again")
		}
		return errors.Join(write(cmd, status), err)
	}
	status.Authentication = "verified"
	return write(cmd, status)
}

// verifySession lists CLI sessions because it is the only endpoint a CLI
// session can call without a resource permission. It returns the user that
// owns the session when the list includes it.
func verifySession(ctx context.Context, cred util.ResolvedCredential) (string, error) {
	res, err := sdk.NewClient(cred.APIURL, nil).Cli.ListSessions(ctx, sdk.Bearer(cred.Key))
	var response *sdk.ResponseError
	if errors.As(err, &response) {
		err = response
	}
	if err != nil {
		return "", fmt.Errorf("could not check the browser session: %w", err)
	}
	if res.V2CliListSessionsResponseBody == nil {
		return "", fmt.Errorf("the API returned HTTP %d", res.HTTPMeta.Response.StatusCode)
	}
	for _, item := range res.V2CliListSessionsResponseBody.Data.Sessions {
		if item.ID == cred.SessionID {
			return item.UserID, nil
		}
	}
	return "", nil
}

func write(cmd *cli.Command, status report) error {
	return util.Output(cmd, struct {
		Data report `json:"data"`
	}{Data: status})
}
