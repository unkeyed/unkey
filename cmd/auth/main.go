package auth

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/unkeyed/unkey/pkg/cli"
	"golang.org/x/term"
)

// Cmd is the auth command for managing CLI authentication.
var Cmd = &cli.Command{
	Name:        "auth",
	Usage:       "Manage authentication",
	Description: "Authenticate with the Unkey API by providing your root key.",
	Flags:       []cli.Flag{},
	Commands: []*cli.Command{
		loginCmd,
	},
}

var loginCmd = &cli.Command{
	Name:  "login",
	Usage: "Authenticate with a root key",
	Description: `Store your Unkey root key locally so other commands can use it without passing --root-key every time.

The key is stored in ~/.unkey/config.toml.`,
	Flags: []cli.Flag{},
	Action: func(ctx context.Context, cmd *cli.Command) error {
		fmt.Print("Enter your root key: ")

		raw, err := term.ReadPassword(int(os.Stdin.Fd()))
		fmt.Println() // newline after hidden input
		if err != nil {
			return fmt.Errorf("failed to read input: %w", err)
		}
		key := strings.TrimSpace(string(raw))

		if key == "" {
			return fmt.Errorf("root key cannot be empty")
		}

		if err := cli.SaveUserConfig(cli.UserConfig{RootKey: key}); err != nil {
			return fmt.Errorf("failed to save config: %w", err)
		}

		path, err := cli.UserConfigPath()
		if err != nil {
			return fmt.Errorf("failed to resolve config path: %w", err)
		}
		fmt.Printf("Authentication successful. Key stored in %s\n", path)
		return nil
	},
}
