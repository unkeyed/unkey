package main

import (
	"github.com/unkeyed/unkey/build/util"
	ctrlapi "github.com/unkeyed/unkey/svc/ctrl/api"
)

func main() {
	util.RunServiceCommand("control-api", "Run the Unkey control plane API", ctrlapi.Run)
}
