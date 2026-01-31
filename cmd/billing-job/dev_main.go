package main

import (
	"context"
	"fmt"
	"os"

	billingjob "github.com/unkeyed/unkey/cmd/billing-job"
)

func main() {
	fmt.Println("DEBUG: main() called, args:", os.Args)
	fmt.Println("DEBUG: billingjob.Cmd is", billingjob.Cmd)

	if err := billingjob.Cmd.Run(context.Background(), os.Args[1:]); err != nil {
		fmt.Println("DEBUG: Run returned error:", err)
		os.Exit(1)
	}
	fmt.Println("DEBUG: main() completed")
}