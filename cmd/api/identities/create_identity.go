package identities

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/unkeyed/sdks/api/go/v2/models/components"
	"github.com/unkeyed/unkey/cmd/api/util"
	"github.com/unkeyed/unkey/pkg/cli"
)

func createIdentityCmd() *cli.Command {
	return &cli.Command{
		Name:  "create-identity",
		Usage: "Create an identity to group multiple API keys under a single entity.",
		Description: `Create an identity to group multiple API keys under a single entity. Identities enable shared rate limits and metadata across all associated keys.

Perfect for users with multiple devices, organizations with multiple API keys, or when you need unified rate limiting across different services.

Important
Requires identity.*.create_identity permission

For full documentation, see https://www.unkey.com/docs/api-reference/v2/identities/create-identity` + util.Disclaimer,
		Examples: []string{
			"unkey api identities create-identity --external-id=user_123",
			`unkey api identities create-identity --external-id=user_123 --meta-json='{"email":"alice@acme.com","name":"Alice Smith","plan":"premium"}'`,
			`unkey api identities create-identity --external-id=user_123 --ratelimits-json='[{"name":"requests","limit":1000,"duration":60000,"autoApply":false}]'`,
		},
		Flags: []cli.Flag{
			util.RootKeyFlag(),
			util.APIURLFlag(),
			util.ConfigFlag(),
			util.OutputFlag(),
			cli.String("external-id", "Your system's unique identifier for the user, organization, or entity.", cli.Required()),
			cli.String("meta-json", "JSON object of arbitrary metadata stored on the identity."),
			cli.String("ratelimits-json", "JSON array of shared rate limit configurations for all keys under this identity."),
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			client, err := util.CreateClient(cmd)
			if err != nil {
				return err
			}

			req := components.V2IdentitiesCreateIdentityRequestBody{
				ExternalID: cmd.String("external-id"),
				Meta:       nil,
				Ratelimits: nil,
			}

			if v := cmd.String("meta-json"); v != "" {
				var meta map[string]any
				if err := json.Unmarshal([]byte(v), &meta); err != nil {
					return fmt.Errorf("invalid JSON for --meta-json: %w", err)
				}
				req.Meta = meta
			}

			if v := cmd.String("ratelimits-json"); v != "" {
				var ratelimits []components.RatelimitRequest
				if err := json.Unmarshal([]byte(v), &ratelimits); err != nil {
					return fmt.Errorf("invalid JSON for --ratelimits-json: %w", err)
				}
				req.Ratelimits = ratelimits
			}

			start := time.Now()
			res, err := client.Identities.CreateIdentity(ctx, req)
			if err != nil {
				return fmt.Errorf("%s", util.FormatError(err))
			}
			return util.Output(cmd, res.V2IdentitiesCreateIdentityResponseBody, time.Since(start))
		},
	}
}
