package session

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/unkeyed/unkey/internal/sdk"
	"github.com/unkeyed/unkey/internal/sdk/models/apierrors"
	"github.com/unkeyed/unkey/internal/sdk/models/components"
)

// The server lets a rotated refresh token be presented again for 10 seconds
// and answers with a fresh pair. Both attempts have to fit in that window, so
// a rotation whose response was lost is recovered by the retry instead of
// revoking the session on the next command.
var refreshAttemptTimeout = 4 * time.Second

// Refresh exchanges a refresh token for a new session pair. A transport
// failure, timeout, or 5xx response is retried once with the same token.
func Refresh(ctx context.Context, apiURL string, refreshToken string) (Session, error) {
	next, err := refreshOnce(ctx, apiURL, refreshToken)
	if err == nil || ctx.Err() != nil {
		return next, err
	}
	var transport *sdk.TransportError
	var response *sdk.ResponseError
	if errors.As(err, &transport) || (errors.As(err, &response) && response.StatusCode >= 500) {
		return refreshOnce(ctx, apiURL, refreshToken)
	}
	return next, err
}

func refreshOnce(ctx context.Context, apiURL string, refreshToken string) (Session, error) {
	api := sdk.NewClient(apiURL, &http.Client{Timeout: refreshAttemptTimeout})
	request := components.CreateV2CliExchangeTokenRequestBodyV2CliExchangeTokenRequestBody2(components.V2CliExchangeTokenRequestBody2{RefreshToken: refreshToken, LoginID: nil})
	res, err := api.Cli.ExchangeToken(ctx, request)
	if err != nil {
		var responseError *sdk.ResponseError
		if errors.As(err, &responseError) {
			return Session{}, responseError
		}
		var response *apierrors.APIError
		if errors.As(err, &response) {
			return Session{}, fmt.Errorf("refresh returned %d", response.StatusCode)
		}
		return Session{}, err
	}
	if res.V2CliExchangeTokenResponseBody == nil {
		return Session{}, fmt.Errorf("refresh returned %d", res.HTTPMeta.Response.StatusCode)
	}
	result := res.V2CliExchangeTokenResponseBody.Data
	if result.Status != components.StatusComplete {
		return Session{}, fmt.Errorf("refresh returned unexpected status %q", result.Status)
	}
	credentials := result.Credentials
	if credentials == nil || credentials.AccessToken == "" || credentials.RefreshToken == "" || credentials.WorkspaceID == "" || credentials.SessionID == "" || credentials.ExpiresIn <= 0 {
		return Session{}, fmt.Errorf("refresh completed without a session")
	}
	return Session{
		WorkspaceID:  credentials.WorkspaceID,
		SessionID:    credentials.SessionID,
		AccessToken:  credentials.AccessToken,
		RefreshToken: credentials.RefreshToken,
		ExpiresAt:    time.Now().Add(time.Duration(credentials.ExpiresIn) * time.Second).Unix(),
		APIURL:       NormalizeAPIURL(apiURL),
	}, nil
}
