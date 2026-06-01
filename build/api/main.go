package main

import (
	"github.com/unkeyed/unkey/build/util"
	"github.com/unkeyed/unkey/svc/api"
)

func main() {
	util.RunServiceCommand("api", "Run the Unkey API server", api.Run)
}

