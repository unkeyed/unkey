package auth

import (
	"github.com/unkeyed/unkey/cmd/login"
	"github.com/unkeyed/unkey/pkg/cli"
)

// Cmd is the auth command for managing CLI authentication.
var Cmd = &cli.Command{
	Name:        "auth",
	Usage:       "Manage authentication",
	Description: "Sign in with a browser, or store a root key, for the Unkey API.",
	Flags:       []cli.Flag{},
	Commands: []*cli.Command{
		login.New(),
	},
}
