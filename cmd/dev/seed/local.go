package seed

import (
	"context"

	devseed "github.com/unkeyed/unkey/internal/devtools/seed"
	"github.com/unkeyed/unkey/pkg/cli"
)

var localCmd = &cli.Command{
	Name:  "local",
	Usage: "Seed database with workspace, project, environment, API, and root key for local development",
	Flags: []cli.Flag{
		cli.String("database-primary", "MySQL database DSN", cli.Default("unkey:password@tcp(127.0.0.1:3306)/unkey?parseTime=true&interpolateParams=true"), cli.EnvVar("UNKEY_DATABASE_PRIMARY")),
		cli.String("slug", "Slug used to generate all IDs and names (e.g., 'flo' creates ws_flo, proj_flo, etc.)", cli.Default("local")),
		cli.String("org-id", "Organization ID for auth matching (defaults to org_localdefault for local auth)", cli.Default("org_localdefault")),
		cli.String("ctrl-url", "Control plane API URL", cli.Default("http://localhost:7091"), cli.EnvVar("UNKEY_CTRL_URL")),
		cli.String("api-key", "API key for control plane authentication", cli.Default("your-local-dev-key"), cli.EnvVar("UNKEY_API_KEY")),
		cli.String("output", "Path to write generated environment variables", cli.Default("dev/.env.seed")),
		cli.Bool("portal", "Also seed portal configuration and branding for this workspace"),
	},
	Action: seedLocal,
}

func seedLocal(ctx context.Context, cmd *cli.Command) error {
	_, err := devseed.SeedLocal(ctx, devseed.LocalParams{
		DatabasePrimary: cmd.RequireString("database-primary"),
		Slug:            cmd.String("slug"),
		OrgID:           cmd.String("org-id"),
		CtrlURL:         cmd.String("ctrl-url"),
		APIKey:          cmd.String("api-key"),
		Output:          cmd.String("output"),
		Portal:          cmd.Bool("portal"),
	})
	return err
}
