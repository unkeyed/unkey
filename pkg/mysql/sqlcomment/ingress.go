package sqlcomment

import (
	"net/http"
	"strings"
)

const restateInvokePrefix = "/invoke/"

// WrapRestateInvokeHandler annotates MySQL queries during a Restate handler with
// route=<service>/<method> and source=restate. Restate ingress uses paths like
// /invoke/hydra.v1.DeployService/Deploy.
func WrapRestateInvokeHandler(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if route, ok := restateInvokeRoute(r.URL.Path); ok {
			r = r.WithContext(WithDynamic(r.Context(), Dynamic{
				Route:  route,
				Source: "restate",
			}))
		}
		h.ServeHTTP(w, r)
	})
}

func restateInvokeRoute(path string) (string, bool) {
	if !strings.HasPrefix(path, restateInvokePrefix) {
		return "", false
	}

	rest := strings.TrimPrefix(path, restateInvokePrefix)
	parts := strings.SplitN(rest, "/", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return "", false
	}

	method := parts[1]
	if i := strings.IndexByte(method, '?'); i >= 0 {
		method = method[:i]
	}

	return parts[0] + "/" + method, true
}
