package main

import (
	"github.com/unkeyed/unkey/build/util"
	"github.com/unkeyed/unkey/svc/logdrain"
)

func main() {
	util.RunServiceCommand("logdrain", "Run the Unkey logdrain delivery service", logdrain.Run)
}
