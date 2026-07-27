package redaction

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// A spec shaped like the real one: a request that sends variables at the root, a
// response that nests the same schema under data, and a second operation with an
// annotated property declared inline.
const envVarSpec = `
openapi: 3.0.3
info:
  title: test
  version: 1.0.0
paths:
  /v2/environments.setEnvironmentVariables:
    post:
      requestBody:
        content:
          application/json:
            schema:
              type: object
              properties:
                variables:
                  type: array
                  items:
                    "$ref": "#/components/schemas/EnvironmentVariableInput"
      responses:
        "200":
          content:
            application/json:
              schema:
                type: object
                properties:
                  data:
                    type: object
                    properties:
                      variables:
                        type: array
                        items:
                          "$ref": "#/components/schemas/EnvironmentVariable"
                  meta:
                    type: object
  /v2/keys.verifyKey:
    post:
      requestBody:
        content:
          application/json:
            schema:
              type: object
              properties:
                apiId:
                  type: string
                key:
                  type: string
                  x-unkey-redact: true
components:
  schemas:
    EnvironmentVariableInput:
      type: object
      properties:
        key:
          type: string
        value:
          type: string
          x-unkey-redact: true
    EnvironmentVariable:
      type: object
      properties:
        key:
          type: string
        value:
          type: string
          x-unkey-redact: true
`

func TestPathsFromSpec(t *testing.T) {
	t.Parallel()

	paths, err := PathsFromSpec([]byte(envVarSpec))
	require.NoError(t, err)

	// One annotation on EnvironmentVariable(.Input).value yields two paths,
	// because the request and the response reach it differently. The variable
	// key stays out: it is a name, not a secret.
	require.Equal(t, []string{
		"data.variables[].value",
		"key",
		"variables[].value",
	}, paths)
}

// End to end through the matcher: the paths the walker emits must actually redact
// the bodies those operations exchange, and leave the neighbouring key alone.
func TestPathsFromSpec_RedactsTheBodiesTheyDescribe(t *testing.T) {
	t.Parallel()

	paths, err := PathsFromSpec([]byte(envVarSpec))
	require.NoError(t, err)
	r := New(paths)

	require.Equal(t,
		`{"variables":[{"key":"DATABASE_URL","value":"[REDACTED]"}]}`,
		string(r.Redact([]byte(`{"variables":[{"key":"DATABASE_URL","value":"postgres://secret"}]}`))),
	)
	require.Equal(t,
		`{"data":{"variables":[{"key":"DATABASE_URL","value":"[REDACTED]"}]},"meta":{"requestId":"req_1"}}`,
		string(r.Redact([]byte(`{"data":{"variables":[{"key":"DATABASE_URL","value":"postgres://secret"}]},"meta":{"requestId":"req_1"}}`))),
	)
	require.Equal(t,
		`{"apiId":"api_123","key":"[REDACTED]"}`,
		string(r.Redact([]byte(`{"apiId":"api_123","key":"secret"}`))),
	)
}

func TestPathsFromSpec_CompositionAndNesting(t *testing.T) {
	t.Parallel()

	spec := []byte(`
paths:
  /v2/thing:
    post:
      requestBody:
        content:
          application/json:
            schema:
              type: object
              properties:
                wrapper:
                  type: object
                  properties:
                    token:
                      type: string
                      x-unkey-redact: true
                combined:
                  allOf:
                    - type: object
                      properties:
                        plaintext:
                          type: string
                          x-unkey-redact: true
                matrix:
                  type: array
                  items:
                    type: array
                    items:
                      type: object
                      properties:
                        secret:
                          type: string
                          x-unkey-redact: true
`)

	paths, err := PathsFromSpec(spec)
	require.NoError(t, err)
	require.Equal(t, []string{
		"combined.plaintext",
		"matrix[][].secret",
		"wrapper.token",
	}, paths)
}

// A recursive schema puts the annotated property at every depth, which no finite
// path set covers. Emitting the shallow paths and missing the deep ones would be
// a silent leak, so this is refused outright and the service fails to start.
func TestPathsFromSpec_SelfReferentialSchemaIsRefused(t *testing.T) {
	t.Parallel()

	spec := []byte(`
paths:
  /v2/tree:
    post:
      requestBody:
        content:
          application/json:
            schema:
              "$ref": "#/components/schemas/Node"
components:
  schemas:
    Node:
      type: object
      properties:
        secret:
          type: string
          x-unkey-redact: true
        child:
          "$ref": "#/components/schemas/Node"
`)

	_, err := PathsFromSpec(spec)
	require.ErrorContains(t, err, "recursive schema")
	require.ErrorContains(t, err, "Node")
}

