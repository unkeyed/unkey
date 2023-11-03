package integration_test

import (
	"fmt"
	"github.com/joho/godotenv"
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
	err := godotenv.Load()
	if err != nil {
		panic(err)
	}
	e := env.Env{
		ErrorHandler: func(err error) {
			panic(err)
		},
	}

	ROOT_KEY = e.String("UNKEY_ROOT_KEY")
	BASE_URL = e.String("UNKEY_BASE_URL", "https://api.unkey.dev")
	fmt.Println("BASE_URL: ", BASE_URL)
	WORKSPACE_ID = e.String("UNKEY_WORKSPACE_ID")

	os.Exit(m.Run())

}
