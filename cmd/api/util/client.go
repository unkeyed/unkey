package util

import (
	"context"
	"errors"
	"fmt"
	"time"

	unkey "github.com/unkeyed/sdks/api/go/v2"
	"github.com/unkeyed/unkey/pkg/cli"
)

// noCredentialsError is the guidance returned when no usable credential exists.
func noCredentialsError() error {
	return fmt.Errorf("no credentials found\n\nAuthenticate via one of:\n  unkey auth login            (WorkOS OAuth)\n  unkey auth login --root-key (store a root key)\n  --root-key flag / UNKEY_ROOT_KEY environment variable")
}

// CreateClient builds an SDK client using, in priority order:
//  1. an explicit --root-key flag or UNKEY_ROOT_KEY env var
//  2. a valid stored OAuth access token (refreshed in place if expired)
//  3. a stored root key
//
// The access token and root key are both sent as Authorization: Bearer by the
// SDK's WithSecurity option, so only the credential choice differs.
func CreateClient(cmd *cli.Command) (*unkey.Unkey, error) {
	token, err := resolveToken(cmd)
	if err != nil {
		return nil, err
	}

	opts := []unkey.SDKOption{unkey.WithSecurity(token)}
	if url := cmd.String("api-url"); url != "" {
		opts = append(opts, unkey.WithServerURL(url))
	}
	return unkey.New(opts...), nil
}

func resolveToken(cmd *cli.Command) (string, error) {
	// 1. Explicit root key always wins.
	if key := cmd.String("root-key"); key != "" {
		return key, nil
	}

	cfg, err := cli.LoadUserConfig(cmd.String("config"))
	if err != nil {
		return "", noCredentialsError()
	}

	// 2. A valid OAuth access token.
	if cfg.AccessTokenValid(time.Now()) {
		return string(cfg.AccessToken), nil
	}

	// 3. Expired access token but a refresh token is available.
	if cfg.RefreshToken != "" {
		if token, rerr := refreshAndStore(cmd, cfg); rerr == nil {
			return token, nil
		}
		// Refresh failed; fall through to a stored root key if present.
	}

	// 4. A stored root key.
	if cfg.RootKey != "" {
		return cfg.RootKey, nil
	}

	// 5. Had an OAuth session but could not refresh and no root key to fall back
	//    on: the session is unusable.
	if cfg.HasOAuth() {
		return "", errors.New("your session has expired; run `unkey auth login`")
	}

	return "", noCredentialsError()
}

// refreshAndStore exchanges the stored refresh token for a fresh access token,
// persists the rotated tokens (newest refresh token wins), and returns the new
// access token. It uses context.Background bounded by the OAuth client's own
// timeout to avoid threading ctx through every api command's CreateClient call.
func refreshAndStore(cmd *cli.Command, cfg cli.UserConfig) (string, error) {
	clientID := ResolveWorkOSClientID(cmd)
	if clientID == "" {
		return "", errors.New("WorkOS client ID is not configured")
	}

	tok, err := NewOAuthClient().Refresh(context.Background(), clientID, string(cfg.RefreshToken), cfg.OrgID)
	if err != nil {
		return "", err
	}

	cfg.AccessToken = cli.Secret(tok.AccessToken)
	cfg.RefreshToken = cli.Secret(tok.RefreshToken)
	cfg.ExpiresAt = tok.ExpiresAt.UTC().Format(time.RFC3339)
	if tok.OrganizationID != "" {
		cfg.OrgID = tok.OrganizationID
	}
	if tok.Email != "" {
		cfg.UserEmail = tok.Email
	}
	// Persist before use so a rotated refresh token is never lost. A save failure
	// is non-fatal: the new access token is still usable for this invocation.
	_ = cli.SaveUserConfig(cfg)

	return tok.AccessToken, nil
}
