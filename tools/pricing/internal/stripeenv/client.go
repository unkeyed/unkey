// Package stripeenv builds a configured Stripe client for a target environment
// and sources the API key (from the environment or AWS Secrets Manager).
package stripeenv

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/stripe/stripe-go/v86"

	"github.com/unkeyed/unkey/tools/pricing"
)

const (
	// defaultSecretID is the AWS Secrets Manager secret holding the Stripe key
	// (`{"api_key":"..."}`). Override with STRIPE_SECRET_ID.
	defaultSecretID = "unkey/stripe"
	// defaultRegion is where the secret lives. Override with AWS_REGION.
	defaultRegion = "us-east-1"
	// defaultSSOSession is the SSO session `aws sso login` authenticates against.
	// The AWS config defines `[sso-session unkey]` with per-account profiles
	// (unkey-<account>-<role>) riding on it. There is no bare `unkey` profile, so
	// login targets the session via `--sso-session`, not `--profile`. The profiles
	// are used for the API calls (see awsProfile). Override with AWS_SSO_SESSION.
	defaultSSOSession = "unkey"
)

// awsProfileEnvVar is the per-environment profile variable, e.g.
// AWS_PROFILE_PRODUCTION.
func awsProfileEnvVar(env pricing.Environment) string {
	return "AWS_PROFILE_" + strings.ToUpper(string(env))
}

// awsProfile resolves the per-account AWS profile for env, used for the Secrets
// Manager call. Names like unkey-<account>-<role> are deployment-specific, so
// none is baked into the source. Resolution order:
//
//   - AWS_PROFILE (the standard AWS variable), else
//   - AWS_PROFILE_<ENV> (e.g. AWS_PROFILE_PRODUCTION), normally set in a local,
//     untracked .env that the `mise run pricing` task loads (see .env.example).
func awsProfile(env pricing.Environment) (string, error) {
	if p := strings.TrimSpace(os.Getenv("AWS_PROFILE")); p != "" {
		return p, nil
	}
	if p := strings.TrimSpace(os.Getenv(awsProfileEnvVar(env))); p != "" {
		return p, nil
	}
	return "", fmt.Errorf("no Stripe key found: export STRIPE_SECRET_KEY (simplest), or set AWS_PROFILE / %s to read it from Secrets Manager (see tools/pricing/.env.example)",
		awsProfileEnvVar(env))
}

// ssoSession returns the SSO session for `aws sso login` (see defaultSSOSession).
func ssoSession() string {
	if s := strings.TrimSpace(os.Getenv("AWS_SSO_SESSION")); s != "" {
		return s
	}
	return defaultSSOSession
}

// secretID honors a STRIPE_SECRET_ID override, else the default.
func secretID() string {
	if s := strings.TrimSpace(os.Getenv("STRIPE_SECRET_ID")); s != "" {
		return s
	}
	return defaultSecretID
}

// region honors an AWS_REGION override, else the default.
func region() string {
	if r := strings.TrimSpace(os.Getenv("AWS_REGION")); r != "" {
		return r
	}
	return defaultRegion
}

// Client is a configured Stripe client bound to the environment it targets.
type Client struct {
	*stripe.Client
	Env pricing.Environment
}

// New builds a Stripe client for env. The key is sourced as:
//
//   - STRIPE_SECRET_KEY from the environment, if set: the simple path for
//     sandbox/dev where you supply your own key; otherwise
//   - AWS Secrets Manager (api_key field) via the `aws` CLI, using the
//     per-account profile from AWS_PROFILE / AWS_PROFILE_<ENV> (see awsProfile).
//     If the session is stale it runs `aws sso login` once against the SSO
//     session (ssoSession, default `unkey`), then retries. STRIPE_SECRET_ID and
//     AWS_REGION override the secret name and region.
//
// It refuses to run when the key's live/test nature does not match env, so you
// can never point a live key at a "sandbox" run or vice versa.
func New(ctx context.Context, env pricing.Environment) (*Client, error) {
	key, err := resolveKey(ctx, env)
	if err != nil {
		return nil, err
	}
	if err := guardKeyMatchesEnv(key, env); err != nil {
		return nil, err
	}
	return &Client{Client: stripe.NewClient(key), Env: env}, nil
}

func resolveKey(ctx context.Context, env pricing.Environment) (string, error) {
	// A key in the environment wins; this is the whole setup for sandbox/local.
	if k := strings.TrimSpace(os.Getenv("STRIPE_SECRET_KEY")); k != "" {
		return k, nil
	}

	profile, err := awsProfile(env)
	if err != nil {
		return "", err
	}

	key, err := keyFromSecretsManager(ctx, profile)
	if err == nil {
		return key, nil
	}

	// A stale/absent SSO session is the common cause; log in once against the SSO
	// session (not the per-account profile) and retry.
	if loginErr := ssoLogin(ctx, ssoSession()); loginErr != nil {
		return "", fmt.Errorf("%w (and `aws sso login` failed: %v)", err, loginErr)
	}

	return keyFromSecretsManager(ctx, profile)
}

func keyFromSecretsManager(ctx context.Context, profile string) (string, error) {
	id := secretID()
	cmd := exec.CommandContext(ctx, "aws", "secretsmanager", "get-secret-value",
		"--profile", profile,
		"--region", region(),
		"--secret-id", id,
		"--query", "SecretString",
		"--output", "text",
		"--no-cli-pager",
	)

	var stderr strings.Builder
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("fetching %q from Secrets Manager (profile %q): %w: %s",
			id, profile, err, strings.TrimSpace(stderr.String()))
	}

	var payload struct {
		APIKey string `json:"api_key"`
	}

	if err := json.Unmarshal(out, &payload); err != nil {
		return "", fmt.Errorf("parsing %q secret JSON: %w", id, err)
	}

	if payload.APIKey == "" {
		return "", fmt.Errorf("%q secret has empty api_key", id)
	}

	return payload.APIKey, nil
}

func ssoLogin(ctx context.Context, session string) error {
	cmd := exec.CommandContext(ctx, "aws", "sso", "login", "--sso-session", session, "--no-cli-pager")
	cmd.Stdin, cmd.Stdout, cmd.Stderr = os.Stdin, os.Stdout, os.Stderr
	return cmd.Run()
}

// guardKeyMatchesEnv rejects a live key on a non-prod run, or a test key on a
// production run, so the wrong account can't be pointed at the wrong catalog.
func guardKeyMatchesEnv(key string, env pricing.Environment) error {
	live := strings.HasPrefix(key, "sk_live_") || strings.HasPrefix(key, "rk_live_")
	switch env {
	case pricing.EnvProduction:
		if !live {
			return errors.New("production selected but the key is not a live key (sk_live_/rk_live_)")
		}
	default:
		if live {
			return fmt.Errorf("environment %q selected but the key is a LIVE key; refusing to run", env)
		}
	}
	return nil
}
