package api

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/cmd/api/internal/discovery"
	"github.com/unkeyed/unkey/pkg/cli"
	"github.com/unkeyed/unkey/pkg/uid"
)

func TestDescribeCatalogIsDeterministicAndOffline(t *testing.T) {
	t.Setenv("UNKEY_ROOT_KEY", "")
	t.Setenv("UNKEY_OUTPUT", "human")
	t.Setenv("UNKEY_CONFIG", filepath.Join(t.TempDir(), "missing.toml"))
	transport := http.DefaultTransport
	http.DefaultTransport = describeNoNetwork{t: t}
	t.Cleanup(func() { http.DefaultTransport = transport })
	output, diagnostic, err := runDescribe(t)
	require.NoError(t, err)
	require.Empty(t, diagnostic)
	var catalog discovery.Catalog
	require.NoError(t, json.Unmarshal([]byte(output), &catalog))
	require.Equal(t, discovery.Scope, catalog.Scope)
	require.Len(t, catalog.Commands, 100)
	require.Len(t, catalog.Commands, len(resourcePaths))
	require.True(t, slices.IsSortedFunc(catalog.Commands, func(a, b discovery.Entry) int { return strings.Compare(a.Path, b.Path) }))
	for _, entry := range catalog.Commands {
		require.NotEmpty(t, entry.Usage, entry.Path)
		require.Equal(t, "POST", entry.Endpoint.Method, entry.Path)
		require.NotContains(t, entry.Path, "api ")
	}
	for range 3 {
		again, diagnostic, err := runDescribe(t)
		require.NoError(t, err)
		require.Empty(t, diagnostic)
		require.Equal(t, output, again)
	}
	for _, path := range []string{"projects create", "gateway policies update", "deployments build-logs list", "workspace flags get"} {
		output, diagnostic, err := runDescribe(t, strings.Fields(path)...)
		require.NoError(t, err, path)
		require.Empty(t, diagnostic)
		var contract discovery.Contract
		require.NoError(t, json.Unmarshal([]byte(output), &contract))
		require.Equal(t, path, contract.Path)
	}
}

func TestDescribeRejectsUnknownAndNoncanonicalPaths(t *testing.T) {
	t.Setenv("UNKEY_OUTPUT", "")
	for _, path := range []string{"projects", "projects create-project", "projects create extra", "gateway policies", "api projects create-project", "login", "deploy", "no-such-command"} {
		t.Run(path, func(t *testing.T) {
			output, diagnostic, err := runDescribe(t, strings.Fields(path)...)
			require.ErrorContains(t, err, "unknown canonical API command")
			require.Empty(t, output)
			require.Empty(t, diagnostic)
		})
	}
}

func TestDescribeNeverIncludesAmbientValues(t *testing.T) {
	t.Setenv("UNKEY_OUTPUT", "human")
	baseline, _, err := runDescribe(t, "projects", "create")
	require.NoError(t, err)
	secrets := map[string]string{
		"UNKEY_ROOT_KEY":     uid.New(uid.KeyPrefix),
		"UNKEY_API_BASE_URL": "https://example.invalid/" + uid.New(uid.RequestPrefix),
		"UNKEY_CONFIG":       filepath.Join(t.TempDir(), uid.New(uid.RequestPrefix)),
		"HOME":               filepath.Join(t.TempDir(), uid.New(uid.RequestPrefix)),
	}
	for name, value := range secrets {
		t.Setenv(name, value)
	}
	t.Setenv("UNKEY_OUTPUT", "json")
	output, diagnostic, err := runDescribe(t, "projects", "create")
	require.NoError(t, err)
	require.Empty(t, diagnostic)
	require.Equal(t, baseline, output)
	for _, secret := range secrets {
		require.NotContains(t, output, secret)
	}
}

