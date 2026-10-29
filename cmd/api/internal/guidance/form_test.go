package guidance

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"slices"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/sdks/api/go/v3/models/components"
	"github.com/unkeyed/sdks/api/go/v3/optionalnullable"
	"github.com/unkeyed/unkey/pkg/prompt"
	"github.com/unkeyed/unkey/pkg/uid"
)

func TestFormsPreserveOmittedClearedAndZeroValues(t *testing.T) {
	schema, ok := Lookup("keys update-key")
	require.True(t, ok)
	keyID := uid.New(uid.KeyPrefix)
	answers := &formAnswers{strings: []string{"0"}, choices: []string{"name", "clear", "enabled", "false", "expires", "set", "roles", "done", "done"}}
	values := map[string]any{"keyId": keyID}
	f := form{prompt: answers, out: io.Discard}
	require.NoError(t, f.object(t.Context(), schema, values, "", true))
	raw, err := json.Marshal(values)
	require.NoError(t, err)
	var request components.V2KeysUpdateKeyRequestBody
	require.NoError(t, json.Unmarshal(raw, &request))
	require.Equal(t, components.V2KeysUpdateKeyRequestBody{
		KeyID:   keyID,
		Name:    optionalnullable.From[string](nil),
		Enabled: new(false),
		Expires: optionalnullable.From(new(int64(0))),
		Roles:   []string{},
	}, request)
	require.Empty(t, answers.choices)
	require.Empty(t, answers.strings)
}

func TestFormsBuildNestedAppWithoutRawJSON(t *testing.T) {
	schema, ok := Lookup("apps create-app")
	require.True(t, ok)
	answers := &formAnswers{
		strings: []string{"Demo Service", "", "example/demo"},
		choices: []string{"0", "repository", "done"},
	}
	values := map[string]any{"project": "demo-project"}
	f := form{prompt: answers, out: io.Discard}
	require.NoError(t, f.object(t.Context(), schema, values, "", false))
	raw, err := json.Marshal(values)
	require.NoError(t, err)
	var request components.V2AppsCreateAppRequestBody1
	require.NoError(t, json.Unmarshal(raw, &request))
	require.Equal(t, "demo-project", request.Project)
	require.Equal(t, "Demo Service", request.Name)
	require.Equal(t, "demo-service", request.Slug)
	require.Equal(t, new("example/demo"), request.Git.Repository)
	require.Nil(t, request.Oci)
	require.Empty(t, answers.choices)
	require.Empty(t, answers.strings)
}

func TestFormsRejectInvalidValuesBeforeAcceptingInput(t *testing.T) {
	schema, ok := Lookup("projects create-project")
	require.True(t, ok)
	answers := &formAnswers{strings: []string{"", "Demo", "bad slug", "demo"}}
	f := form{prompt: answers, out: io.Discard}
	values := map[string]any{}
	require.NoError(t, f.object(t.Context(), schema, values, "", false))
	require.Equal(t, map[string]any{"name": "Demo", "slug": "demo"}, values)

	answers = &formAnswers{strings: []string{"Demo"}}
	f.prompt = answers
	require.ErrorIs(t, f.object(t.Context(), schema, map[string]any{}, "", false), io.EOF)
}

func TestReviewEditsPreservePatchValues(t *testing.T) {
	schema, ok := Lookup("keys update-key")
	require.True(t, ok)
	keyID := uid.New(uid.KeyPrefix)
	values := map[string]any{
		"keyId": keyID, "name": "Original", "enabled": false, "expires": json.Number("0"), "roles": []any{"reader"},
	}
	answers := &formAnswers{choices: []string{"name", "edit", "clear", "roles", "edit", "0", "remove", "done", "done"}}
	f := form{prompt: answers, out: io.Discard}
	require.NoError(t, f.review(t.Context(), schema, values, "", true))
	raw, err := json.Marshal(values)
	require.NoError(t, err)
	var request components.V2KeysUpdateKeyRequestBody
	require.NoError(t, json.Unmarshal(raw, &request))
	require.Equal(t, components.V2KeysUpdateKeyRequestBody{
		KeyID: keyID, Name: optionalnullable.From[string](nil), Enabled: new(false), Expires: optionalnullable.From(new(int64(0))), Roles: []string{},
	}, request)
	require.Empty(t, answers.choices)
	answers.choices = []string{"name", "omit", "done"}
	require.NoError(t, f.review(t.Context(), schema, values, "", true))
	require.NotContains(t, values, "name")
	require.Equal(t, false, values["enabled"])
}

