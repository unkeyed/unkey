// Command depotbuildtimeout sets a 30 minute build timeout on every existing
// Depot project. New projects get the same timeout when the ctrl worker
// creates them.
//
// Usage:
//
//	DEPOT_TOKEN=... mise exec -- go run ./tools/depotbuildtimeout [--dry-run]
package main

import (
	"context"
	"fmt"
	"net/http"
	"os"

	"buf.build/gen/go/depot/api/connectrpc/go/depot/core/v1/corev1connect"
	corev1 "buf.build/gen/go/depot/api/protocolbuffers/go/depot/core/v1"
	"connectrpc.com/connect"
	"github.com/unkeyed/unkey/pkg/cli"
)

const buildTimeoutMinutes = 30

func main() {
	cmd := &cli.Command{
		Name:        "depotbuildtimeout",
		Usage:       "Set a 30 minute build timeout on existing Depot projects",
		Description: "",
		Examples: []string{
			"depotbuildtimeout --dry-run",
			"depotbuildtimeout",
		},
		AcceptsArgs: false,
		Version:     "",
		Aliases:     []string{},
		Commands:    []*cli.Command{},
		Flags: []cli.Flag{
			cli.String("token", "Depot API token", cli.EnvVar("DEPOT_TOKEN"), cli.Required()),
			cli.Bool("dry-run", "Print the projects that would change without updating them"),
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			return run(ctx, cmd.String("token"), cmd.Bool("dry-run"))
		},
	}

	if err := cmd.Run(context.Background(), os.Args); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

func run(ctx context.Context, token string, dryRun bool) error {
	auth := connect.UnaryInterceptorFunc(func(next connect.UnaryFunc) connect.UnaryFunc {
		return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
			req.Header().Set("Authorization", "Bearer "+token)
			return next(ctx, req)
		}
	})
	client := corev1connect.NewProjectServiceClient(http.DefaultClient, "https://api.depot.dev", connect.WithInterceptors(auth))

	var updated, skipped int
	var pageToken *string
	for {
		//nolint:exhaustruct // optional filters
		res, err := client.ListProjects(ctx, connect.NewRequest(&corev1.ListProjectsRequest{PageToken: pageToken}))
		if err != nil {
			return fmt.Errorf("list projects: %w", err)
		}

		for _, p := range res.Msg.GetProjects() {
			if p.GetBuildTimeoutMinutes() == buildTimeoutMinutes {
				skipped++
				continue
			}

			fmt.Printf("%s %s: %d -> %d minutes\n", p.GetProjectId(), p.GetName(), p.GetBuildTimeoutMinutes(), buildTimeoutMinutes)
			if dryRun {
				continue
			}

			//nolint:exhaustruct // unset fields keep their current values
			_, err := client.UpdateProject(ctx, connect.NewRequest(&corev1.UpdateProjectRequest{
				ProjectId: p.GetProjectId(),
				BuildTimeoutMinutes: &corev1.NullableInt32{
					Kind: &corev1.NullableInt32_Value{Value: buildTimeoutMinutes},
				},
			}))
			if err != nil {
				return fmt.Errorf("update project %s: %w", p.GetProjectId(), err)
			}
			updated++
		}

		if res.Msg.GetNextPageToken() == "" {
			break
		}
		pageToken = res.Msg.NextPageToken
	}

	fmt.Printf("updated %d, already set %d\n", updated, skipped)
	return nil
}
