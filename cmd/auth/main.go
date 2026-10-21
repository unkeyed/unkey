package auth

import (
	"github.com/unkeyed/unkey/cmd/auth/login"
	"github.com/unkeyed/unkey/cmd/auth/logout"
	"github.com/unkeyed/unkey/pkg/cli"
)

// Cmd is the auth command for managing CLI authentication.
var Cmd = &cli.Command{
	Name:        "auth",
	Usage:       "Manage authentication",
	Description: "Sign in with a browser, store a root key, or sign out of the Unkey API.",
	Flags:       []cli.Flag{},
	Commands: []*cli.Command{
		login.New(),
		logout.New(),
	},
}
