package validation

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

const compatibilitySpec = `openapi: 3.0.3
info:
  title: Validation compatibility
  version: 1.0.0
security:
  - bearer: []
paths:
  /items:
    post:
      requestBody:
        required: true
        content:
          application/json:
            schema:
              $ref: '#/components/schemas/Item'
      responses:
        '200':
          description: OK
  /batches:
    post:
      requestBody:
        required: true
        content:
          application/json:
            schema:
              type: array
              maxItems: 2
              items:
                $ref: '#/components/schemas/Item'
      responses:
        '200':
          description: OK
components:
  securitySchemes:
    bearer:
      type: http
      scheme: bearer
  schemas:
    Item:
      type: object
      required: [name, secret, serverId]
      properties:
        name:
          type: string
          minLength: 1
          maxLength: 10
          pattern: '^[a-z]+$'
        secret:
          type: string
          writeOnly: true
        serverId:
          type: string
          readOnly: true
        credits:
          type: integer
          minimum: 0
        label:
          type: string
          nullable: true
        mode:
          type: string
          default: live
          enum: [live, test]
        metadata:
          type: object
          maxProperties: 0
        child:
          $ref: '#/components/schemas/Item'
`

func TestConcurrentRequestCompatibility(t *testing.T) {
	v, err := NewFromBytes([]byte(compatibilitySpec))
	require.NoError(t, err)

	for _, tc := range []struct {
		name  string
		path  string
		body  string
		valid bool
	}{
		{"omitted read-only required field", "/items", `{"name":"alice","secret":"x"}`, true},
		{"supplied read-only field", "/items", `{"name":"alice","secret":"x","serverId":"id"}`, true},
		{"wrong read-only field type", "/items", `{"name":"alice","secret":"x","serverId":1}`, false},
		{"missing write-only required field", "/items", `{"name":"alice"}`, false},
		{"missing required field", "/items", `{"secret":"x"}`, false},
		{"empty name", "/items", `{"name":"","secret":"x"}`, false},
		{"invalid name pattern", "/items", `{"name":"123","secret":"x"}`, false},
		{"name too long", "/items", `{"name":"abcdefghijkl","secret":"x"}`, false},
		{"zero credits", "/items", `{"name":"alice","secret":"x","credits":0}`, true},
		{"negative credits", "/items", `{"name":"alice","secret":"x","credits":-1}`, false},
		{"fractional credits", "/items", `{"name":"alice","secret":"x","credits":0.5}`, false},
		{"nullable label", "/items", `{"name":"alice","secret":"x","label":null}`, true},
		{"invalid label type", "/items", `{"name":"alice","secret":"x","label":1}`, false},
		{"invalid enum", "/items", `{"name":"alice","secret":"x","mode":"other"}`, false},
		{"empty metadata", "/items", `{"name":"alice","secret":"x","metadata":{}}`, true},
		{"zero property limit", "/items", `{"name":"alice","secret":"x","metadata":{"extra":1}}`, false},
		{"recursive schema", "/items", `{"name":"alice","secret":"x","child":{"name":"bob","secret":"y"}}`, true},
		{"invalid nested type", "/items", `{"name":"alice","secret":"x","child":{"name":1,"secret":"y"}}`, false},
		{"malformed JSON", "/items", `{"name":`, false},
		{"trailing JSON value", "/items", `{"name":"alice","secret":"x"}{}`, false},
		{"trailing garbage", "/items", `{"name":"alice","secret":"x"}garbage`, false},
		{"batch", "/batches", `[{"name":"alice","secret":"x"}]`, true},
		{"batch rejects object", "/batches", `{"name":"alice","secret":"x"}`, false},
		{"item rejects array", "/items", `[{"name":"alice","secret":"x"}]`, false},
		{"batch item required fields", "/batches", `[{"name":"alice"}]`, false},
		{"batch size limit", "/batches", `[{"name":"a","secret":"x"},{"name":"b","secret":"x"},{"name":"c","secret":"x"}]`, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			for range 50 {
				r := httptest.NewRequest(http.MethodPost, tc.path, strings.NewReader(tc.body))
				r.Header.Set("Content-Type", "application/json")
				r.Header.Set("Authorization", "Bearer test")
				result := v.Validate(r)
				require.Equal(t, tc.valid, result == nil, "%+v", result)
				body, err := io.ReadAll(r.Body)
				require.NoError(t, err)
				require.Equal(t, tc.body, string(body))
				require.NoError(t, r.Body.Close())
			}
		})
	}
}

func TestBearerValidationCompatibility(t *testing.T) {
	v, err := NewFromBytes([]byte(compatibilitySpec))
	require.NoError(t, err)
	for _, tc := range []struct {
		name          string
		authorization string
		body          string
		valid         bool
	}{
		{"missing authorization", "", `{"name":"alice","secret":"x"}`, false},
		{"bearer token", "Bearer test", `{"name":"alice","secret":"x"}`, true},
		{"scheme check belongs to authenticator", "Basic test", `{"name":"alice","secret":"x"}`, true},
		{"ignored scheme does not hide body errors", "Basic test", `{"name":1}`, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			r := httptest.NewRequest(http.MethodPost, "/items", strings.NewReader(tc.body))
			r.Header.Set("Content-Type", "application/json")
			if tc.authorization != "" {
				r.Header.Set("Authorization", tc.authorization)
			}
			result := v.Validate(r)
			require.Equal(t, tc.valid, result == nil, "%+v", result)
		})
	}
}
