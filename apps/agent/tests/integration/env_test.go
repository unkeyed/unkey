package integration_test

import (
	"testing"

	"github.com/unkeyed/unkey/apps/agent/pkg/env"
	"os"
)

var (
	BASE_URL     string
	ROOT_KEY     string
	WORKSPACE_ID string
)

func TestMain(m *testing.M) {

	e := env.Env{
		ErrorHandler: func(err error) {
			panic(err)
		},
	}

	ROOT_KEY = e.String("UNKEY_ROOT_KEY")
	BASE_URL = e.String("UNKEY_BASE_URL", "https://api.unkey.dev")
	WORKSPACE_ID = e.String("UNKEY_WORKSPACE_ID")

	os.Exit(m.Run())

}
