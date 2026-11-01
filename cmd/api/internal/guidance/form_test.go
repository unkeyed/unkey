package guidance

import (
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
