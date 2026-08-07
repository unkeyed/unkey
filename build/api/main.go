package main

import (
	"context"

	"github.com/unkeyed/unkey/build/util"
	openapivalidation "github.com/unkeyed/unkey/pkg/openapi/validation"
	"github.com/unkeyed/unkey/svc/api"
)

func main() {
	util.RunServiceCommand("api", "Run the Unkey API server", run)
}

func run(ctx context.Context, cfg api.Config) error {
	return api.Run(ctx, cfg, openapivalidation.NewFromBytes)
}
