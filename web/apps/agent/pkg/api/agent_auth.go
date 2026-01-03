package api

import (
	"crypto/subtle"
	"net/http"
	"strings"

	"github.com/unkeyed/unkey/apps/agent/pkg/api/routes"
)

func newBearerAuthMiddleware(secret string) routes.Middeware {
	secretB := []byte(secret)

	return func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {

			authorizationHeader := r.Header.Get("Authorization")
			if authorizationHeader == "" {
				w.WriteHeader(401)
				_, err := w.Write([]byte("Authorization header is required"))
				if err != nil {
					http.Error(w, err.Error(), http.StatusInternalServerError)
				}
				return
			}

			token := strings.TrimPrefix(authorizationHeader, "Bearer ")
			if token == "" {
				w.WriteHeader(401)
				_, err := w.Write([]byte("Bearer token is required"))
				if err != nil {
					http.Error(w, err.Error(), http.StatusInternalServerError)
				}
				return

			}

			if subtle.ConstantTimeCompare([]byte(token), secretB) != 1 {
				w.WriteHeader(401)
				_, err := w.Write([]byte("Bearer token is invalid"))
				if err != nil {
					http.Error(w, err.Error(), http.StatusInternalServerError)
				}
				return
			}

			next(w, r)
		}
	}

}
