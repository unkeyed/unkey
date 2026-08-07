package main

import (
	"context"

	"github.com/unkeyed/unkey/build/util"
	"github.com/unkeyed/unkey/pkg/openapi/validation"
	"github.com/unkeyed/unkey/svc/api"
)

func main() {
	util.RunServiceCommand("api", "Run the Unkey API server", runAPI)
}

func runAPI(ctx context.Context, cfg api.Config) error {
	return api.Run(ctx, cfg, validation.NewFromBytes)
}
