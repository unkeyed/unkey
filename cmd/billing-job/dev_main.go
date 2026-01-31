package main

import (
	"context"
	"os"

	billingjob "github.com/unkeyed/unkey/cmd/billing-job"
)

func main() {
	if err := billingjob.Cmd.Run(context.Background(), billingjob.Cmd); err != nil {
		os.Exit(1)
	}
}