package privatenetwork

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestServiceName(t *testing.T) {
	require.Equal(t, "unkey-private-680e7f7bd57aa679f56c8c6d437f7091", ServiceName("dep_123", 8080))
}

func TestServiceHost(t *testing.T) {
	require.Equal(t, "unkey-private-a.customer-1.svc.cluster.local", ServiceHost("unkey-private-a", "customer-1"))
}
