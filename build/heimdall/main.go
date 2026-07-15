package main

import (
	"github.com/unkeyed/unkey/build/util"
	"github.com/unkeyed/unkey/svc/heimdall"
)

func main() {
	util.RunServiceCommand("heimdall", "Run the resource metering agent", heimdall.Run)
}
