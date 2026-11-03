package tools

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"slices"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"

	"github.com/unkeyed/unkey/svc/api/openapi"
)

var update = flag.Bool("update", false, "rewrite testdata/toolsets.golden")

const goldenPath = "testdata/toolsets.golden"

// TestToolsetsGolden pins the public tool surface of the embedded spec.
// Adding, removing, renaming or moving a tool between toolsets fails until
// the golden file is regenerated, so the change shows up as a reviewed diff:
//
//	mise exec -- go test ./svc/mcp/internal/tools -run TestToolsetsGolden -update
func TestToolsetsGolden(t *testing.T) {
	registry := loadEmbedded(t)

	var lines []string
	for _, toolset := range Toolsets {
		for _, tool := range registry.Tools(toolset.ID) {
			lines = append(lines, fmt.Sprintf("%s %s %s", toolset.ID, tool.Name, tool.OperationID))
		}
	}
	got := strings.Join(lines, "\n") + "\n"

	if *update {
		require.NoError(t, os.WriteFile(goldenPath, []byte(got), 0o600))
	}
	want, err := os.ReadFile(goldenPath)
	require.NoError(t, err)
	require.Equal(t, string(want), got)
}

// TestToolNameMatchesSpeakeasy guarantees that every tool Speakeasy published
// keeps its exact name, including operations that no longer exist, because
// users have these names in client permission allowlists.
func TestToolNameMatchesSpeakeasy(t *testing.T) {
	published := readSpeakeasy(t)
	require.Len(t, published, 153)
	for _, tool := range published {
		require.Equal(t, tool.name, toolName(tool.operationID), tool.operationID)
	}
}

// TestToolsetsMatchSpeakeasy guarantees that the self-hosted toolsets serve
// every Speakeasy tool except the ones removed on purpose. Each removal is
// listed with its reason; anything else missing or added fails.
func TestToolsetsMatchSpeakeasy(t *testing.T) {
	removed := map[string]string{
		"unkey_v2_liveness":                 "unauthenticated GET health check",
		"unkey_v2_cli_approve_device_login": "operation removed from the API",
		"unkey_v2_cli_deny_device_login":    "operation removed from the API",
		"unkey_v2_cli_poll_device_login":    "operation removed from the API",
		"unkey_v2_cli_refresh_token":        "operation removed from the API",
		"unkey_v2_cli_get_device_login":     "CLI sign-in plumbing",
		"unkey_v2_cli_list_sessions":        "CLI sign-in plumbing",
		"unkey_v2_cli_revoke_session":       "CLI sign-in plumbing",
		"unkey_v2_cli_start_device_login":   "CLI sign-in plumbing",
		"unkey_v2_portal_exchange_code":     "creates a portal browser session",
		"unkey_v2_portal_get_verifications": "portal session cookie only",
		"unkey_v2_portal_list_keys":         "portal session cookie only",
		"unkey_v2_portal_reroll_key":        "portal session cookie only",
	}
	registry := loadEmbedded(t)
	published := readSpeakeasy(t)

	for _, toolset := range Toolsets {
		var want []string
		for _, tool := range published {
			if tool.toolset == toolset.ID && removed[tool.name] == "" {
				want = append(want, tool.name)
			}
		}
		var got []string
		for _, tool := range registry.Tools(toolset.ID) {
			got = append(got, tool.Name)
		}
		slices.Sort(want)
		require.Equal(t, want, got, "toolset %s", toolset.ID)
	}
}

// TestSchemaBudget keeps each toolset's tools/list payload under a fixed
// size. Clients load every tool definition into the model's context, so
// growth past the budget has to be a decision, not drift.
func TestSchemaBudget(t *testing.T) {
	budgets := map[ToolsetID]int{
		ToolsetV2:      300_000,
		ToolsetCompute: 140_000,
	}
	registry := loadEmbedded(t)

	type listedTool struct {
		Name        string          `json:"name"`
		Title       string          `json:"title"`
		Description string          `json:"description"`
		InputSchema json.RawMessage `json:"inputSchema"`
	}
	for _, toolset := range Toolsets {
		var listed []listedTool
		for _, tool := range registry.Tools(toolset.ID) {
			listed = append(listed, listedTool{Name: tool.Name, Title: tool.Title, Description: tool.Description, InputSchema: tool.InputSchema})
		}
		payload, err := json.Marshal(listed)
		require.NoError(t, err)
		t.Logf("%s: %d tools, %d bytes", toolset.ID, len(listed), len(payload))
		require.LessOrEqual(t, len(payload), budgets[toolset.ID], "toolset %s", toolset.ID)
	}
}

// TestEmbeddedSchemasAreSelfContained guarantees that clients which do not
// resolve JSON Schema references can still read every tool's input.
func TestEmbeddedSchemasAreSelfContained(t *testing.T) {
	registry := loadEmbedded(t)
	for _, tool := range registry.Tools(ToolsetV2) {
		require.NotContains(t, string(tool.InputSchema), `"$ref"`, tool.Name)
		var root map[string]any
		require.NoError(t, json.Unmarshal(tool.InputSchema, &root), tool.Name)
		require.Equal(t, "object", root["type"], tool.Name)
	}
}

