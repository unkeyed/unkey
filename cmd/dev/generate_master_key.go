package dev

import (
	"context"
	"fmt"

	"github.com/unkeyed/unkey/pkg/cli"
	"github.com/unkeyed/unkey/pkg/vault/keys"
)

var generateMasterKeyCmd = &cli.Command{
	Name:        "generate-master-key",
	Usage:       "Generate a new master key for vault encryption",
	Description: "Generates a new master key and prints the base64-encoded representation.",
	Action: func(ctx context.Context, cmd *cli.Command) error {
		_, encoded, err := keys.GenerateMasterKey()
		if err != nil {
			return fmt.Errorf("failed to generate master key: %w", err)
		}

		fmt.Println(encoded)
		return nil
	},
}
