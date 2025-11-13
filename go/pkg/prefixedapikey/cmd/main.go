package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/unkeyed/unkey/go/pkg/cli"
	"github.com/unkeyed/unkey/go/pkg/prefixedapikey"
)

func main() {
	app := &cli.Command{
		Name:        "prefixed-api-key",
		Usage:       "Generate prefixed API keys",
		Description: "A CLI tool for generating prefixed API keys compatible with github.com/seamapi/prefixed-api-key",
		Version:     "1.0.0",
		Action:      generateAction,
		Flags: []cli.Flag{
			cli.String("prefix", "Key prefix (e.g., 'myapp', 'prod', 'dev')", cli.Required()),
			cli.String("short-prefix", "Optional prefix for the short token (e.g., 'test')"),
			cli.Int("short-length", "Length of the short token in bytes", cli.Default(8)),
			cli.Int("long-length", "Length of the long token in bytes", cli.Default(24)),
			cli.Int("count", "Number of keys to generate", cli.Default(1)),
			cli.Bool("json", "Output in JSON format"),
		},
	}

	if err := app.Run(context.Background(), os.Args); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func generateAction(ctx context.Context, cmd *cli.Command) error {
	opts := &prefixedapikey.GenerateAPIKeyOptions{
		KeyPrefix:        cmd.RequireString("prefix"),
		ShortTokenPrefix: cmd.String("short-prefix"),
		ShortTokenLength: cmd.Int("short-length"),
		LongTokenLength:  cmd.Int("long-length"),
	}

	count := cmd.Int("count")
	useJSON := cmd.Bool("json")

	// Validate count
	if count < 1 {
		return fmt.Errorf("count must be at least 1")
	}

	// Generate keys
	var keys []*prefixedapikey.APIKey
	for i := 0; i < count; i++ {
		key, err := prefixedapikey.GenerateAPIKey(opts)
		if err != nil {
			return fmt.Errorf("failed to generate key: %w", err)
		}
		keys = append(keys, key)
	}

	// Output results
	if useJSON {
		return outputJSON(keys)
	}

	return outputTable(keys)
}

func outputJSON(keys []*prefixedapikey.APIKey) error {
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	return encoder.Encode(keys)
}

func outputTable(keys []*prefixedapikey.APIKey) error {
	if len(keys) == 1 {
		// Single key output - detailed format
		key := keys[0]
		fmt.Println("Generated API Key:")
		fmt.Println("==================")
		fmt.Printf("Full Token:       %s\n", key.Token)
		fmt.Printf("Short Token:      %s\n", key.ShortToken)
		fmt.Printf("Long Token:       %s\n", key.LongToken)
		fmt.Printf("Long Token Hash:  %s\n", key.LongTokenHash)
		fmt.Println()
		fmt.Println("Store the 'Long Token Hash' in your database.")
		fmt.Println("Give the 'Full Token' to your user (only shown once).")
	} else {
		// Multiple keys - compact table format
		fmt.Printf("Generated %d API Keys:\n", len(keys))
		fmt.Println("==================================================")
		fmt.Printf("%-10s %-40s %s\n", "Index", "Full Token", "Hash (store this)")
		fmt.Println("--------------------------------------------------")
		for i, key := range keys {
			// Truncate token for display if too long
			token := key.Token
			if len(token) > 40 {
				token = token[:37] + "..."
			}
			fmt.Printf("%-10d %-40s %s...\n", i+1, token, key.LongTokenHash[:16])
		}
		fmt.Println("==================================================")
		fmt.Println()
		fmt.Println("Use --json flag for full output including all tokens and hashes.")
	}

	return nil
}