func TestLoadBuildsToolsFromSpec(t *testing.T) {
	spec := testSpec{
		Paths: map[string]map[string]testOperation{
			"/v2/keys.createKey": {"post": {
				OperationID: "keys.createKey",
				Summary:     "Create key",
				Description: "Creates a key.",
				MCP:         toolsets(ToolsetV2, ToolsetCompute),
				RequestBody: jsonBody(map[string]any{"$ref": "#/components/schemas/CreateKey"}),
			}},
			"/v2/workspace.getLimits": {"post": {
				OperationID: "workspace.getLimits",
				MCP:         toolsets(ToolsetV2),
			}},
			"/v2/ratelimit.multiLimit": {"post": {
				OperationID: "ratelimit.multiLimit",
				MCP:         toolsets(ToolsetV2),
				RequestBody: jsonBody(map[string]any{"type": "array", "items": map[string]any{"type": "object"}}),
			}},
			"/v2/liveness": {"get": {
				OperationID: "liveness",
				MCP:         exclude("health check"),
			}},
		},
		Components: testComponents{Schemas: map[string]any{
			"CreateKey": map[string]any{
				"type":                      "object",
				"example":                   map[string]any{"apiId": "api_1"},
				"x-speakeasy-name-override": "CreateKey",
				"properties": map[string]any{
					"example": map[string]any{"type": "string", "examples": []any{"a"}},
					"meta": map[string]any{
						"$ref":        "#/components/schemas/Meta",
						"description": "Overrides the component description.",
					},
				},
			},
			"Meta": map[string]any{"type": "object", "description": "Component description."},
		}},
	}

	registry, err := Load(spec.encode(t))
	require.NoError(t, err)

	v2 := registry.Tools(ToolsetV2)
	require.Equal(t, []string{"unkey_v2_keys_create_key", "unkey_v2_ratelimit_multi_limit", "unkey_v2_workspace_get_limits"}, toolNames(v2))
	require.Equal(t, []string{"unkey_v2_keys_create_key"}, toolNames(registry.Tools(ToolsetCompute)))

	createKey := v2[0]
	require.Equal(t, "/v2/keys.createKey", createKey.Path)
	require.Equal(t, "Create key", createKey.Title)
	require.Equal(t, "Creates a key.", createKey.Description)
	require.Equal(t, map[string]any{
		"type": "object",
		"properties": map[string]any{
			"example": map[string]any{"type": "string"},
			"meta":    map[string]any{"type": "object", "description": "Overrides the component description."},
		},
	}, decodeSchema(t, createKey.InputSchema))
	require.False(t, createKey.WrapsBody)

	multiLimit := v2[1]
	require.True(t, multiLimit.WrapsBody)
	require.Equal(t, map[string]any{
		"type":                 "object",
		"properties":           map[string]any{"body": map[string]any{"type": "array", "items": map[string]any{"type": "object"}}},
		"required":             []any{"body"},
		"additionalProperties": false,
	}, decodeSchema(t, multiLimit.InputSchema))

	require.Equal(t, map[string]any{"type": "object"}, decodeSchema(t, v2[2].InputSchema))
}

