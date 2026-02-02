package main

import (
	"context"
	"os"

	"github.com/unkeyed/unkey/cmd/billing-job"
)

func main() {
	if err := billingjob.Cmd.Run(context.Background(), os.Args[1:]); err != nil {
		os.Exit(1)
	}
}