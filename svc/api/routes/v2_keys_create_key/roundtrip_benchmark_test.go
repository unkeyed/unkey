package handler_test

import (
	"fmt"
	"os"
	"testing"
	"time"
)

func TestCreateKeyProtocolExchangeBenchmark(t *testing.T) {
	if os.Getenv("UNKEY_RUN_SLOW_CREATE_KEY_BENCHMARK") != "1" {
		t.Skip("set UNKEY_RUN_SLOW_CREATE_KEY_BENCHMARK=1 to run the protocol latency simulation")
	}

	const latency = 225 * time.Millisecond
	tests := []struct {
		request   string
		version   string
		exchanges []string
	}{
		{
			request: "basic",
			version: "before",
			exchanges: []string{
				"api and keyspace lookup",
				"transactional write batch",
			},
		},
		{
			request: "basic",
			version: "after",
			exchanges: []string{
				"api and keyspace lookup",
				"transactional write batch",
			},
		},
		{
			request: "maximal",
			version: "before",
			exchanges: []string{
				"api and keyspace lookup",
				"begin transaction",
				"identity, permission, role, and project lookup",
				"identity insert",
				"key insert",
				"encrypted key insert",
				"ratelimit insert",
				"permission insert",
				"key-permission insert",
				"key-role insert",
				"audit outbox insert",
				"commit transaction",
			},
		},
		{
			request: "maximal",
			version: "after",
			exchanges: []string{
				"api and keyspace lookup",
				"identity, permission, role, and project lookup",
				"transactional write batch",
			},
		},
	}

	fmt.Printf("simulated database latency per protocol exchange: %s\n", latency)
	fmt.Println("request\tversion\texchanges\tmeasured wall time")
	for _, test := range tests {
		started := time.Now()
		for range test.exchanges {
			time.Sleep(latency)
		}
		elapsed := time.Since(started)
		fmt.Printf("%s\t%s\t%d\t%s\n", test.request, test.version, len(test.exchanges), elapsed.Round(time.Millisecond))
	}
}
