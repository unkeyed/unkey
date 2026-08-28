package engine

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/svc/logdrain/internal/db"
)

func TestFactoryBuild_RejectsMalformedWireFormat(t *testing.T) {
	drain := db.GetLeasedAndDueLogdrainRow{Config: []byte{0xff}}
	_, err := (factory{}).build(context.Background(), drain)
	require.ErrorContains(t, err, "decode logdrain config")
}
