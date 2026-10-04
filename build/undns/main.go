package main

import (
	"github.com/unkeyed/unkey/build/util"
	"github.com/unkeyed/unkey/svc/undns"
)

func main() {
	util.RunServiceCommand("undns", "Run the private deployment DNS resolver", undns.Run)
}
