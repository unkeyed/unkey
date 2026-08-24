package util

import (
	"os"

	"github.com/unkeyed/unkey/pkg/cli"
)

// defaultWorkOSClientID is the Unkey CLI's public WorkOS client ID for the OAuth
// device flow. Device flow uses a public client (no secret), so baking the ID
// into the distributed binary is safe — the risk is misconfiguration (shipping
// the wrong environment's ID), not leakage.
//
// It is a var, not a const, so release builds inject the per-environment value at
// link time:
//
//	go build -ldflags "-X 'github.com/unkeyed/unkey/cmd/api/util.defaultWorkOSClientID=client_...'" .
//
// The build task reads UNKEY_WORKOS_CLIENT_ID from the environment and performs
// this injection (see .mise/tasks/build). When left empty (local/dev builds),
// the ID must be supplied at runtime via --workos-client-id or the
// UNKEY_WORKOS_CLIENT_ID environment variable.
var defaultWorkOSClientID = ""

// Disclaimer is appended to command descriptions to inform users about the CLI's stability guarantees.
const Disclaimer = `

Note: This CLI is early and provided on a best-effort basis. There are no breaking change guarantees for commands, flags, or output format.`

// RootKeyFlag returns a flag for overriding the root key used for authentication.
func RootKeyFlag() *cli.StringFlag {
	return cli.String("root-key", "Override root key for authentication", cli.EnvVar("UNKEY_ROOT_KEY"))
}

// APIURLFlag returns a flag for overriding the API base URL.
func APIURLFlag() *cli.StringFlag {
	return cli.String("api-url", "Override API base URL", cli.EnvVar("UNKEY_API_BASE_URL"), cli.Default("https://api.unkey.com"))
}

// ConfigFlag returns a flag for overriding the config file location.
// Defaults to ~/.unkey/config.toml. If the home directory cannot be determined,
// the flag has no default and must be provided explicitly.
func ConfigFlag() *cli.StringFlag {
	opts := []cli.FlagOption{cli.EnvVar("UNKEY_CONFIG")}
	if defaultPath, err := cli.UserConfigPath(); err == nil {
		opts = append(opts, cli.Default(defaultPath))
	}
	return cli.String("config", "Path to config file", opts...)
}

// OutputFlag returns a flag for controlling output format.
func OutputFlag() *cli.StringFlag {
	return cli.String("output", "Output format. Use 'json' for raw JSON output suitable for piping.", cli.EnvVar("UNKEY_OUTPUT"))
}

// WorkOSClientIDFlag returns a flag for overriding the WorkOS client ID used by
// the OAuth device flow.
func WorkOSClientIDFlag() *cli.StringFlag {
	return cli.String("workos-client-id", "WorkOS client ID for CLI OAuth", cli.EnvVar("UNKEY_WORKOS_CLIENT_ID"), cli.Default(defaultWorkOSClientID))
}

// ResolveWorkOSClientID returns the WorkOS client ID from, in order: the
// --workos-client-id flag if set, the UNKEY_WORKOS_CLIENT_ID environment
// variable, or the baked-in default. It works on commands that do not register
// the flag (e.g. `unkey api ...`, which refreshes tokens via CreateClient) by
// falling back to the environment and default.
func ResolveWorkOSClientID(cmd *cli.Command) string {
	if v := cmd.String("workos-client-id"); v != "" {
		return v
	}
	if v := os.Getenv("UNKEY_WORKOS_CLIENT_ID"); v != "" {
		return v
	}
	return defaultWorkOSClientID
}