func TestDescribeProjectsContract(t *testing.T) {
	t.Setenv("UNKEY_OUTPUT", "")
	output, _, err := runDescribe(t, "projects", "create")
	require.NoError(t, err)
	var contract discovery.Contract
	require.NoError(t, json.Unmarshal([]byte(output), &contract))
	require.Equal(t, discovery.Endpoint{Method: "POST", Path: "/v2/projects.createProject"}, contract.Endpoint)
	require.Equal(t, "3.1.0", contract.OpenAPI)
	require.Equal(t, "https://spec.openapis.org/oas/3.1/dialect/base", contract.JSONSchemaDialect)
	require.Equal(t, "svc/api/openapi/openapi-generated.yaml", contract.Source)
	require.Contains(t, contract.Defaults, "omitted")
	require.NotEmpty(t, contract.Description)
	require.NotEmpty(t, contract.Examples)
	for _, example := range contract.Examples {
		require.Contains(t, example, "unkey projects create")
	}
	var body struct {
		Required bool `json:"required"`
	}
	require.NoError(t, json.Unmarshal(contract.RequestBody, &body))
	require.True(t, body.Required)
	var schema struct {
		Required             []string                   `json:"required"`
		AdditionalProperties bool                       `json:"additionalProperties"`
		Properties           map[string]json.RawMessage `json:"properties"`
	}
	require.NoError(t, json.Unmarshal(contract.Components["schemas"]["V2ProjectsCreateProjectRequestBody"], &schema))
	require.Equal(t, []string{"name", "slug"}, schema.Required)
	require.False(t, schema.AdditionalProperties)
	flags := describeFlagsByName(contract.Flags)
	require.True(t, flags["name"].Required)
	require.Equal(t, []string{"body"}, flags["name"].RequiredUnless)
	require.Equal(t, "string", flags["name"].Type)
	require.Equal(t, schema.Properties["name"], flags["name"].Schema)
	require.Equal(t, []string{"human", "json"}, flags["output"].Enum)
	require.Equal(t, "UNKEY_ROOT_KEY", flags["root-key"].Environment)
	require.Equal(t, "json", flags["body"].Encoding)
	require.Contains(t, flags["body"].Conflicts, "name")
	require.Contains(t, flags["interactive"].Conflicts, "no-input")
}

func TestDescribeAllCommandsHaveClosedRequestSchemas(t *testing.T) {
	t.Setenv("UNKEY_OUTPUT", "")
	commands := ResourceCommands()
	for _, path := range resourcePaths {
		t.Run(path, func(t *testing.T) {
			contract, err := discovery.Describe(commands, resourcePaths, path)
			require.NoError(t, err)
			encoded, err := json.Marshal(contract)
			require.NoError(t, err)
			var tree map[string]any
			require.NoError(t, json.Unmarshal(encoded, &tree))
			for _, ref := range describeReferences(tree) {
				require.True(t, strings.HasPrefix(ref, "#/components/"), ref)
				var current any = tree
				for _, token := range strings.Split(strings.TrimPrefix(ref, "#/"), "/") {
					object, ok := current.(map[string]any)
					require.True(t, ok, ref)
					name := strings.ReplaceAll(strings.ReplaceAll(token, "~1", "/"), "~0", "~")
					current, ok = object[name]
					require.True(t, ok, ref)
				}
			}
			for _, flag := range contract.Flags {
				if slices.Contains(flag.Conflicts, "body") && flag.Name != "interactive" {
					require.NotEmpty(t, flag.Schema, "request flag --%s needs a schema", flag.Name)
				}
			}
		})
	}
}

func TestDescribeCompositePolicyAndOptionalDeploymentSources(t *testing.T) {
	t.Setenv("UNKEY_OUTPUT", "")
	commands := ResourceCommands()
	policy, err := discovery.Describe(commands, resourcePaths, "gateway policies update")
	require.NoError(t, err)
	flag := describeFlagsByName(policy.Flags)["policy"]
	require.Equal(t, "json", flag.Encoding)
	require.ElementsMatch(t, []string{"name", "enabled", "match", "keyauth", "ratelimit", "firewall", "openapi", "logging"}, flag.RequestFields)
	require.ElementsMatch(t, []string{"keyauth", "ratelimit", "firewall", "openapi", "logging"}, flag.AtMostOneField)
	var schema struct {
		MinProperties        int                        `json:"minProperties"`
		AdditionalProperties *bool                      `json:"additionalProperties"`
		Properties           map[string]json.RawMessage `json:"properties"`
	}
	require.NoError(t, json.Unmarshal(flag.Schema, &schema))
	require.Equal(t, 1, schema.MinProperties)
	require.Equal(t, new(false), schema.AdditionalProperties)
	require.Len(t, schema.Properties, 8)

	deployment, err := discovery.Describe(commands, resourcePaths, "deployments create")
	require.NoError(t, err)
	require.Equal(t, "/v3/deployments.createDeployment", deployment.Endpoint.Path)
	require.Empty(t, deployment.RequireOneOf)
	flags := describeFlagsByName(deployment.Flags)
	for _, source := range []string{"git", "oci", "deployment"} {
		require.False(t, flags[source].Required)
		require.Equal(t, "json", flags[source].Encoding)
		require.NotEmpty(t, flags[source].Schema)
		for _, other := range []string{"git", "oci", "deployment", "body"} {
			if source != other {
				require.Contains(t, flags[source].Conflicts, other)
			}
		}
	}
	app, err := discovery.Describe(commands, resourcePaths, "apps create")
	require.NoError(t, err)
	require.Equal(t, []discovery.Choice{{Flags: []string{"git", "oci"}, Unless: []string{"body"}}}, app.RequireOneOf)
	settings, err := discovery.Describe(commands, resourcePaths, "environments settings update")
	require.NoError(t, err)
	require.Empty(t, describeFlagsByName(settings.Flags)["dockerfile"].Encoding, "nullable API strings are still plain CLI strings")
}

