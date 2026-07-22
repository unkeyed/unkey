package auth

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/cli/browser"
	"github.com/unkeyed/unkey/cmd/api/util"
	"github.com/unkeyed/unkey/cmd/internal/oauth"
	"github.com/unkeyed/unkey/pkg/cli"
	"golang.org/x/term"
)

// Cmd is the auth command for managing CLI authentication.
var Cmd = &cli.Command{
	Name:        "auth",
	Usage:       "Manage authentication",
	Description: "Authenticate with the Unkey API using WorkOS OAuth (device flow) or a root key.",
	Flags:       []cli.Flag{},
	Commands: []*cli.Command{
		loginCmd,
		logoutCmd,
		whoamiCmd,
	},
}

var loginCmd = &cli.Command{
	Name:  "login",
	Usage: "Authenticate with Unkey",
	Description: `Sign in to Unkey using WorkOS OAuth via the device authorization flow.

The CLI prints a code and a URL. Open the URL in a browser, confirm the code,
and select your organization when prompted. On success your access and refresh
tokens are stored in ~/.unkey/config.toml.

Use --root-key to store a root key instead (legacy authentication).`,
	Flags: []cli.Flag{
		util.WorkOSClientIDFlag(),
		cli.Bool("root-key", "Authenticate by storing a root key instead of OAuth"),
	},
	Action: func(ctx context.Context, cmd *cli.Command) error {
		if cmd.Bool("root-key") {
			return storeRootKey()
		}

		clientID := util.ResolveWorkOSClientID(cmd)
		if clientID == "" {
			return errors.New("WorkOS client ID is not configured; set --workos-client-id or UNKEY_WORKOS_CLIENT_ID")
		}
		return deviceLogin(ctx, util.NewOAuthClient(), clientID, os.Stdout, func(url string) { _ = browser.OpenURL(url) })
	},
}

var logoutCmd = &cli.Command{
	Name:        "logout",
	Usage:       "Sign out and clear stored credentials",
	Description: "Revoke the WorkOS session (best effort) and remove stored credentials from ~/.unkey/config.toml.",
	Flags: []cli.Flag{
		util.WorkOSClientIDFlag(),
		cli.Bool("keep-root-key", "Clear the OAuth session but keep a stored root key"),
	},
	Action: func(ctx context.Context, cmd *cli.Command) error {
		return logout(ctx, util.NewOAuthClient(), util.ResolveWorkOSClientID(cmd), cmd.Bool("keep-root-key"), os.Stdout)
	},
}

var whoamiCmd = &cli.Command{
	Name:        "whoami",
	Usage:       "Show the current CLI identity",
	Description: "Report who you are authenticated as, your organization, and token validity.",
	Flags: []cli.Flag{
		util.OutputFlag(),
	},
	Action: func(ctx context.Context, cmd *cli.Command) error {
		return whoami(os.Stdout, cmd.String("output") == "json", time.Now())
	},
}

// storeRootKey preserves the legacy behavior: prompt for a root key and persist
// it, keeping any existing OAuth session untouched.
func storeRootKey() error {
	fmt.Print("Enter your root key: ")
	raw, err := term.ReadPassword(int(os.Stdin.Fd()))
	fmt.Println()
	if err != nil {
		return fmt.Errorf("failed to read input: %w", err)
	}
	key := strings.TrimSpace(string(raw))
	if key == "" {
		return errors.New("root key cannot be empty")
	}

	cfg := loadConfigOrEmpty()
	cfg.RootKey = key
	if err := cli.SaveUserConfig(cfg); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	path, _ := cli.UserConfigPath()
	fmt.Printf("Authentication successful. Key stored in %s\n", path)
	return nil
}

// deviceLogin runs the WorkOS device authorization flow and stores the resulting
// tokens, preserving any existing root key.
func deviceLogin(ctx context.Context, client *oauth.Client, clientID string, out io.Writer, openURL func(string)) error {
	da, err := client.RequestDeviceAuthorization(ctx, clientID)
	if err != nil {
		return fmt.Errorf("could not start device authorization: %w", err)
	}

	fmt.Fprintf(out, "\nTo sign in, visit:\n\n  %s\n\n", da.VerificationURIComplete)
	fmt.Fprintf(out, "and confirm this code: %s\n\n", da.UserCode)
	fmt.Fprintln(out, "Only continue if you started this login yourself; never enter a code someone sent you.")
	if openURL != nil {
		openURL(da.VerificationURIComplete)
	}
	fmt.Fprintln(out, "\nWaiting for confirmation...")

	tok, err := client.PollForToken(ctx, clientID, da.DeviceCode, da.Interval, da.ExpiresIn)
	if err != nil {
		return fmt.Errorf("login failed: %w", err)
	}

	// An empty organization_id yields a token the API rejects with 401 on every
	// call, so fail here with actionable guidance instead of persisting it.
	if tok.OrganizationID == "" {
		return errors.New("login did not resolve an organization; select an organization during sign-in or ensure you have an active organization membership")
	}

	cfg := loadConfigOrEmpty()
	cfg.AccessToken = cli.Secret(tok.AccessToken)
	cfg.RefreshToken = cli.Secret(tok.RefreshToken)
	cfg.ExpiresAt = tok.ExpiresAt.UTC().Format(time.RFC3339)
	cfg.OrgID = tok.OrganizationID
	cfg.UserEmail = tok.Email
	if err := cli.SaveUserConfig(cfg); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	who := tok.Email
	if who == "" {
		who = "you"
	}
	fmt.Fprintf(out, "\nLogged in as %s (org: %s)\n", who, tok.OrganizationID)
	return nil
}

