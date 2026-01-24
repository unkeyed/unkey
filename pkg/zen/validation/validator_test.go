package validation

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNew(t *testing.T) {
	v, err := New()
	require.NoError(t, err)
	require.NotNil(t, v)
	require.NotNil(t, v.matcher)
	require.NotNil(t, v.compiler)
	require.NotNil(t, v.securitySchemes)
}

func TestValidate_ValidRequest(t *testing.T) {
	v, err := New()
	require.NoError(t, err)

	// Test with a valid keys.setRoles request
	body := `{"keyId": "key_123abc", "roles": ["admin", "user"]}`
	req := httptest.NewRequest(http.MethodPost, "/v2/keys.setRoles", bytes.NewReader([]byte(body)))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer test_key")

	resp, valid := v.Validate(context.Background(), req)
	require.True(t, valid, "expected valid request, got errors: %+v", resp)
}

func TestValidate_InvalidRequest_MissingRequired(t *testing.T) {
	v, err := New()
	require.NoError(t, err)

	// Missing required 'keyId' field
	body := `{"roles": ["admin"]}`
	req := httptest.NewRequest(http.MethodPost, "/v2/keys.setRoles", bytes.NewReader([]byte(body)))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer test_key")

	resp, valid := v.Validate(context.Background(), req)
	require.False(t, valid, "expected invalid request")
	badReq, ok := resp.(*BadRequestError)
	require.True(t, ok, "expected BadRequestError")
	require.Equal(t, "Bad Request", badReq.Error.Title)
	require.NotEmpty(t, badReq.Error.Errors)
}

func TestValidate_InvalidRequest_WrongType(t *testing.T) {
	v, err := New()
	require.NoError(t, err)

	// keyId should be string, not number
	body := `{"keyId": 123, "roles": ["admin"]}`
	req := httptest.NewRequest(http.MethodPost, "/v2/keys.setRoles", bytes.NewReader([]byte(body)))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer test_key")

	resp, valid := v.Validate(context.Background(), req)
	require.False(t, valid, "expected invalid request")
	badReq, ok := resp.(*BadRequestError)
	require.True(t, ok, "expected BadRequestError")
	require.Equal(t, "Bad Request", badReq.Error.Title)
	require.NotEmpty(t, badReq.Error.Errors)
}

func TestValidate_InvalidJSON(t *testing.T) {
	v, err := New()
	require.NoError(t, err)

	body := `{invalid json}`
	req := httptest.NewRequest(http.MethodPost, "/v2/keys.setRoles", bytes.NewReader([]byte(body)))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer test_key")

	resp, valid := v.Validate(context.Background(), req)
	require.False(t, valid, "expected invalid request")
	badReq, ok := resp.(*BadRequestError)
	require.True(t, ok, "expected BadRequestError")
	require.Equal(t, "Bad Request", badReq.Error.Title)
	require.Contains(t, badReq.Error.Detail, "Invalid JSON")
}

func TestValidate_UnknownPath_PassThrough(t *testing.T) {
	v, err := New()
	require.NoError(t, err)

	// Unknown path should pass through (let router handle 404)
	body := `{"foo": "bar"}`
	req := httptest.NewRequest(http.MethodPost, "/unknown/path", bytes.NewReader([]byte(body)))
	req.Header.Set("Content-Type", "application/json")

	_, valid := v.Validate(context.Background(), req)
	require.True(t, valid, "unknown path should pass through")
}

func TestValidate_EmptyBody_RequiredBodyFails(t *testing.T) {
	v, err := New()
	require.NoError(t, err)

	// keys.setRoles has required: true on requestBody
	req := httptest.NewRequest(http.MethodPost, "/v2/keys.setRoles", nil)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer test_key")

	resp, valid := v.Validate(context.Background(), req)
	require.False(t, valid, "empty body should fail for required request body")
	badReq, ok := resp.(*BadRequestError)
	require.True(t, ok, "expected BadRequestError")
	require.Contains(t, badReq.Error.Detail, "required")
}

func TestValidate_UnknownPath_EmptyBodyPassThrough(t *testing.T) {
	v, err := New()
	require.NoError(t, err)

	// Unknown paths should pass through regardless of body
	req := httptest.NewRequest(http.MethodPost, "/unknown/path", nil)
	req.Header.Set("Content-Type", "application/json")

	_, valid := v.Validate(context.Background(), req)
	require.True(t, valid, "unknown path should pass through")
}