func TestDescribeBodylessAndArrayRequests(t *testing.T) {
	t.Setenv("UNKEY_OUTPUT", "")
	commands := ResourceCommands()
	for _, test := range []struct {
		path     string
		bodyFlag bool
	}{
		{path: "github install", bodyFlag: true},
		{path: "workspace flags get", bodyFlag: false},
		{path: "workspace limits", bodyFlag: false},
	} {
		t.Run(test.path, func(t *testing.T) {
			contract, err := discovery.Describe(commands, resourcePaths, test.path)
			require.NoError(t, err)
			var body *struct{}
			require.NoError(t, json.Unmarshal(contract.RequestBody, &body))
			require.Nil(t, body)
			require.Empty(t, contract.Components)
			flag, exists := describeFlagsByName(contract.Flags)["body"]
			require.Equal(t, test.bodyFlag, exists)
			if exists {
				require.Contains(t, flag.Usage, "empty JSON object")
				require.Equal(t, "json", flag.Encoding)
				require.Empty(t, flag.Schema, "a CLI compatibility flag must not invent a wire-body schema")
			}
		})
	}
	contract, err := discovery.Describe(commands, resourcePaths, "ratelimit multi-limit")
	require.NoError(t, err)
	flags := describeFlagsByName(contract.Flags)
	require.Equal(t, flags["body"].Schema, flags["limits"].Schema)
	require.Equal(t, "json", flags["limits"].Encoding)
	var schema struct {
		Type string `json:"type"`
	}
	require.NoError(t, json.Unmarshal(contract.Components["schemas"]["V2RatelimitMultiLimitRequestBody"], &schema))
	require.Equal(t, "array", schema.Type)
}

func TestDescribeInvalidOutputEnvironmentRemainsOfflineAndDoesNotLeakValue(t *testing.T) {
	secret := uid.New(uid.KeyPrefix)
	t.Setenv("UNKEY_OUTPUT", secret)
	transport := http.DefaultTransport
	http.DefaultTransport = describeNoNetwork{t: t}
	t.Cleanup(func() { http.DefaultTransport = transport })
	stdout, stderr, err := runDescribe(t)
	require.NoError(t, err)
	require.Empty(t, stderr)
	require.NotContains(t, stdout, secret)
	var catalog discovery.Catalog
	require.NoError(t, json.Unmarshal([]byte(stdout), &catalog))
	require.Len(t, catalog.Commands, len(resourcePaths))
}

type describeNoNetwork struct{ t *testing.T }

func (d describeNoNetwork) RoundTrip(*http.Request) (*http.Response, error) {
	d.t.Error("describe must not make network requests")
	return nil, fmt.Errorf("network disabled for discovery test")
}

func runDescribe(t *testing.T, args ...string) (string, string, error) {
	t.Helper()
	stdout, stderr := captureFile(t), captureFile(t)
	originalOut, originalErr := os.Stdout, os.Stderr
	os.Stdout, os.Stderr = stdout, stderr
	t.Cleanup(func() { os.Stdout, os.Stderr = originalOut, originalErr })
	root := &cli.Command{Name: "unkey", Commands: []*cli.Command{DescribeCmd()}}
	err := root.Run(t.Context(), append([]string{"unkey", "describe"}, args...))
	os.Stdout, os.Stderr = originalOut, originalErr
	_, seekErr := stdout.Seek(0, io.SeekStart)
	require.NoError(t, seekErr)
	_, seekErr = stderr.Seek(0, io.SeekStart)
	require.NoError(t, seekErr)
	out, readErr := io.ReadAll(stdout)
	require.NoError(t, readErr)
	diagnostic, readErr := io.ReadAll(stderr)
	require.NoError(t, readErr)
	return string(out), string(diagnostic), err
}

func describeFlagsByName(flags []discovery.Flag) map[string]discovery.Flag {
	byName := make(map[string]discovery.Flag, len(flags))
	for _, flag := range flags {
		byName[flag.Name] = flag
	}
	return byName
}

func describeReferences(value any) []string {
	var refs []string
	switch value := value.(type) {
	case map[string]any:
		if ref, ok := value["$ref"].(string); ok {
			refs = append(refs, ref)
		}
		for _, child := range value {
			refs = append(refs, describeReferences(child)...)
		}
	case []any:
		for _, child := range value {
			refs = append(refs, describeReferences(child)...)
		}
	}
	return refs
}
