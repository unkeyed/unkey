package api

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/sdks/api/go/v3/models/apierrors"
	"github.com/unkeyed/sdks/api/go/v3/models/components"
	"github.com/unkeyed/unkey/pkg/cli"
	"github.com/unkeyed/unkey/pkg/uid"
)

func TestResourceErrorsHonorJSONBeforeAndAfterInvalidInput(t *testing.T) {
	secret := uid.New(uid.KeyPrefix)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		t.Error("invalid input must not send an API request")
		w.WriteHeader(http.StatusInternalServerError)
	}))
	t.Cleanup(server.Close)
	for _, test := range []struct {
		name      string
		canonical string
		legacy    string
		args      []string
		want      string
		kind      string
		flags     []string
	}{
		{name: "missing required", canonical: "projects create", legacy: "projects create-project", want: "required flag missing: name", kind: "missing_required_flag", flags: []string{"--name"}},
		{name: "unknown", canonical: "projects create", legacy: "projects create-project", args: []string{"--unknown=" + secret}, want: "unknown flag: unknown", kind: "unknown_flag", flags: []string{"--unknown"}},
		{name: "invalid enum", canonical: "projects list", legacy: "projects list-projects", args: []string{"--output=" + secret}, want: "invalid value for flag --output", kind: "invalid_flag_value", flags: []string{"--output"}},
		{name: "invalid number", canonical: "projects list", legacy: "projects list-projects", args: []string{"--limit", secret}, want: "expected an integer", kind: "invalid_flag_value", flags: []string{"--limit"}},
		{name: "invalid bool", canonical: "projects list", legacy: "projects list-projects", args: []string{"--json=" + secret}, want: "expected a boolean", kind: "invalid_flag_value", flags: []string{"--json"}},
		{name: "conflict", canonical: "projects create", legacy: "projects create-project", args: []string{"--body=null", "--name=" + secret}, want: "mutually exclusive", kind: "conflicting_flags", flags: []string{"--name", "--body"}},
		{name: "malformed body", canonical: "projects create", legacy: "projects create-project", args: []string{"--body={"}, want: "invalid JSON for --body"},
	} {
		for _, path := range []string{test.canonical, "api " + test.legacy} {
			for _, first := range []bool{true, false} {
				name := test.name + "/" + path + "/format-last"
				if first {
					name = test.name + "/" + path + "/format-first"
				}
				t.Run(name, func(t *testing.T) {
					args := append(strings.Fields(path), "--root-key="+secret, "--api-url="+server.URL)
					if first {
						args = append(args, "--error-format=json")
					}
					args = append(args, test.args...)
					if !first {
						args = append(args, "--error-format", "json")
					}
					stdout, stderr, err := runCLI(t, args...)
					require.Error(t, err)
					require.Empty(t, stdout)
					require.Empty(t, stderr)
					var envelope struct {
						Error struct {
							Message string                       `json:"message"`
							Type    string                       `json:"type"`
							Fields  []components.ValidationError `json:"fields"`
						} `json:"error"`
					}
					require.NoError(t, json.Unmarshal([]byte(err.Error()), &envelope))
					require.Contains(t, envelope.Error.Message, test.want)
					require.NotContains(t, err.Error(), secret)
					if test.kind != "" {
						require.Equal(t, "urn:unkey:cli:"+test.kind, envelope.Error.Type)
					} else {
						require.Empty(t, envelope.Error.Type)
					}
					var flags []string
					for _, field := range envelope.Error.Fields {
						flags = append(flags, field.Location)
					}
					require.Equal(t, test.flags, flags)
				})
			}
		}
	}
}

func TestResourceErrorFormatRespectsValuesAndSeparator(t *testing.T) {
	body, err := json.Marshal(components.V2ProjectsListProjectsRequestBody{Search: new("--error-format=json")})
	require.NoError(t, err)
	for _, test := range []struct {
		name string
		args []string
		json bool
	}{
		{name: "string value", args: []string{"--unknown", "--search", "--error-format=json"}},
		{name: "inline string value", args: []string{"--unknown", "--search=--error-format=json"}},
		{name: "body value", args: []string{"--unknown", "--body", "--error-format=json"}},
		{name: "body text", args: []string{"--unknown", "--body", string(body)}},
		{name: "separator", args: []string{"--unknown", "--", "--error-format=json"}},
		{name: "invalid number consumes format", args: []string{"--limit", "--error-format=json"}},
		{name: "later option after consumed format", args: []string{"--limit", "--error-format=human", "--error-format=json"}, json: true},
		{name: "unknown before format", args: []string{"--unknown", "--error-format=json"}, json: true},
		{name: "invalid format before valid", args: []string{"--error-format=invalid", "--error-format=json"}, json: true},
		{name: "last format wins", args: []string{"--unknown", "--error-format=json", "--error-format=human"}},
		{name: "missing value", args: []string{"--error-format=json", "--limit"}, json: true},
	} {
		t.Run(test.name, func(t *testing.T) {
			stdout, stderr, err := runCLI(t, append([]string{"projects", "list"}, test.args...)...)
			require.Error(t, err)
			require.Empty(t, stdout)
			require.Empty(t, stderr)
			require.Equal(t, test.json, json.Valid([]byte(err.Error())))
		})
	}
}

