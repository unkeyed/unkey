package handler

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/db"
)

func TestTransportRegistry_HTTP1(t *testing.T) {
	r := NewTransportRegistry()
	transport := r.Get(db.DeploymentsUpstreamProtocolHttp1)
	require.NotNil(t, transport)
}

func TestTransportRegistry_H2C(t *testing.T) {
	r := NewTransportRegistry()
	transport := r.Get(db.DeploymentsUpstreamProtocolH2c)
	require.NotNil(t, transport)
}

func TestTransportRegistry_H2CDiffersFromHTTP1(t *testing.T) {
	r := NewTransportRegistry()
	h1 := r.Get(db.DeploymentsUpstreamProtocolHttp1)
	h2c := r.Get(db.DeploymentsUpstreamProtocolH2c)
	require.NotEqual(t, h1, h2c, "h2c and http1 should be different transports")
}

func TestTransportRegistry_UnknownFallsBackToHTTP1(t *testing.T) {
	r := NewTransportRegistry()
	h1 := r.Get(db.DeploymentsUpstreamProtocolHttp1)

	require.Equal(t, h1, r.Get(db.DeploymentsUpstreamProtocol("something")))
}

func TestTransportRegistry_EmptyStringFallsBack(t *testing.T) {
	r := NewTransportRegistry()
	h1 := r.Get(db.DeploymentsUpstreamProtocolHttp1)
	require.Equal(t, h1, r.Get(db.DeploymentsUpstreamProtocol("")))
}
