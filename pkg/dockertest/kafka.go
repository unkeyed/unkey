package dockertest

import (
	"fmt"
	"testing"
	"time"
)

const (
	kafkaImage = "bufbuild/bufstream:0.4.4"
	kafkaPort  = "9092/tcp"
)

var kafkaCtr shared

// kafkaContainerConfig returns the container configuration for Kafka (bufstream).
func kafkaContainerConfig() containerConfig {
	return containerConfig{
		Image:        kafkaImage,
		ExposedPorts: []string{kafkaPort},
		WaitStrategy: NewTCPWait(kafkaPort),
		WaitTimeout:  30 * time.Second,
		Env:          map[string]string{},
		Cmd:          []string{"serve", "--inmemory"},
		Tmpfs:        nil,
		SkipCleanup:  false,
	}
}

// Kafka starts (or reuses) a shared Kafka (bufstream) container and returns
// the broker addresses.
//
// The container starts on the first call in the process and is reused by all
// subsequent calls.
func Kafka(t *testing.T) []string {
	t.Helper()

	ctr := kafkaCtr.get(t, kafkaContainerConfig())
	port := ctr.Port(kafkaPort)

	return []string{fmt.Sprintf("127.0.0.1:%s", port)}
}
