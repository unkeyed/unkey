package main

import (
	"github.com/unkeyed/unkey/build/util"
	"github.com/unkeyed/unkey/svc/vault"
)

func main() {
	util.RunServiceCommand("vault", "Run Unkey Vault", vault.Run)
}
