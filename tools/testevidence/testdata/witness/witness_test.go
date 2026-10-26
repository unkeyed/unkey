package witness

import (
	"os"
	"testing"
)

var executions int

func TestMain(m *testing.M) {
	code := m.Run()
	if os.Getenv("UNKEY_EVIDENCE_TEST_OUTCOME") == "swallowed-failure" {
		os.Exit(0)
	}
	os.Exit(code)
}

func TestWitness(t *testing.T) {
	executions++
	switch os.Getenv("UNKEY_EVIDENCE_TEST_OUTCOME") {
	case "skip":
		t.Skip("intentional gate control")
	case "fail":
		t.Fatal("intentional gate control")
	case "fail-once", "swallowed-failure":
		if executions == 1 {
			t.Fatal("intentional first-execution failure")
		}
	}
}