func TestRootAndGroupUnknownCommandsHonorJSON(t *testing.T) {
	for _, path := range []string{"missing", "projects missing", "api missing", "api projects missing", "deployments build-logs missing", "help missing"} {
		t.Run(path, func(t *testing.T) {
			stdout, stderr, err := runCLI(t, append(strings.Fields(path), "--error-format=json")...)
			require.Error(t, err)
			require.Empty(t, stdout)
			require.Empty(t, stderr)
			var envelope struct {
				Error struct {
					Type string `json:"type"`
				} `json:"error"`
			}
			require.NoError(t, json.Unmarshal([]byte(err.Error()), &envelope))
			require.Equal(t, "urn:unkey:cli:unknown_command", envelope.Error.Type)
			require.Contains(t, err.Error(), "unknown command: missing")
			_, _, err = runCLI(t, append(strings.Fields(path), "--body", "--error-format=json")...)
			require.Error(t, err)
			require.False(t, json.Valid([]byte(err.Error())), "%v", err)
		})
	}
}

func TestInteractiveWithoutTerminalUsesJSONBoundary(t *testing.T) {
	stdout, stderr, err := runCLI(t, "projects", "list", "--interactive", "--error-format=json")
	require.Error(t, err)
	require.Empty(t, stdout)
	require.Empty(t, stderr)
	require.True(t, json.Valid([]byte(err.Error())), "%v", err)
	require.Contains(t, err.Error(), "--interactive requires a terminal on stdin and stderr")
}

func TestActionWrappersReceiveTypedAPIErrorBeforeFormatting(t *testing.T) {
	requestID := uid.New(uid.RequestPrefix)
	response := apierrors.BadRequestErrorResponse{
		Meta: components.Meta{RequestID: requestID},
		Error_: components.BadRequestErrorDetails{
			Detail: "Invalid project.", Status: http.StatusBadRequest, Type: "urn:example:validation",
			Errors: []components.ValidationError{{Location: "body.slug", Message: "Choose another slug."}},
		},
	}
	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		calls.Add(1)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		if err := json.NewEncoder(w).Encode(response); err != nil {
			t.Error(err)
		}
	}))
	t.Cleanup(server.Close)
	commands := ResourceCommands()
	var target *cli.Command
	for _, group := range commands {
		if group.Name == "projects" {
			for _, command := range group.Commands {
				if command.Name == "create" {
					target = command
				}
			}
		}
	}
	require.NotNil(t, target)
	action := target.Action
	var typed *apierrors.BadRequestErrorResponse
	target.Action = func(ctx context.Context, cmd *cli.Command) error {
		err := action(ctx, cmd)
		require.True(t, errors.As(err, &typed), "wrapper needs the original SDK error")
		return err
	}
	root := &cli.Command{Name: "unkey", Commands: commands, OnError: HandleError}
	err := root.Run(t.Context(), []string{
		"unkey", "projects", "create", "--no-input", "--name=Payments", "--slug=payments",
		"--root-key=" + uid.New(uid.KeyPrefix), "--api-url=" + server.URL, "--error-format=json",
	})
	require.Error(t, err)
	require.NotNil(t, typed)
	require.Equal(t, int32(1), calls.Load())
	var envelope struct {
		Error struct {
			Message   string                       `json:"message"`
			Status    int                          `json:"status"`
			RequestID string                       `json:"requestId"`
			Fields    []components.ValidationError `json:"fields"`
		} `json:"error"`
	}
	require.NoError(t, json.Unmarshal([]byte(err.Error()), &envelope))
	require.Equal(t, response.Error_.Detail, envelope.Error.Message)
	require.Equal(t, http.StatusBadRequest, envelope.Error.Status)
	require.Equal(t, requestID, envelope.Error.RequestID)
	require.Len(t, envelope.Error.Fields, 1)
	require.Equal(t, "--slug", envelope.Error.Fields[0].Location)
}

func TestInvalidOutputEnvironmentUsesBoundaryAndAllowsOverride(t *testing.T) {
	secret := uid.New(uid.KeyPrefix)
	t.Setenv("UNKEY_OUTPUT", secret)
	response := components.V2ProjectsListProjectsResponseBody{Meta: components.Meta{RequestID: uid.New(uid.RequestPrefix)}}
	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		calls.Add(1)
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(response); err != nil {
			t.Error(err)
		}
	}))
	t.Cleanup(server.Close)
	args := []string{"projects", "list", "--root-key=" + uid.New(uid.KeyPrefix), "--api-url=" + server.URL, "--error-format=json"}
	stdout, stderr, err := runCLI(t, args...)
	require.Error(t, err)
	require.Empty(t, stdout)
	require.Empty(t, stderr)
	require.Zero(t, calls.Load())
	var envelope struct {
		Error struct {
			Type   string                       `json:"type"`
			Fields []components.ValidationError `json:"fields"`
		} `json:"error"`
	}
	require.NoError(t, json.Unmarshal([]byte(err.Error()), &envelope))
	require.Equal(t, "urn:unkey:cli:invalid_flag_value", envelope.Error.Type)
	require.Len(t, envelope.Error.Fields, 1)
	require.Equal(t, "--output", envelope.Error.Fields[0].Location)
	require.NotContains(t, err.Error(), secret)

	stdout, stderr, err = runCLI(t, append(args, "--output=json")...)
	require.NoError(t, err)
	require.Empty(t, stderr)
	require.Equal(t, int32(1), calls.Load())
	var got components.V2ProjectsListProjectsResponseBody
	require.NoError(t, json.Unmarshal([]byte(stdout), &got))
	require.Equal(t, response, got)
	stdout, stderr, err = runCLI(t, "projects", "list", "--help")
	require.NoError(t, err)
	require.Empty(t, stderr)
	require.NotContains(t, stdout, secret)
	require.Equal(t, int32(1), calls.Load())
}