// A component no operation references cannot appear in any body, so it yields no
// path. Annotating an unused schema is a no-op rather than a global rule.
func TestPathsFromSpec_UnreachableComponentIsIgnored(t *testing.T) {
	t.Parallel()

	spec := []byte(`
components:
  schemas:
    Orphan:
      type: object
      properties:
        secret:
          type: string
          x-unkey-redact: true
`)

	paths, err := PathsFromSpec(spec)
	require.NoError(t, err)
	require.Empty(t, paths)
}

func TestPathsFromSpec_IgnoresExamples(t *testing.T) {
	t.Parallel()

	// An example payload can contain anything, including something shaped like a
	// schema. Treating one as a schema would invent paths out of request data.
	spec := []byte(`
paths:
  /v2/thing:
    post:
      requestBody:
        content:
          application/json:
            schema:
              type: object
              properties:
                name:
                  type: string
            examples:
              sample:
                value:
                  properties:
                    other:
                      x-unkey-redact: true
`)

	paths, err := PathsFromSpec(spec)
	require.NoError(t, err)
	require.Empty(t, paths)
}

func TestPathsFromSpec_IgnoresNonBooleanAnnotation(t *testing.T) {
	t.Parallel()

	spec := []byte(`
paths:
  /v2/thing:
    post:
      requestBody:
        content:
          application/json:
            schema:
              type: object
              properties:
                a:
                  x-unkey-redact: "true"
                b:
                  x-unkey-redact: yes
                c:
                  x-unkey-redact: false
                d:
                  x-unkey-redact: true
`)

	paths, err := PathsFromSpec(spec)
	require.NoError(t, err)
	// Only a real boolean true counts. The quoted string, YAML 1.2's plain `yes`
	// (a string, not a bool), and the explicit false are all ignored, so a typo
	// shows up as a missing path in the pinned list instead of silent redaction.
	require.Equal(t, []string{"d"}, paths)
}

func TestPathsFromSpec_Malformed(t *testing.T) {
	t.Parallel()

	_, err := PathsFromSpec([]byte("\tnot: [valid: yaml"))
	require.Error(t, err)
}

func TestPathsFromSpec_EmptySpec(t *testing.T) {
	t.Parallel()

	paths, err := PathsFromSpec([]byte(``))
	require.NoError(t, err)
	require.Empty(t, paths)
}

// Found by an adversarial review. A body that is itself an array had its items
// skipped, so /v2/ratelimit.multiLimit would stay unredacted while its
// object-bodied twin /v2/ratelimit.limit was covered by the same schema.
func TestPathsFromSpec_RootArrayBody(t *testing.T) {
	t.Parallel()

	spec := []byte(`
paths:
  /v2/ratelimit.limit:
    post:
      requestBody:
        content:
          application/json:
            schema:
              "$ref": "#/components/schemas/Limit"
  /v2/ratelimit.multiLimit:
    post:
      requestBody:
        content:
          application/json:
            schema:
              type: array
              items:
                "$ref": "#/components/schemas/Limit"
components:
  schemas:
    Limit:
      type: object
      properties:
        identifier:
          type: string
          x-unkey-redact: true
`)

	paths, err := PathsFromSpec(spec)
	require.NoError(t, err)
	require.Equal(t, []string{"[].identifier", "identifier"}, paths)

	r := New(paths)
	require.Equal(t,
		`[{"identifier":"[REDACTED]"},{"identifier":"[REDACTED]"}]`,
		string(r.Redact([]byte(`[{"identifier":"user@example.com"},{"identifier":"1.2.3.4"}]`))),
	)
}

// Found by an adversarial review. An annotation only means something on a property,
// because only a property has a name to build a path from. Annotating an items
// schema or a body root used to be silently dropped, which is the exact failure
// this mechanism exists to prevent: an author marks a secret and nothing happens.
func TestPathsFromSpec_UnaddressableAnnotationIsRefused(t *testing.T) {
	t.Parallel()

	onItems := []byte(`
paths:
  /v2/thing:
    post:
      requestBody:
        content:
          application/json:
            schema:
              type: object
              properties:
                tokens:
                  type: array
                  items:
                    type: string
                    x-unkey-redact: true
`)

	_, err := PathsFromSpec(onItems)
	require.ErrorContains(t, err, "only a property can carry x-unkey-redact")
	require.ErrorContains(t, err, "tokens[] elements")

	onRoot := []byte(`
paths:
  /v2/thing:
    post:
      requestBody:
        content:
          application/json:
            schema:
              type: object
              x-unkey-redact: true
              properties:
                a: {type: string}
`)

	_, err = PathsFromSpec(onRoot)
	require.ErrorContains(t, err, "a request body root")
}
