package api

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/cli"
)

func TestEveryAPILeafHasOneBodyFlag(t *testing.T) {
	var visit func(*cli.Command)
	visit = func(command *cli.Command) {
		if len(command.Commands) > 0 {
			for _, child := range command.Commands {
				visit(child)
			}
			return
		}
		count := 0
		for _, flag := range command.Flags {
			if flag.Name() == "body" {
				count++
			}
		}
		require.Equal(t, 1, count, command.Name)
	}
	visit(Cmd())
}

func TestTypedBodyRejectsRequestFlags(t *testing.T) {
	called := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
		called = true
	}))
	t.Cleanup(server.Close)

	root := &cli.Command{Name: "unkey", Commands: []*cli.Command{Cmd()}}
	err := root.Run(context.Background(), []string{"unkey", "api", "apis", "create-api", `--body={"name":"from-body"}`, "--name=ignored", "--api-url=" + server.URL, "--root-key=test", "--output=json"})
	require.ErrorContains(t, err, "flags --name and --body are mutually exclusive")
	require.False(t, called)
}

func TestBodyRejectsInvalidJSONLocally(t *testing.T) {
	called := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
		called = true
	}))
	t.Cleanup(server.Close)

	root := &cli.Command{Name: "unkey", Commands: []*cli.Command{Cmd()}}
	err := root.Run(context.Background(), []string{"unkey", "api", "apis", "create-api", "--body={", "--api-url=" + server.URL, "--root-key=test"})
	require.ErrorContains(t, err, "invalid JSON for --body")
	require.False(t, called)
}

func TestExplicitEmptyBodyReachesJSONDecoding(t *testing.T) {
	root := &cli.Command{Name: "unkey", Commands: []*cli.Command{Cmd()}}
	err := root.Run(context.Background(), []string{"unkey", "api", "apis", "create-api", "--body=", "--root-key=test"})
	require.ErrorContains(t, err, "invalid JSON for --body")
}

func TestUpdatePolicyBodyPreservesMatchClear(t *testing.T) {
	var got []byte
	var readErr error
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
		got, readErr = io.ReadAll(request.Body)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"meta":{"requestId":"test"},"data":{}}`))
	}))
	t.Cleanup(server.Close)

	body := `{"project":"payments","app":"payments-api","environment":"production","policyId":"pol_123","match":null}`
	root := &cli.Command{Name: "unkey", Commands: []*cli.Command{Cmd()}}
	err := root.Run(context.Background(), []string{"unkey", "api", "gateway", "update-policy", "--body=" + body, "--api-url=" + server.URL, "--root-key=test"})
	require.NoError(t, err)
	require.NoError(t, readErr)

	var request map[string]json.RawMessage
	require.NoError(t, json.Unmarshal(got, &request))
	require.JSONEq(t, `[]`, string(request["match"]))
}

func TestBodyBypassesSpecialRequestConstruction(t *testing.T) {
	tests := []struct {
		name       string
		command    []string
		body       string
		statusCode int
	}{
		{
			name:       "deployment union",
			command:    []string{"deployments", "create-deployment"},
			body:       `{"project":"project","app":"app","environment":"production","git":{"branch":"main"}}`,
			statusCode: http.StatusCreated,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var got []byte
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
				var err error
				got, err = io.ReadAll(request.Body)
				require.NoError(t, err)
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tt.statusCode)
				_, err = w.Write([]byte("{\"meta\":{\"requestId\":\"test\"},\"data\":{}}"))
				require.NoError(t, err)
			}))
			t.Cleanup(server.Close)

			args := []string{"unkey", "api"}
			args = append(args, tt.command...)
			args = append(args, "--body="+tt.body, "--api-url="+server.URL, "--root-key=test")
			root := &cli.Command{Name: "unkey", Commands: []*cli.Command{Cmd()}}
			require.NoError(t, root.Run(context.Background(), args))
			require.NotEmpty(t, got)
		})
	}
}
