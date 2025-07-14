package main

import (
	"context"
	"fmt"
	"os"

	"github.com/unkeyed/unkey/go/cmd/cli/app"
	"github.com/unkeyed/unkey/go/pkg/version"
)

func main() {
	c := app.New(
		os.Args,
		"unkey",
		"Deploy and manage your API versions",
		version.Version,
	)
	ctx := context.Background()
	if err := c.Run(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
