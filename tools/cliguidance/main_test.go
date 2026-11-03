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

type schemaContract struct {
	Type         string                     `json:"type,omitempty"`
	Description  string                     `json:"description,omitempty"`
	Nullable     bool                       `json:"nullable,omitempty"`
	Secret       bool                       `json:"secret,omitempty"`
	Enum         []string                   `json:"enum,omitempty"`
	Properties   map[string]*schemaContract `json:"properties,omitempty"`
	Required     []string                   `json:"required,omitempty"`
	Items        *schemaContract            `json:"items,omitempty"`
	Alternatives []*schemaContract          `json:"alternatives,omitempty"`
	Additional   *schemaContract            `json:"additional,omitempty"`
	Minimum      *float64                   `json:"minimum,omitempty"`
	Maximum      *float64                   `json:"maximum,omitempty"`
	MinLength    int                        `json:"minLength,omitempty"`
	MaxLength    int                        `json:"maxLength,omitempty"`
	Pattern      string                     `json:"pattern,omitempty"`
	MinItems     int                        `json:"minItems,omitempty"`
	MaxItems     int                        `json:"maxItems,omitempty"`
	Default      json.RawMessage            `json:"default,omitempty"`
}

func TestGeneratedSchemasCurrent(t *testing.T) {
	output := filepath.Join(t.TempDir(), "schemas.json")
	require.NoError(t, generate("../../svc/api/openapi/openapi-generated.yaml", "../../cmd/api/resources.go", output))
	got, err := os.ReadFile(output)
	require.NoError(t, err)
	want, err := os.ReadFile("../../cmd/api/internal/guidance/schemas_generated.json")
	require.NoError(t, err)
	require.Equal(t, string(want), string(got), "run mise exec -- go generate ./cmd/api/internal/guidance")

	schemas := decodeSchemas(t, got)
	keys, err := resourceKeys("../../cmd/api/resources.go")
	require.NoError(t, err)
	require.Len(t, schemas, len(keys))
	require.Equal(t, len(keys)+2, strings.Count(string(got), "\n"))
	for _, key := range keys {
		require.Contains(t, schemas, key)
	}
	for _, key := range []string{"github install-app", "workspace get-flags", "workspace get-limits"} {
		require.Equal(t, &schemaContract{Type: "object"}, schemas[key])
	}
	require.Contains(t, schemas, "keys whoami")
	require.Contains(t, schemas, "root-keys create-key")
	require.NotContains(t, schemas, "cli exchange-token")
	require.Equal(t, []string{"name", "slug"}, schemas["projects create-project"].Required)
	require.Equal(t, "any", schemas["keys create-key"].Properties["meta"].Additional.Type)
	require.Equal(t, []string{"git"}, schemas["apps create-app"].Alternatives[0].Required)
	require.Equal(t, []string{"oci"}, schemas["apps create-app"].Alternatives[1].Required)
	require.True(t, schemas["apps update-app"].Properties["git"].Nullable)
	require.Equal(t, "object", schemas["apps update-app"].Properties["git"].Type)
	require.True(t, schemas["environments update-settings"].Properties["dockerfile"].Nullable)
	require.Equal(t, "array", schemas["ratelimit multi-limit"].Type)
	require.Equal(t, "object", schemas["ratelimit multi-limit"].Items.Type)
	require.True(t, schemas["keys whoami"].Properties["key"].Secret)
	require.True(t, schemas["environments set-environment-variables"].Properties["variables"].Items.Properties["value"].Secret)
}

func TestDeploymentGuidanceMatchesV3WithOptionalSource(t *testing.T) {
	spec, err := os.ReadFile("../../svc/api/openapi/openapi-generated.yaml")
	require.NoError(t, err)
	data, err := extract(spec, []string{"deployments create-deployment"})
	require.NoError(t, err)
	schema := decodeSchemas(t, data)["deployments create-deployment"]
	require.Contains(t, schema.Properties, "oci")
	require.NotContains(t, schema.Properties, "image")
	require.Equal(t, []string{"project", "app", "environment"}, schema.Required)
	require.Empty(t, schema.Alternatives, "omitting the source must use the app default")
	for _, source := range []string{"git", "oci", "deployment"} {
		require.Contains(t, schema.Properties, source)
		require.Equal(t, "object", schema.Properties[source].Type)
	}
	require.Equal(t, []string{"image"}, schema.Properties["oci"].Required)
}

