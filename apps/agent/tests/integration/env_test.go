package integration_test

import (
	"fmt"
	"testing"

	"os"

	"github.com/unkeyed/unkey/apps/agent/pkg/env"
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

	ROOT_KEY = e.String("INTEGRATION_ROOT_KEY")
	BASE_URL = e.String("INTEGRATION_BASE_URL", "https://api.unkey.dev")
	fmt.Println("INTEGRATION_BASE_URL: ", BASE_URL)
	WORKSPACE_ID = e.String("INTEGRATION_WORKSPACE_ID")

	os.Exit(m.Run())

}
