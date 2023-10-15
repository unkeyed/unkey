package integration

import "sync/atomic"

var (
	stepId = atomic.Int32{}
)

func getStepId() int32 {
	return stepId.Add(1)
}