func TestValidate_BodyResetForDownstream(t *testing.T) {
	v, err := New()
	require.NoError(t, err)

	body := `{"keyId": "key_123abc", "roles": ["admin"]}`
	req := httptest.NewRequest(http.MethodPost, "/v2/keys.setRoles", bytes.NewReader([]byte(body)))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer test_key")

	_, valid := v.Validate(context.Background(), req)
	require.True(t, valid)

	// Body should be readable again after validation
	readBody, err := io.ReadAll(req.Body)
	require.NoError(t, err)
	require.Equal(t, body, string(readBody))
}

func TestValidate_AdditionalProperties(t *testing.T) {
	v, err := New()
	require.NoError(t, err)

	// V2KeysSetRolesRequestBody has additionalProperties: false
	body := `{"keyId": "key_123abc", "roles": ["admin"], "unknownField": "value"}`
	req := httptest.NewRequest(http.MethodPost, "/v2/keys.setRoles", bytes.NewReader([]byte(body)))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer test_key")

	resp, valid := v.Validate(context.Background(), req)
	require.False(t, valid, "expected invalid request due to additional properties")
	badReq, ok := resp.(*BadRequestError)
	require.True(t, ok, "expected BadRequestError")
	require.Equal(t, "Bad Request", badReq.Error.Title)
}

func TestValidate_MissingAuthorizationHeader(t *testing.T) {
	v, err := New()
	require.NoError(t, err)

	body := `{"keyId": "key_123abc", "roles": ["admin"]}`
	req := httptest.NewRequest(http.MethodPost, "/v2/keys.setRoles", bytes.NewReader([]byte(body)))
	req.Header.Set("Content-Type", "application/json")
	// No Authorization header

	resp, valid := v.Validate(context.Background(), req)
	require.False(t, valid, "expected invalid request due to missing auth")
	unauthResp, ok := resp.(*UnauthorizedError)
	require.True(t, ok, "expected UnauthorizedError")
	require.Equal(t, "Unauthorized", unauthResp.Error.Title)
	require.Equal(t, http.StatusUnauthorized, unauthResp.Error.Status)
	require.Contains(t, unauthResp.Error.Detail, "Authorization header")
	require.Equal(t, "https://unkey.com/docs/errors/unkey/authentication/missing", unauthResp.Error.Type)
}

func TestValidate_MalformedAuthorizationHeader(t *testing.T) {
	v, err := New()
	require.NoError(t, err)

	body := `{"keyId": "key_123abc", "roles": ["admin"]}`
	req := httptest.NewRequest(http.MethodPost, "/v2/keys.setRoles", bytes.NewReader([]byte(body)))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Basic abc123") // Wrong scheme

	resp, valid := v.Validate(context.Background(), req)
	require.False(t, valid, "expected invalid request due to wrong auth scheme")
	unauthResp, ok := resp.(*UnauthorizedError)
	require.True(t, ok, "expected UnauthorizedError")
	require.Equal(t, "Unauthorized", unauthResp.Error.Title)
	require.Equal(t, http.StatusUnauthorized, unauthResp.Error.Status)
	require.Contains(t, unauthResp.Error.Detail, "Bearer")
	require.Equal(t, "https://unkey.com/docs/errors/unkey/authentication/malformed", unauthResp.Error.Type)
}

func TestValidate_EmptyBearerToken(t *testing.T) {
	v, err := New()
	require.NoError(t, err)

	body := `{"keyId": "key_123abc", "roles": ["admin"]}`
	req := httptest.NewRequest(http.MethodPost, "/v2/keys.setRoles", bytes.NewReader([]byte(body)))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer ")

	resp, valid := v.Validate(context.Background(), req)
	require.False(t, valid, "expected invalid request due to empty token")
	unauthResp, ok := resp.(*UnauthorizedError)
	require.True(t, ok, "expected UnauthorizedError")
	require.Equal(t, "Unauthorized", unauthResp.Error.Title)
	require.Equal(t, http.StatusUnauthorized, unauthResp.Error.Status)
	require.Equal(t, "https://unkey.com/docs/errors/unkey/authentication/malformed", unauthResp.Error.Type)
}

func TestValidate_ContentTypeWithCharset(t *testing.T) {
	v, err := New()
	require.NoError(t, err)

	body := `{"keyId": "key_123abc", "roles": ["admin"]}`
	req := httptest.NewRequest(http.MethodPost, "/v2/keys.setRoles", bytes.NewReader([]byte(body)))
	req.Header.Set("Content-Type", "application/json; charset=utf-8")
	req.Header.Set("Authorization", "Bearer test_key")

	resp, valid := v.Validate(context.Background(), req)
	require.True(t, valid, "expected valid request with charset, got errors: %+v", resp)
}