func TestEndpointVersionsAreExplicit(t *testing.T) {
	const spec = `
paths:
  /v2/keys.whoami:
    post: {}
  /v3/keys.whoami:
    post:
      requestBody:
        content:
          application/json:
            schema: {type: string}
  /v2/deployments.createDeployment:
    post: {}
`
	t.Run("newer endpoints do not replace v2 implicitly", func(t *testing.T) {
		data, err := extract([]byte(spec), []string{"keys whoami"})
		require.NoError(t, err)
		require.Equal(t, &schemaContract{Type: "object"}, decodeSchemas(t, data)["keys whoami"])
	})
	t.Run("missing explicit version does not fall back", func(t *testing.T) {
		_, err := extract([]byte(spec), []string{"deployments create-deployment"})
		require.ErrorContains(t, err, "deployments create-deployment")
	})
}

func TestRequestMetadataContract(t *testing.T) {
	const spec = `
components:
  schemas:
    Text:
      type: string
      minLength: 1
      maxLength: 16
      pattern: '^[a-z]+$'
      x-unkey-redact: true
      description: Original description. More details.
    Choice:
      required: [kind]
      properties:
        kind:
          type: string
          enum: [git, oci]
          writeOnly: true
paths:
  /v2/rootKeys.createKey:
    post:
      requestBody:
        content:
          application/json:
            schema:
              required: [text, variants]
              properties:
                text:
                  $ref: '#/components/schemas/Text'
                  description: Replacement description. Details omitted.
                  default: hello
                wrapped:
                  allOf:
                    - $ref: '#/components/schemas/Text'
                  description: Wrapper description.
                nullable:
                  type: [string, 'null']
                  minLength: 1
                  maxLength: 5
                  default: null
                number:
                  type: number
                  minimum: 0
                  maximum: 12.5
                  default: 0
                enabled:
                  type: boolean
                  default: false
                choice:
                  anyOf:
                    - $ref: '#/components/schemas/Choice'
                    - type: 'null'
                variants:
                  oneOf:
                    - properties:
                        kind: {type: string, enum: [first]}
                    - properties:
                        kind: {type: string, enum: [second]}
                labels:
                  type: object
                  additionalProperties:
                    $ref: '#/components/schemas/Text'
                list:
                  type: array
                  minItems: 1
                  maxItems: 4
                  items:
                    type: object
                    additionalProperties: {}
                payload: {}
                metadata:
                  type: object
                  additionalProperties: true
                flags:
                  type: array
                  items: {type: boolean}
                  default: [true, false]
  /v2/keys.whoami:
    post: {}
  /v2/ignored.operation:
    post:
      requestBody:
        content:
          text/plain: {}
`
	data, err := extract([]byte(spec), []string{"root-keys create-key", "keys whoami"})
	require.NoError(t, err)
	schemas := decodeSchemas(t, data)
	require.Len(t, schemas, 2)
	require.Equal(t, &schemaContract{Type: "object"}, schemas["keys whoami"])
	root := schemas["root-keys create-key"]
	require.Equal(t, "object", root.Type)
	require.Equal(t, []string{"text", "variants"}, root.Required)
	fields := root.Properties
	require.Equal(t, "Replacement description.", fields["text"].Description)
	require.Equal(t, "Wrapper description.", fields["wrapped"].Description)
	require.Equal(t, "string", fields["wrapped"].Type)
	require.Equal(t, 1, fields["text"].MinLength)
	require.Equal(t, 16, fields["text"].MaxLength)
	require.Equal(t, "^[a-z]+$", fields["text"].Pattern)
	require.True(t, fields["text"].Secret)
	require.True(t, fields["wrapped"].Secret)
	require.True(t, fields["choice"].Properties["kind"].Secret)
	require.True(t, fields["labels"].Additional.Secret)
	require.False(t, fields["enabled"].Secret)
	require.True(t, fields["nullable"].Nullable)
	require.Equal(t, "string", fields["nullable"].Type)
	require.Equal(t, 5, fields["nullable"].MaxLength)
	require.NotNil(t, fields["number"].Minimum)
	require.Equal(t, 0.0, *fields["number"].Minimum)
	require.NotNil(t, fields["number"].Maximum)
	require.Equal(t, 12.5, *fields["number"].Maximum)
	require.True(t, fields["choice"].Nullable)
	require.Equal(t, "object", fields["choice"].Type)
	require.Equal(t, []string{"kind"}, fields["choice"].Required)
	require.Equal(t, []string{"git", "oci"}, fields["choice"].Properties["kind"].Enum)
	require.Len(t, fields["variants"].Alternatives, 2)
	require.Equal(t, []string{"first"}, fields["variants"].Alternatives[0].Properties["kind"].Enum)
	require.Equal(t, []string{"second"}, fields["variants"].Alternatives[1].Properties["kind"].Enum)
	require.Equal(t, "string", fields["labels"].Additional.Type)
	require.Equal(t, 16, fields["labels"].Additional.MaxLength)
	require.Equal(t, 1, fields["list"].MinItems)
	require.Equal(t, 4, fields["list"].MaxItems)
	require.Equal(t, "object", fields["list"].Items.Type)
	require.Equal(t, "any", fields["list"].Items.Additional.Type)
	require.Equal(t, "any", fields["payload"].Type)
	require.Equal(t, "any", fields["metadata"].Additional.Type)
	for name, expected := range map[string]any{"text": "hello", "nullable": nil, "number": float64(0), "enabled": false, "flags": []any{true, false}} {
		t.Run("default_"+name, func(t *testing.T) {
			require.NotEmpty(t, fields[name].Default)
			var actual any
			require.NoError(t, json.Unmarshal(fields[name].Default, &actual))
			require.Equal(t, expected, actual)
		})
	}
}

