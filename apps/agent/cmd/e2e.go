/*
Copyright © 2023 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"math/rand"
	"os"

	"github.com/spf13/cobra"
	"github.com/unkeyed/unkey/apps/agent/pkg/integration"
)

var (
	baseUrl string
	rootKey string
)

func init() {
	rootCmd.AddCommand(e2eCmd)
	e2eCmd.Flags().StringVar(&baseUrl, "base-url", "https://api.unkey.dev", "The base url of the API including the protocol")
	e2eCmd.Flags().StringVar(&rootKey, "root-key", "", "The root key to use, if not provided will use the UNKEY_ROOT_KEY environment variable")

}

// e2eCmd represents the e2e command
var e2eCmd = &cobra.Command{
	Use:   "e2e",
	Short: "Run end to end tests",
	Run: func(cmd *cobra.Command, args []string) {

		env := integration.Env{
			BaseUrl: baseUrl,
			RootKey: rootKey,
		}
		if env.RootKey == "" {
			env.RootKey = os.Getenv("UNKEY_ROOT_KEY")
		}

		if env.RootKey == "" {
			cmd.PrintErrln("root key not provided")
			os.Exit(1)
		}

		scenarios := []integration.Scenario{
			integration.CreateVerifyDeleteKeys,
			integration.UpdateRemaining,
			integration.ListKeys,
		}
		// shuffle
		rand.Shuffle(len(scenarios), func(i, j int) {
			scenarios[i], scenarios[j] = scenarios[j], scenarios[i]
		})

		for _, scenario := range scenarios {
			scenario.Run(cmd.Context(), env)
		}

	},
}
