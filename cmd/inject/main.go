package main

import (
	"context"
	"fmt"
	"os"
)

func main() {
	if err := cmd.Run(context.Background(), os.Args); err != nil {
		fmt.Fprintf(os.Stderr, "inject: %v\n", err)
		os.Exit(1)
	}
}
