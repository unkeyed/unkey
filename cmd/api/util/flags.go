package util

import "github.com/unkeyed/unkey/pkg/cli"

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
