package api

import (
	"crypto/subtle"
	"net/http"
	"strings"

	"github.com/danielgtaylor/huma/v2"
)

func (s *Server) BearerAuthFromSecret(secret string) func(huma.Context, func(huma.Context)) {

	secretB := []byte(secret)

	return func(ctx huma.Context, next func(huma.Context)) {

		authorizationHeader := ctx.Header("Authorization")
		if authorizationHeader == "" {
			huma.WriteErr(s.api, ctx, http.StatusUnauthorized, "Authorization header is required")
			return
		}

		token := strings.TrimPrefix(authorizationHeader, "Bearer ")
		if token == "" {
			huma.WriteErr(s.api, ctx, http.StatusUnauthorized, "Bearer token is required")
			return
		}

		if subtle.ConstantTimeCompare([]byte(token), secretB) != 1 {
			huma.WriteErr(s.api, ctx, http.StatusUnauthorized, "Invalid bearer token")

			return
		}

		next(ctx)
	}

}
