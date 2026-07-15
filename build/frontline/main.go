package main

import (
	"github.com/unkeyed/unkey/build/util"
	"github.com/unkeyed/unkey/svc/frontline"
)

func main() {
	util.RunServiceCommand("frontline", "Run the Unkey Frontline server", frontline.Run)
}
