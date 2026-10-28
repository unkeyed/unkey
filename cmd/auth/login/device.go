package login

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/unkeyed/unkey/internal/sdk"
	"github.com/unkeyed/unkey/internal/sdk/models/components"
	"github.com/unkeyed/unkey/internal/sdk/polling"
)

func startLogin(ctx context.Context, client *http.Client, apiURL string, deviceName string) (components.V2CliStartDeviceLoginResponseData, error) {
	api := sdk.NewClient(apiURL, client)
	body := components.V2CliStartDeviceLoginRequestBody{DeviceName: nil}
	if deviceName != "" {
		body.DeviceName = &deviceName
	}
	res, err := api.Cli.StartDeviceLogin(ctx, body)
	if err != nil {
		return components.V2CliStartDeviceLoginResponseData{}, err
	}
	if res.V2CliStartDeviceLoginResponseBody == nil {
		return components.V2CliStartDeviceLoginResponseData{}, fmt.Errorf("unkey API returned %d", res.HTTPMeta.Response.StatusCode)
	}
	return res.V2CliStartDeviceLoginResponseBody.Data, nil
}

type issuedSession = components.V2CliExchangeTokenResponseData

func pollUntilKey(ctx context.Context, client *http.Client, apiURL string, loginID string, deadline time.Time, interval time.Duration, sleep func(context.Context, time.Duration) error) (issuedSession, error) {
	if deadline.IsZero() {
		deadline = time.Now().Add(10 * time.Minute)
	}
	pollCtx, cancel := context.WithDeadline(ctx, deadline)
	defer cancel()
	fmt.Println("Waiting for you to allow this login in the browser...")
	api := sdk.NewClient(apiURL, client)
	request := components.CreateV2CliExchangeTokenRequestBodyV2CliExchangeTokenRequestBody1(components.V2CliExchangeTokenRequestBody1{LoginID: loginID, RefreshToken: nil})
	res, err := api.WaitForDeviceLogin(pollCtx, request, interval, sleep)
	if ctx.Err() != nil {
		return issuedSession{}, ctx.Err()
	}
	var limit *polling.LimitCountError
	if errors.Is(err, context.DeadlineExceeded) || errors.As(err, &limit) {
		return issuedSession{}, fmt.Errorf("login timed out. Run unkey login again")
	}
	if err != nil {
		return issuedSession{}, err
	}
	if res == nil || res.V2CliExchangeTokenResponseBody == nil {
		return issuedSession{}, fmt.Errorf("login returned no response")
	}
	credentials := res.V2CliExchangeTokenResponseBody.Data
	if credentials.AccessToken == "" || credentials.RefreshToken == "" || credentials.WorkspaceID == "" || credentials.SessionID == "" || credentials.ExpiresIn <= 0 {
		return issuedSession{}, fmt.Errorf("login completed without a session")
	}
	return credentials, nil
}

func revokeIssuedSession(ctx context.Context, client *http.Client, apiURL string, issued issuedSession) error {
	api := sdk.NewClient(apiURL, client)
	res, err := api.Cli.RevokeSession(ctx, components.V2CliRevokeSessionRequestBody{SessionID: issued.SessionID}, sdk.Bearer(issued.AccessToken))
	if err != nil {
		return err
	}
	if res.V2CliRevokeSessionResponseBody == nil {
		return fmt.Errorf("revoke returned %d", res.HTTPMeta.Response.StatusCode)
	}
	return nil
}

func formatUserCode(code string) string {
	fields := strings.Fields(strings.TrimSpace(code))
	compact := strings.ToUpper(strings.Join(fields, ""))
	var b strings.Builder
	for i, r := range compact {
		if i > 0 {
			b.WriteByte(' ')
		}
		b.WriteRune(r)
	}
	return b.String()
}