// TestLoadRejectsInvalidOperations guarantees that a spec svc/mcp cannot turn
// into well-formed tools fails Load, naming the operation, so it fails CI
// before it can crash-loop the service.
func TestLoadRejectsInvalidOperations(t *testing.T) {
	object := jsonBody(map[string]any{"type": "object"})
	cases := []struct {
		name      string
		operation testOperation
		method    string
		extra     map[string]testOperation
		schemas   map[string]any
		wantError string
	}{
		{name: "missing extension", operation: testOperation{RequestBody: object}, wantError: "missing x-unkey-mcp"},
		{name: "both keys", operation: testOperation{RequestBody: object, MCP: map[string]any{"toolsets": []any{"v2"}, "exclude": "no"}}, wantError: "exactly one"},
		{name: "empty toolsets", operation: testOperation{RequestBody: object, MCP: map[string]any{"toolsets": []any{}}}, wantError: "non-empty list"},
		{name: "unknown toolset", operation: testOperation{RequestBody: object, MCP: toolsets("billing")}, wantError: "unknown toolset billing"},
		{name: "duplicate toolset", operation: testOperation{RequestBody: object, MCP: toolsets(ToolsetV2, ToolsetV2)}, wantError: "listed twice"},
		{name: "blank exclusion reason", operation: testOperation{RequestBody: object, MCP: exclude(" ")}, wantError: "non-empty reason"},
		{name: "deprecated tool", operation: testOperation{RequestBody: object, Deprecated: true, MCP: toolsets(ToolsetV2)}, wantError: "deprecated"},
		{name: "not POST", method: "get", operation: testOperation{MCP: toolsets(ToolsetV2)}, wantError: "POST"},
		{name: "parameters", operation: testOperation{RequestBody: object, MCP: toolsets(ToolsetV2), Parameters: []map[string]any{{"name": "id", "in": "query"}}}, wantError: "parameters"},
		{
			name:      "non-JSON body",
			operation: testOperation{MCP: toolsets(ToolsetV2), RequestBody: &testBody{Content: map[string]testContent{"text/plain": {Schema: map[string]any{"type": "string"}}}}},
			wantError: "application/json",
		},
		{
			name:      "recursive schema",
			operation: testOperation{MCP: toolsets(ToolsetV2), RequestBody: jsonBody(map[string]any{"$ref": "#/components/schemas/Node"})},
			schemas: map[string]any{"Node": map[string]any{
				"type":       "object",
				"properties": map[string]any{"child": map[string]any{"$ref": "#/components/schemas/Node"}},
			}},
			wantError: "recursive schema",
		},
		{name: "invalid name", operation: testOperation{OperationID: "keys.create Key", RequestBody: object, MCP: toolsets(ToolsetV2)}, wantError: "does not match"},
		{
			name:      "name collision",
			operation: testOperation{RequestBody: object, MCP: toolsets(ToolsetV2)},
			extra: map[string]testOperation{"/v2/keys-createKey": {
				OperationID: "keys-createKey",
				RequestBody: object,
				MCP:         toolsets(ToolsetCompute),
			}},
			wantError: "both derive tool name unkey_v2_keys_create_key",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			operation := tc.operation
			if operation.OperationID == "" {
				operation.OperationID = "keys.createKey"
			}
			method := tc.method
			if method == "" {
				method = "post"
			}
			spec := testSpec{
				Paths:      map[string]map[string]testOperation{"/v2/keys.createKey": {method: operation}},
				Components: testComponents{Schemas: tc.schemas},
			}
			for path, extra := range tc.extra {
				spec.Paths[path] = map[string]testOperation{"post": extra}
			}

			_, err := Load(spec.encode(t))
			require.ErrorContains(t, err, tc.wantError)
			require.ErrorContains(t, err, "keys")
		})
	}
}

type speakeasyTool struct {
	toolset     ToolsetID
	name        string
	operationID string
}

// readSpeakeasy reads testdata/speakeasy.txt: every tool on the Speakeasy
// install pages for /mcp/v2 and /mcp/compute, with the operationId it was
// generated from in the spec Speakeasy last built.
func readSpeakeasy(t *testing.T) []speakeasyTool {
	t.Helper()
	file, err := os.Open("testdata/speakeasy.txt")
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, file.Close()) })

	var published []speakeasyTool
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		require.Len(t, fields, 3, scanner.Text())
		published = append(published, speakeasyTool{toolset: ToolsetID(fields[0]), name: fields[1], operationID: fields[2]})
	}
	require.NoError(t, scanner.Err())
	return published
}

func loadEmbedded(t *testing.T) *Registry {
	t.Helper()
	registry, err := Load(openapi.Spec)
	require.NoError(t, err)
	return registry
}

func toolNames(list []Tool) []string {
	names := make([]string, 0, len(list))
	for _, tool := range list {
		names = append(names, tool.Name)
	}
	return names
}

func decodeSchema(t *testing.T, raw json.RawMessage) map[string]any {
	t.Helper()
	var schema map[string]any
	require.NoError(t, json.Unmarshal(raw, &schema))
	return schema
}

type testSpec struct {
	Paths      map[string]map[string]testOperation `yaml:"paths"`
	Components testComponents                      `yaml:"components"`
}

type testComponents struct {
	Schemas map[string]any `yaml:"schemas,omitempty"`
}

type testOperation struct {
	OperationID string           `yaml:"operationId"`
	Summary     string           `yaml:"summary,omitempty"`
	Description string           `yaml:"description,omitempty"`
	Deprecated  bool             `yaml:"deprecated,omitempty"`
	Parameters  []map[string]any `yaml:"parameters,omitempty"`
	RequestBody *testBody        `yaml:"requestBody,omitempty"`
	MCP         map[string]any   `yaml:"x-unkey-mcp,omitempty"`
}

type testBody struct {
	Content map[string]testContent `yaml:"content"`
}

type testContent struct {
	Schema map[string]any `yaml:"schema"`
}

func (s testSpec) encode(t *testing.T) []byte {
	t.Helper()
	encoded, err := yaml.Marshal(s)
	require.NoError(t, err)
	return encoded
}

func jsonBody(schema map[string]any) *testBody {
	return &testBody{Content: map[string]testContent{"application/json": {Schema: schema}}}
}

func toolsets(ids ...ToolsetID) map[string]any {
	list := make([]any, len(ids))
	for i, id := range ids {
		list[i] = string(id)
	}
	return map[string]any{"toolsets": list}
}

func exclude(reason string) map[string]any {
	return map[string]any{"exclude": reason}
}