// logout revokes the WorkOS session (best effort) and clears stored credentials.
func logout(ctx context.Context, client *oauth.Client, clientID string, keepRootKey bool, out io.Writer) error {
	cfg := loadConfigOrEmpty()

	if !cfg.HasOAuth() && cfg.RootKey == "" {
		fmt.Fprintln(out, "Not logged in.")
		return nil
	}

	revocationConfirmed := true
	if cfg.RefreshToken != "" {
		if err := client.Revoke(ctx, clientID, string(cfg.RefreshToken)); err != nil {
			// Non-fatal: local logout must always complete.
			revocationConfirmed = false
		}
	}

	cleared := cfg
	cleared.AccessToken = ""
	cleared.RefreshToken = ""
	cleared.ExpiresAt = ""
	cleared.OrgID = ""
	cleared.UserEmail = ""
	if !keepRootKey {
		cleared.RootKey = ""
	}
	if err := cli.SaveUserConfig(cleared); err != nil {
		return fmt.Errorf("failed to update config: %w", err)
	}

	fmt.Fprintln(out, "Logged out.")
	if cfg.HasOAuth() && !revocationConfirmed {
		fmt.Fprintln(out, "Note: could not confirm server-side session revocation; the access token expires on its own.")
	}
	return nil
}

// whoami reports the current identity without ever emitting raw token values.
func whoami(out io.Writer, jsonOutput bool, now time.Time) error {
	cfg := loadConfigOrEmpty()

	info := identity{} //nolint:exhaustruct // populated per auth method below
	switch {
	case cfg.HasOAuth():
		info.Method = "oauth"
		info.Email = cfg.UserEmail
		info.OrgID = cfg.OrgID
		info.ExpiresAt = cfg.ExpiresAt
		info.Valid = cfg.AccessTokenValid(now)
	case cfg.RootKey != "":
		info.Method = "root_key"
	default:
		info.Method = "none"
	}

	if jsonOutput {
		return writeIdentityJSON(out, info)
	}

	switch info.Method {
	case "oauth":
		status := "valid"
		if !info.Valid {
			status = "expired"
		}
		fmt.Fprintf(out, "Authenticated as %s\n", nonEmpty(info.Email, "(unknown user)"))
		fmt.Fprintf(out, "Organization: %s\n", info.OrgID)
		fmt.Fprintf(out, "Token: %s (expires %s)\n", status, info.ExpiresAt)
	case "root_key":
		fmt.Fprintln(out, "Authenticated with a root key.")
	default:
		fmt.Fprintln(out, "Not logged in. Run `unkey auth login`.")
	}
	return nil
}

// identity is the token-free view of the stored credential surfaced by whoami.
// It deliberately excludes access_token and refresh_token.
type identity struct {
	Method    string `json:"method"`
	Email     string `json:"email,omitempty"`
	OrgID     string `json:"orgId,omitempty"`
	ExpiresAt string `json:"expiresAt,omitempty"`
	Valid     bool   `json:"valid,omitempty"`
}

func writeIdentityJSON(out io.Writer, info identity) error {
	enc := json.NewEncoder(out)
	enc.SetIndent("", "  ")
	return enc.Encode(info)
}

func loadConfigOrEmpty() cli.UserConfig {
	path, err := cli.UserConfigPath()
	if err != nil {
		return cli.UserConfig{} //nolint:exhaustruct // empty config when none is stored/loadable
	}
	if _, statErr := os.Stat(path); statErr != nil {
		return cli.UserConfig{} //nolint:exhaustruct // empty config when none is stored/loadable
	}
	cfg, err := cli.LoadUserConfig(path)
	if err != nil {
		return cli.UserConfig{} //nolint:exhaustruct // empty config when none is stored/loadable
	}
	return cfg
}

func nonEmpty(v, fallback string) string {
	if v == "" {
		return fallback
	}
	return v
}
