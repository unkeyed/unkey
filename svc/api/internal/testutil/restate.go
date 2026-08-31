package testutil

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	restateingress "github.com/restatedev/sdk-go/ingress"
)

// NewRestateIngressClient returns an ingress client backed by an HTTP server
// that responds with statusCode.
func NewRestateIngressClient(t testing.TB, statusCode int) *restateingress.Client {
	t.Helper()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(statusCode)
		if _, err := io.WriteString(w, "{\"invocationId\":\"inv_test\",\"status\":\"Accepted\"}"); err != nil {
			t.Errorf("write Restate response: %v", err)
		}
	}))
	t.Cleanup(server.Close)
	return restateingress.NewClient(server.URL)
}

// NewUnavailableRestateIngressClient returns an ingress client whose HTTP
// endpoint is closed.
func NewUnavailableRestateIngressClient(t testing.TB) *restateingress.Client {
	t.Helper()

	server := httptest.NewServer(http.NotFoundHandler())
	url := server.URL
	server.Close()
	return restateingress.NewClient(url)
}
