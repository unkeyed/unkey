package cmd

import (
	"fmt"
	"time"

	"github.com/spf13/cobra"
	"github.com/unkeyed/unkey/apps/vault/pkg/keys"
)

func init() {
	rootCmd.AddCommand(GenerateKEK)
}

// AgentCmd represents the agent command
var GenerateKEK = &cobra.Command{
	Use:   "generate-kek",
	Short: "Generate and print a new master key",

	RunE: func(cmd *cobra.Command, args []string) error {
		kek, key, err := keys.GenerateMasterKey()
		if err != nil {
			return fmt.Errorf("failed to generate master key: %w", err)
		}

		fmt.Printf("Key ID  : %s\n", kek.Id)
		fmt.Printf("Created : %v\n", time.UnixMilli(kek.CreatedAt))
		fmt.Printf("Secret  : %s\n", key)
		return nil
	},
}
