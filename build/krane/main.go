package main

import (
	"github.com/unkeyed/unkey/build/util"
	"github.com/unkeyed/unkey/svc/krane"
)

func main() {
	util.RunServiceCommand("krane", "Run the Kubernetes deployment service", krane.Run)
}