func TestReviewUsesReadableLabelsWithoutChangingRequestValues(t *testing.T) {
	schema, ok := Lookup("keys update-key")
	require.True(t, ok)
	values := map[string]any{
		"keyId": uid.New(uid.KeyPrefix), "name": nil, "enabled": false,
		"expires": json.Number("0"), "roles": []any{},
	}
	before, err := json.Marshal(values)
	require.NoError(t, err)
	var output bytes.Buffer
	f := form{prompt: &formAnswers{choices: []string{"done"}}, out: &output}
	require.NoError(t, f.review(t.Context(), schema, values, "", true))
	for _, field := range []string{"Enabled: No", "Expires: 0", "Name: Clear (null)", "Roles: (no items)"} {
		require.Contains(t, output.String(), field)
	}
	require.NotContains(t, output.String(), "{")
	require.NotContains(t, output.String(), "Credits:")
	after, err := json.Marshal(values)
	require.NoError(t, err)
	require.Equal(t, before, after)
}

func TestReviewNestedEditsPreserveAndHideSecrets(t *testing.T) {
	schema, ok := Lookup("environments set-environment-variables")
	require.True(t, ok)
	secret := uid.New(uid.KeyPrefix)
	values := map[string]any{
		"project": "demo", "app": "demo-app", "environment": "preview",
		"variables": []any{map[string]any{"key": "DEMO_TOKEN", "value": secret, "description": "Original"}},
	}
	answers := &formAnswers{
		choices: []string{"variables", "0", "edit", "edit", "description", "edit", "edit", "done", "done", "done"},
		strings: []string{"Updated description"},
	}
	var output bytes.Buffer
	f := form{prompt: answers, out: &output}
	require.NoError(t, f.review(t.Context(), schema, values, "", false))
	raw, err := json.Marshal(values)
	require.NoError(t, err)
	var request components.V2EnvironmentsSetEnvironmentVariablesRequestBody
	require.NoError(t, json.Unmarshal(raw, &request))
	require.Equal(t, secret, request.Variables[0].Value)
	require.Equal(t, new("Updated description"), request.Variables[0].Description)
	require.NotContains(t, output.String(), secret)
	require.Contains(t, output.String(), "Variables:\n    Item 1:\n")
	require.Contains(t, output.String(), "Value: [hidden]")
	require.Contains(t, output.String(), "Description: Updated description")
	require.Empty(t, answers.choices)
	require.Empty(t, answers.strings)
	output.Reset()
	answers.choices = []string{"variables", "0", "edit", "edit", "description", "edit", "empty", "done", "done", "done"}
	require.NoError(t, f.review(t.Context(), schema, values, "", false))
	variables := values["variables"].([]any)
	require.Equal(t, "", variables[0].(map[string]any)["description"])
	require.Contains(t, output.String(), "Description: (empty text)")
}

func TestReviewMasksUnknownValuesAndEscapesTerminalControls(t *testing.T) {
	schema, ok := Lookup("keys update-key")
	require.True(t, ok)
	secret := uid.New(uid.KeyPrefix)
	values := map[string]any{
		"keyId": uid.New(uid.KeyPrefix), "name": "Demo\x1b[2J\u202e\nEnabled: Yes", "metadata": map[string]any{"private": secret},
	}
	var output bytes.Buffer
	f := form{prompt: &formAnswers{choices: []string{"done"}}, out: &output}
	require.NoError(t, f.review(t.Context(), schema, values, "", true))
	require.NotContains(t, output.String(), secret)
	require.NotContains(t, output.String(), "\x1b")
	require.NotContains(t, output.String(), "\u202e")
	require.NotContains(t, output.String(), "\nEnabled: Yes")
	require.Contains(t, output.String(), `\nEnabled: Yes`)
	require.Equal(t, "Demo\x1b[2J\u202e\nEnabled: Yes", values["name"])
}

