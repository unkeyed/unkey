package util

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/url"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/sdks/api/go/v3/models/apierrors"
	"github.com/unkeyed/sdks/api/go/v3/models/components"
	"github.com/unkeyed/unkey/pkg/cli"
	"github.com/unkeyed/unkey/pkg/uid"
)

func TestCommandErrorIncludesPublicDiagnosticsWithoutDumpingResponses(t *testing.T) {
	requestID := uid.New(uid.RequestPrefix)
	meta := components.Meta{RequestID: requestID}
	for _, test := range []struct {
		name string
		err  error
		want string
	}{
		{"server", &apierrors.InternalServerErrorResponse{Meta: meta, Error_: components.BaseError{Detail: "Internal error."}}, "Server error: Internal error."},
		{"authentication", &apierrors.UnauthorizedErrorResponse{Meta: meta, Error_: components.BaseError{Detail: "Expired session."}}, "Authentication failed: Expired session."},
		{"not found", &apierrors.NotFoundErrorResponse{Meta: meta, Error_: components.BaseError{Detail: "Resource does not exist."}}, "Not found or not accessible: Resource does not exist."},
		{"permission", &apierrors.ForbiddenErrorResponse{Meta: meta, Error_: components.BaseError{Detail: "Missing permission."}}, "Permission denied: Missing permission."},
		{"rate limit", &apierrors.TooManyRequestsErrorResponse{Meta: meta, Error_: components.BaseError{Detail: "Try after 10 seconds."}}, "Rate limited: Try after 10 seconds."},
	} {
		t.Run(test.name, func(t *testing.T) {
			err := runError(t, fmt.Errorf("request: %w", test.err))
			require.ErrorContains(t, err, "Error: unkey keys create failed.")
			require.ErrorContains(t, err, test.want)
			require.ErrorContains(t, err, "Request ID: "+requestID)
			require.NotContains(t, err.Error(), `"meta"`)
		})
	}
	err := runError(t, &apierrors.APIError{StatusCode: 502, Body: "private upstream HTML"})
	require.ErrorContains(t, err, "API request failed (HTTP 502)")
	require.NotContains(t, err.Error(), "private upstream HTML")
	require.NoError(t, runError(t, nil))
}

func TestCommandErrorAuthenticationHintFollowsCredentialPrecedence(t *testing.T) {
	secret := uid.New(uid.KeyPrefix)
	unauthorized := &apierrors.UnauthorizedErrorResponse{Error_: components.BaseError{Detail: "Invalid credential."}}
	for _, test := range []struct {
		name string
		env  string
		args []string
		want string
	}{
		{"flag before environment", secret, []string{"--root-key=" + secret, "--config=custom.toml"}, "Update the --root-key credential"},
		{"environment before config", secret, []string{"--config=custom.toml"}, "Update UNKEY_ROOT_KEY, or unset it"},
		{"explicit config", "", []string{"--config=custom.toml"}, "Update the credential in --config"},
		{"stored credential", "", nil, "Run 'unkey login' to sign in again, or check your saved root key and API host."},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Setenv("UNKEY_ROOT_KEY", test.env)
			err := runError(t, unauthorized, test.args...)
			require.ErrorContains(t, err, test.want)
			require.NotContains(t, err.Error(), secret)
		})
	}
}

func TestCommandErrorValidationUsesFlagsOrBodyPaths(t *testing.T) {
	problem := &apierrors.BadRequestErrorResponse{Error_: components.BadRequestErrorDetails{
		Detail: "Invalid request.",
		Errors: []components.ValidationError{{Location: "body.apiId", Message: "Invalid identifier.", Fix: new("Use an API ID.")}},
	}}
	err := runError(t, problem)
	require.ErrorContains(t, err, "--api-id: Invalid identifier.")
	require.ErrorContains(t, err, "Use an API ID.")
	body, err := json.Marshal(components.V2KeysCreateKeyRequestBody{APIID: uid.New(uid.APIPrefix)})
	require.NoError(t, err)
	err = runError(t, problem, "--body="+string(body))
	require.ErrorContains(t, err, "body.apiId: Invalid identifier.")
}

func TestCommandErrorJSONRetainsSafeStructuredFields(t *testing.T) {
	requestID := uid.New(uid.RequestPrefix)
	err := runError(t, &apierrors.ForbiddenErrorResponse{
		Meta:   components.Meta{RequestID: requestID},
		Error_: components.BaseError{Detail: "Missing permission.", Status: 403, Type: "urn:example:access-denied"},
	}, "--error-format=json")
	require.Error(t, err)
	var result struct {
		Error struct {
			Message   string `json:"message"`
			Status    int    `json:"status"`
			Type      string `json:"type"`
			RequestID string `json:"requestId"`
			Hint      string `json:"hint"`
		} `json:"error"`
	}
	require.NoError(t, json.Unmarshal([]byte(err.Error()), &result))
	require.Equal(t, "Permission denied: Missing permission.", result.Error.Message)
	require.Equal(t, 403, result.Error.Status)
	require.Equal(t, "urn:example:access-denied", result.Error.Type)
	require.Equal(t, requestID, result.Error.RequestID)
	require.Contains(t, result.Error.Hint, "unkey keys create --help")
}

func TestCommandErrorDoesNotPromiseMutationRetrySafety(t *testing.T) {
	secret := uid.New(uid.KeyPrefix)
	for _, problem := range []error{
		&apierrors.InternalServerErrorResponse{Error_: components.BaseError{Detail: "Internal error."}},
		&url.Error{Op: "Post", URL: "https://example.invalid/?key=" + secret, Err: &net.DNSError{Err: "connection reset"}},
		&apierrors.ResponseValidationError{StatusCode: 400, Body: secret, Cause: fmt.Errorf("invalid response value: %s", secret)},
	} {
		err := runError(t, problem)
		require.ErrorContains(t, err, "The operation may have completed.")
		require.ErrorContains(t, err, "Check the resource state before retrying")
		require.NotContains(t, err.Error(), secret)
	}
}

func runError(t *testing.T, problem error, args ...string) error {
	t.Helper()
	cmd := &cli.Command{
		Name: "test",
		Flags: []cli.Flag{
			cli.String("root-key", "Credential"), cli.String("config", "Configuration"),
			cli.String("api-id", "API ID"), cli.String("body", "JSON body"),
			cli.Enum("error-format", "Error format", []string{"human", "json"}, cli.Default("human")),
		},
		Action: func(context.Context, *cli.Command) error { return problem },
		OnError: func(_ context.Context, cmd *cli.Command, args []string, err error) error {
			return CommandError(cmd, "keys create", err, args...)
		},
	}
	return cmd.Run(t.Context(), append([]string{"test"}, args...))
}
