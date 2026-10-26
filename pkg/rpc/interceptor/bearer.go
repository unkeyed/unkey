package interceptor

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"connectrpc.com/connect"
)

// NewBearerAuth requires exactly one token source for unary RPCs. Token files
// are reopened on every call to support rotation; read failures fail closed.
func NewBearerAuth(token, tokenFile string) connect.UnaryInterceptorFunc {
	return func(next connect.UnaryFunc) connect.UnaryFunc {
		return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
			if (token == "") == (tokenFile == "") {
				return nil, connect.NewError(connect.CodeUnauthenticated, errors.New("exactly one bearer token source is required"))
			}
			bearer := token
			if tokenFile != "" {
				var err error
				bearer, err = readBearerToken(tokenFile)
				if err != nil {
					return nil, connect.NewError(connect.CodeUnauthenticated, err)
				}
			}
			req.Header().Set("Authorization", "Bearer "+bearer)
			return next(ctx, req)
		}
	}
}

func readBearerToken(path string) (token string, err error) {
	file, err := os.Open(path)
	if err != nil {
		return "", fmt.Errorf("open bearer token file: %w", err)
	}
	defer func() { err = errors.Join(err, file.Close()) }()
	const sizeMax = 16 * 1024
	data, err := io.ReadAll(io.LimitReader(file, sizeMax+1))
	if err != nil {
		return "", fmt.Errorf("read bearer token file: %w", err)
	}
	token = strings.TrimSpace(string(data))
	if token == "" || len(data) > sizeMax || strings.ContainsAny(token, " \t\r\n") {
		return "", errors.New("bearer token file is empty or invalid")
	}
	return token, nil
}
