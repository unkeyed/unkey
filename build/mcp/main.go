package main

import (
	"context"
	"fmt"
	"os"

	"github.com/unkeyed/unkey/svc/mcp"
)

func main() {
	cfg, err := mcp.ConfigFromEnv(os.Getenv)
	if err != nil {
		fmt.Fprintln(os.Stderr, "invalid config:", err)
		os.Exit(1)
	}
	if err := mcp.Run(context.Background(), cfg); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