func TestClosedObjectsDoNotBecomeArbitraryMaps(t *testing.T) {
	for name, schema := range map[string]map[string]any{
		"reference": {"$ref": "#/components/schemas/Closed"},
		"nullable":  {"type": []any{"object", "null"}, "additionalProperties": false},
		"allOf":     {"allOf": []any{map[string]any{"$ref": "#/components/schemas/Closed"}}},
	} {
		t.Run(name, func(t *testing.T) {
			document := map[string]any{
				"components": map[string]any{"schemas": map[string]any{"Closed": map[string]any{"type": "object", "additionalProperties": false}}},
				"paths": map[string]any{"/v2/keys.whoami": map[string]any{"post": map[string]any{
					"requestBody": map[string]any{"content": map[string]any{"application/json": map[string]any{"schema": schema}}},
				}}},
			}
			spec, err := yaml.Marshal(document)
			require.NoError(t, err)
			data, err := extract(spec, []string{"keys whoami"})
			require.NoError(t, err)
			result := decodeSchemas(t, data)["keys whoami"]
			require.Equal(t, "object", result.Type)
			require.Nil(t, result.Additional)
			require.Nil(t, result.Properties)
		})
	}
}

func TestGenerationRejectsMissingCanonicalCommand(t *testing.T) {
	spec, err := yaml.Marshal(map[string]any{"paths": map[string]any{"/v2/keys.whoami": map[string]any{"post": map[string]any{}}}})
	require.NoError(t, err)
	_, err = extract(spec, []string{"keys whoami", "projects create-project"})
	require.ErrorContains(t, err, "projects create-project")
}

func TestGenerationRejectsUnsupportedSchemas(t *testing.T) {
	for name, schema := range map[string]map[string]any{
		"external reference": {"$ref": "https://example.com/schema"},
		"missing reference":  {"$ref": "#/components/schemas/Missing"},
		"recursive reference": {
			"type":       "object",
			"properties": map[string]any{"child": map[string]any{"$ref": "#/components/schemas/Request"}},
		},
		"multiple allOf members": {"allOf": []any{map[string]any{"type": "string"}, map[string]any{"minLength": 1}}},
		"conditional schema":     {"if": map[string]any{"type": "string"}},
	} {
		t.Run(name, func(t *testing.T) {
			document := map[string]any{
				"components": map[string]any{"schemas": map[string]any{"Request": schema}},
				"paths": map[string]any{"/v2/keys.whoami": map[string]any{"post": map[string]any{
					"requestBody": map[string]any{"content": map[string]any{"application/json": map[string]any{"schema": map[string]any{"$ref": "#/components/schemas/Request"}}}},
				}}},
			}
			spec, err := yaml.Marshal(document)
			require.NoError(t, err)
			_, err = extract(spec, []string{"keys whoami"})
			require.Error(t, err)
		})
	}
}

func TestDescriptionsAreShortHelpers(t *testing.T) {
	for _, description := range []string{
		"First paragraph.\n\nDo not include the next paragraph.",
		"First line wraps\nonto the next line. More sentences are omitted.",
		strings.Repeat("multi-byte text éè ", 30),
	} {
		t.Run(description[:15], func(t *testing.T) {
			short := helper(description)
			require.LessOrEqual(t, len([]rune(short)), 180)
			require.NotContains(t, short, "\n")
			require.NotContains(t, short, "Do not include")
			require.NotContains(t, short, "More sentences")
		})
	}
}

func decodeSchemas(t *testing.T, data []byte) map[string]*schemaContract {
	t.Helper()
	var schemas map[string]*schemaContract
	decoder := json.NewDecoder(strings.NewReader(string(data)))
	decoder.DisallowUnknownFields()
	require.NoError(t, decoder.Decode(&schemas))
	return schemas
}
