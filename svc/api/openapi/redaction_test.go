package openapi_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/unkeyed/unkey/pkg/redaction"
	"github.com/unkeyed/unkey/svc/api/openapi"
)

// The bundled spec is what the running service reads, so an annotation that
// survives in the split sources but is dropped during bundling would silently
// disable redaction for that property. Pin the exact paths.
//
// Adding one is expected when a new secret is annotated. Removing one means a
// value that used to be kept out of ClickHouse now reaches it, so treat a
// deletion here as the change under review.
func TestSpecDeclaresRedactedPaths(t *testing.T) {
	t.Parallel()

	paths, err := redaction.PathsFromSpec(openapi.Spec)
	require.NoError(t, err)

	require.Equal(t, []string{
		"data.dnsRecords[].value", // createDomain response, one entry per record
		"data.key",                // createKey and rerollKey responses
		"data.plaintext",          // getKey response, recoverable key material
		"data.sessionId",          // portal createSession response
		"data.token",              // portal exchangeSession response
		"data.url",                // portal URL, which carries the session token in its query
		"data[].plaintext",        // listKeys response, one entry per key
		"data[].value",            // listEnvironmentVariables response
		"key",                     // verifyKey and whoami requests
		"sessionId",               // portal exchangeSession request
		"variables[].value",       // setEnvironmentVariables request
	}, paths)
}

// Asserts the consequence end to end rather than trusting the path list, using
// the body shapes these operations actually exchange.
func TestRedactorFromSpecStripsKnownSecrets(t *testing.T) {
	t.Parallel()

	paths, err := redaction.PathsFromSpec(openapi.Spec)
	require.NoError(t, err)
	redactor := redaction.New(paths)

	bodies := map[string]string{
		"verifyKey request":         `{"apiId":"api_123","key":"FIXTURE_LEAK"}`,
		"createKey response":        `{"data":{"keyId":"key_123","key":"FIXTURE_LEAK"},"meta":{"requestId":"req_1"}}`,
		"getKey response":           `{"data":{"keyId":"key_123","plaintext":"FIXTURE_LEAK"},"meta":{"requestId":"req_1"}}`,
		"listKeys response":         `{"data":[{"keyId":"key_1","plaintext":"FIXTURE_LEAK"},{"keyId":"key_2"}],"meta":{"requestId":"req_1"}}`,
		"setEnvironmentVariables":   `{"variables":[{"key":"DATABASE_URL","value":"postgresql://user:FIXTURE_LEAK@host/db"}]}`,
		"listEnvironmentVariables":  `{"data":[{"key":"DATABASE_URL","kind":"recoverable","value":"postgresql://user:FIXTURE_LEAK@host/db"}],"meta":{"requestId":"req_1"}}`,
		"portal createSession":      `{"data":{"sessionId":"pst_FIXTURE_LEAK","url":"https://portal.unkey.com/?session=pst_FIXTURE_LEAK"},"meta":{"requestId":"req_1"}}`,
		"portal exchangeSession":    `{"sessionId":"pst_FIXTURE_LEAK"}`,
		"portal session token":      `{"data":{"token":"ps_FIXTURE_LEAK","expiresAt":1711386400000},"meta":{"requestId":"req_1"}}`,
		"truncated env var payload": `{"variables":[{"key":"DATABASE_URL","value":"postgresql://user:FIXTURE_LEAK`,
		"createDomain response":     `{"data":{"domainId":"dom_123","dnsRecords":[{"type":"CNAME","name":"api.acme.com","value":"FIXTURE_LEAK.cname.unkey.com"},{"type":"TXT","name":"_unkey.api.acme.com","value":"unkey-domain-verify=FIXTURE_LEAK"}]},"meta":{"requestId":"req_1"}}`,
	}

	for name, body := range bodies {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			got := string(redactor.Redact([]byte(body)))
			require.NotContains(t, got, "LEAK", "secret survived redaction: %s", got)
		})
	}
}

// What paths buy over matching names: an environment variable's key is its name,
// not a secret, and it stays readable in the logs even though `key` is a secret
// at the root of verifyKey. Matching by name alone redacted both.
func TestRedactorFromSpecKeepsNonSecretsReadable(t *testing.T) {
	t.Parallel()

	paths, err := redaction.PathsFromSpec(openapi.Spec)
	require.NoError(t, err)
	redactor := redaction.New(paths)

	got := string(redactor.Redact([]byte(
		`{"project":"payments","variables":[{"key":"DATABASE_URL","value":"postgresql://user:secret@host/db","kind":"recoverable"}]}`,
	)))

	require.Contains(t, got, `"key":"DATABASE_URL"`, "variable names should stay in the log")
	require.Contains(t, got, `"project":"payments"`)
	require.Contains(t, got, `"kind":"recoverable"`)
	require.NotContains(t, got, "secret@host")
}
