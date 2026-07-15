package main

import (
	"github.com/unkeyed/unkey/build/util"
	"github.com/unkeyed/unkey/svc/ctrl/worker"
)

func main() {
	util.RunServiceCommand("control-worker", "Run the Unkey control plane worker", worker.Run)
}
