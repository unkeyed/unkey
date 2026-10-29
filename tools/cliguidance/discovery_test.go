package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

type discoveryDocumentTest struct {
	OpenAPI           string                     `json:"openapi"`
	JSONSchemaDialect string                     `json:"jsonSchemaDialect"`
	Endpoints         map[string]endpointTest    `json:"endpoints"`
	Components        map[string]json.RawMessage `json:"components"`
}

type endpointTest struct {
	Method      string          `json:"method"`
	Path        string          `json:"path"`
	RequestBody json.RawMessage `json:"requestBody"`
	References  []string        `json:"references"`
}

func TestGeneratedDiscoveryCurrent(t *testing.T) {
	output := filepath.Join(t.TempDir(), "schemas.json")
	require.NoError(t, generateMode("../../svc/api/openapi/openapi-generated.yaml", "../../cmd/api/resources.go", output, "discovery"))
	got, err := os.ReadFile(output)
	require.NoError(t, err)
	want, err := os.ReadFile("../../cmd/api/internal/discovery/schemas_generated.json")
	require.NoError(t, err)
	require.Equal(t, string(want), string(got), "run mise exec -- go generate ./cmd/api/internal/discovery")
	var generated discoveryDocumentTest
	require.NoError(t, json.Unmarshal(got, &generated))
	require.Len(t, generated.Endpoints, 100)
	require.Equal(t, len(generated.Endpoints)+len(generated.Components)+9, strings.Count(string(got), "\n"))
	require.Equal(t, "/v3/deployments.createDeployment", generated.Endpoints["deployments create-deployment"].Path)
	require.Equal(t, "3.1.0", generated.OpenAPI)
	require.Equal(t, "https://spec.openapis.org/oas/3.1/dialect/base", generated.JSONSchemaDialect)

	spec, err := os.ReadFile("../../svc/api/openapi/openapi-generated.yaml")
	require.NoError(t, err)
	var source map[string]any
	require.NoError(t, yaml.Unmarshal(spec, &source))
	for ref, component := range generated.Components {
		original, err := resolve(source, ref)
		require.NoError(t, err)
		encoded, err := json.Marshal(original)
		require.NoError(t, err)
		require.Equal(t, string(encoded), string(component), ref)
	}
	keys, err := resourceKeys("../../cmd/api/resources.go")
	require.NoError(t, err)
	selected, err := selectEndpoints(source, keys)
	require.NoError(t, err)
	for key, endpoint := range selected {
		body, err := json.Marshal(endpoint.Operation["requestBody"])
		require.NoError(t, err)
		require.Equal(t, string(body), string(generated.Endpoints[key].RequestBody), key)
	}
}

func TestDiscoveryRetainsSchemaSemanticsAndRecursiveReferences(t *testing.T) {
	const spec = `
openapi: 3.1.0
jsonSchemaDialect: https://json-schema.org/draft/2020-12/schema
components:
  requestBodies:
    Body:
      required: false
      description: Keep this entire annotation. Including this sentence.
      content:
        application/json:
          example: {name: sample}
          schema:
            $ref: '#/components/schemas/Node'
  schemas:
    Unused: {type: string}
    Node:
      $schema: https://json-schema.org/draft/2020-12/schema
      type: object
      additionalProperties: false
      unevaluatedProperties: false
      properties:
        child: {$ref: '#/components/schemas/Node'}
        tuple:
          type: array
          prefixItems: [{type: string}, {const: 4}]
          uniqueItems: true
        number: {type: number, exclusiveMinimum: 0, multipleOf: 0.5}
        escaped: {$ref: '#/components/schemas/a~1b~0c'}
      oneOf:
        - required: [child]
        - required: [tuple]
      anyOf:
        - required: [number]
        - required: [escaped]
      allOf:
        - if: {required: [number]}
          then: {required: [tuple]}
          else: {not: {required: [tuple]}}
      dependentRequired: {child: [number]}
      examples: [{number: 0.5}]
    a/b~c:
      type: string
      pattern: '^[a-z]+$'
      minLength: 1
      maxLength: 32
paths:
  /v2/keys.whoami:
    post:
      requestBody: {$ref: '#/components/requestBodies/Body'}
`
	encoded, err := extractDiscovery([]byte(spec), []string{"keys whoami"})
	require.NoError(t, err)
	var generated discoveryDocumentTest
	require.NoError(t, json.Unmarshal(encoded, &generated))
	require.Equal(t, "https://json-schema.org/draft/2020-12/schema", generated.JSONSchemaDialect)
	require.Len(t, generated.Components, 3)
	require.ElementsMatch(t, []string{"#/components/requestBodies/Body", "#/components/schemas/Node", "#/components/schemas/a~1b~0c"}, generated.Endpoints["keys whoami"].References)
	var source map[string]any
	require.NoError(t, yaml.Unmarshal([]byte(spec), &source))
	for ref, component := range generated.Components {
		original, err := resolve(source, ref)
		require.NoError(t, err)
		want, err := json.Marshal(original)
		require.NoError(t, err)
		require.Equal(t, string(want), string(component), ref)
	}
}

func TestDiscoveryRejectsUnclosedReferences(t *testing.T) {
	for _, ref := range []string{"https://example.com/schema", "#/components/schemas/Missing", "#/paths/missing"} {
		t.Run(ref, func(t *testing.T) {
			document := map[string]any{
				"openapi": "3.1.0",
				"paths": map[string]any{"/v2/keys.whoami": map[string]any{"post": map[string]any{
					"requestBody": map[string]string{"$ref": ref},
				}}},
			}
			spec, err := yaml.Marshal(document)
			require.NoError(t, err)
			_, err = extractDiscovery(spec, []string{"keys whoami"})
			require.ErrorContains(t, err, ref)
		})
	}
}
