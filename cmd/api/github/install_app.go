package github

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/unkeyed/unkey/cmd/api/util"
	"github.com/unkeyed/unkey/pkg/cli"
)

func installAppCmd() *cli.Command {
	return &cli.Command{
		Name:  "install-app",
		Usage: "Start installing the Unkey GitHub App for your workspace.",
		Description: `Start installing the Unkey GitHub App for your workspace. Returns a GitHub App install URL: open it in a browser to install the app and grant repository access. After installation GitHub returns to Unkey, which binds the installation to your workspace and lands you in the workspace settings.

Installation is workspace-wide and takes no parameters. Once installed, link repositories to individual apps with the git field on apps.createApp and apps.updateApp.

Required Permissions

Your root key must have the following permission:
- workspace.*.install_github

` + util.Disclaimer,
		Examples: []string{
			"unkey api github install-app",
		},
		Flags: []cli.Flag{
			cli.String("body", "Decode this JSON as the endpoint request body. This endpoint only accepts an empty JSON object."),
			util.RootKeyFlag(),
			util.APIURLFlag(),
			util.ConfigFlag(),
			util.OutputFlag(),
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			client, err := util.CreateClient(cmd)
			if err != nil {
				return err
			}

			if cmd.FlagIsSet("body") {
				decoder := json.NewDecoder(strings.NewReader(cmd.String("body")))
				decoder.DisallowUnknownFields()
				if err := decoder.Decode(&struct{}{}); err != nil {
					return fmt.Errorf("invalid JSON for --body: %w", err)
				}
				if err := decoder.Decode(&struct{}{}); err != io.EOF {
					if err == nil {
						return fmt.Errorf("invalid JSON for --body: multiple JSON values")
					}
					return fmt.Errorf("invalid JSON for --body: %w", err)
				}
			}

			res, err := client.Github.InstallApp(ctx)
			if err != nil {
				return fmt.Errorf("%s", util.FormatError(err))
			}
			return util.Output(cmd, res.V2GithubInstallAppResponseBody)
		},
	}
}
