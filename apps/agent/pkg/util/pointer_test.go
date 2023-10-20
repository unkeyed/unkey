package util_test

import (
	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/apps/agent/pkg/util"
	"testing"
)

func TestPointer(t *testing.T) {
	val := "hello"
	p := util.Pointer(val)

	require.Equal(t, val, *p)

}

func TestPointer_IsACopy(t *testing.T) {
	val := "hello"
	p := util.Pointer(val)

	// let's modify the value of the pointer
	*p = "world"
	// the original value is not modified
	require.Equal(t, val, "hello")
	// the pointer is modified
	require.Equal(t, *p, "world")

}
