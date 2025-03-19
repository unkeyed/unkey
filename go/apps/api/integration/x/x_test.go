package x

import (
	"testing"

	"github.com/unkeyed/unkey/go/pkg/testutil/containers"
)

func TestX(t *testing.T) {

	c := containers.New(t)
	c.BuildAndRunAPI()
	// Test code here
}
