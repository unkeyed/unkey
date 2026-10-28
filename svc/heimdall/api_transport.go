package heimdall

import (
	"context"
	"errors"
	"net/http"

	"github.com/unkeyed/unkey/svc/heimdall/internal/metrics"
)

type apiTransport struct {
	http.RoundTripper
}

func (t apiTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	res, err := t.RoundTripper.RoundTrip(req)
	if !errors.Is(req.Context().Err(), context.Canceled) && (err != nil || res.StatusCode >= http.StatusBadRequest) {
		metrics.KubernetesAPIRequestErrors.Inc()
	}
	return res, err
}
