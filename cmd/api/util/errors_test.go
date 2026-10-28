package util

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/sdks/api/go/v3/models/apierrors"
	"github.com/unkeyed/sdks/api/go/v3/models/components"
	"github.com/unkeyed/unkey/pkg/uid"
)

func TestFormatError_NilError(t *testing.T) {
	result := FormatError(nil)
	require.Equal(t, "", result)
}

func TestFormatError_GenericError(t *testing.T) {
	err := fmt.Errorf("something went wrong")
	result := FormatError(err)
	require.Equal(t, "something went wrong", result)
}

func TestFormatErrorIncludesSupportDiagnosticsWithoutDumpingResponses(t *testing.T) {
	requestID := uid.New(uid.RequestPrefix)
	meta := components.Meta{RequestID: requestID}
	for _, test := range []struct {
		name string
		err  error
		want string
	}{
		{"server", &apierrors.InternalServerErrorResponse{Meta: meta, Error_: components.BaseError{Detail: "Try again later."}}, "Server error: Try again later."},
		{"authentication", &apierrors.UnauthorizedErrorResponse{Meta: meta, Error_: components.BaseError{Detail: "Expired session."}}, "run 'unkey login'"},
		{"permission", &apierrors.ForbiddenErrorResponse{Meta: meta, Error_: components.BaseError{Detail: "Missing project permission."}}, "Permission denied: Missing project permission."},
		{"rate limit", &apierrors.TooManyRequestsErrorResponse{Meta: meta, Error_: components.BaseError{Detail: "Try after 10 seconds."}}, "Rate limited: Try after 10 seconds."},
	} {
		t.Run(test.name, func(t *testing.T) {
			output := FormatError(fmt.Errorf("request: %w", test.err))
			require.Contains(t, output, test.want)
			require.Contains(t, output, "Request ID: "+requestID)
			require.NotContains(t, output, `"meta"`)
		})
	}
	output := FormatError(&apierrors.APIError{StatusCode: 502, Body: "upstream HTML with private diagnostics"})
	require.Equal(t, "API request failed (HTTP 502)", output)
}
