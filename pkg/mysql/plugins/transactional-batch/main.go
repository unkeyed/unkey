// Package main generates typed wrappers for transactional MySQL statement batches.
package main

import (
	"context"

	"github.com/sqlc-dev/plugin-sdk-go/codegen"
	"github.com/sqlc-dev/plugin-sdk-go/plugin"
)

func main() {
	codegen.Run(Generate)
}

func Generate(_ context.Context, req *plugin.GenerateRequest) (*plugin.GenerateResponse, error) {
	generator := NewGenerator()
	if err := generator.Configure(req.GetPluginOptions()); err != nil {
		return nil, err
	}
	return generator.Generate(req)
}