func TestReviewMetadataEditsPreserveSiblingsAndAllowClearing(t *testing.T) {
	schema, ok := Lookup("keys update-key")
	require.True(t, ok)
	secret := uid.New(uid.KeyPrefix)
	values := map[string]any{
		"keyId": uid.New(uid.KeyPrefix),
		"meta":  map[string]any{"budget": json.Number("0"), "obsolete": "old", "private": secret},
	}
	answers := &formAnswers{
		choices: []string{"meta", "edit", "edit", "0", "edit", "number", "1", "remove", "add", "boolean", "false", "done", "done"},
		strings: []string{"1", "reviewed"},
	}
	var output bytes.Buffer
	f := form{prompt: answers, out: &output}
	require.NoError(t, f.review(t.Context(), schema, values, "", true))
	require.Equal(t, map[string]any{"budget": json.Number("1"), "private": secret, "reviewed": false}, values["meta"])
	require.NotContains(t, output.String(), secret)
	require.Empty(t, answers.choices)
	require.Empty(t, answers.strings)
	answers.choices = []string{"meta", "edit", "replace", "clear", "done"}
	require.NoError(t, f.review(t.Context(), schema, values, "", true))
	require.Contains(t, values, "meta")
	require.Nil(t, values["meta"])
}

func TestReviewChangesConfigurationWithoutMixingSources(t *testing.T) {
	schema, ok := Lookup("apps create-app")
	require.True(t, ok)
	values := map[string]any{
		"project": "demo", "name": "Demo", "slug": "demo",
		"git": map[string]any{"repository": "example/demo"},
	}
	answers := &formAnswers{choices: []string{"configuration", "1", "done"}, strings: []string{"ghcr.io/example/demo:v2"}}
	f := form{prompt: answers, out: io.Discard}
	require.NoError(t, f.review(t.Context(), schema, values, "", false))
	require.Equal(t, map[string]any{
		"project": "demo", "name": "Demo", "slug": "demo",
		"oci": map[string]any{"image": "ghcr.io/example/demo:v2"},
	}, values)
	require.Empty(t, answers.choices)
	require.Empty(t, answers.strings)
}

type formAnswers struct {
	strings []string
	choices []string
}

func (a *formAnswers) String(_ string, defaults ...string) (string, error) {
	if len(a.strings) == 0 {
		return "", io.EOF
	}
	value := a.strings[0]
	a.strings = a.strings[1:]
	if value == "" && len(defaults) > 0 {
		return defaults[0], nil
	}
	return value, nil
}

func (a *formAnswers) Secret(label string) (string, error) {
	return a.String(label)
}

func (a *formAnswers) SelectOrdered(_ string, options []prompt.SelectOption, _ ...string) (string, error) {
	if len(a.choices) == 0 {
		return "", io.EOF
	}
	choice := a.choices[0]
	a.choices = a.choices[1:]
	if !slices.ContainsFunc(options, func(option prompt.SelectOption) bool { return option.Key == choice }) {
		return "", fmt.Errorf("choice %q is not offered", choice)
	}
	return choice, nil
}

func TestSplitColumnsCarriesStyleAcrossWrappedLines(t *testing.T) {
	head, rest := splitColumns("\x1b[1m✓\x1b[0m \x1b[36m界abc\x1b[0m", 4)
	require.Equal(t, "\x1b[1m✓\x1b[0m \x1b[36m界\x1b[0m", head)
	require.Equal(t, "\x1b[36mabc\x1b[0m", rest)
	require.Equal(t, 4, visibleWidth(head))
}

func TestFieldProblemNamesFieldOnce(t *testing.T) {
	require.Equal(t, "Slug is already in use.", fieldProblem("body.slug", "Slug is already in use."))
	require.Equal(t, "Display name: must be at most 256 characters", fieldProblem("body.displayName", "must be at most 256 characters"))
	require.Equal(t, "Invalid input.", fieldProblem("", "Invalid input."))
}
