package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync/atomic"
	"syscall"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/tonistiigi/vt100"
	"github.com/unkeyed/sdks/api/go/v3/models/components"
	"github.com/unkeyed/sdks/api/go/v3/optionalnullable"
	"github.com/unkeyed/unkey/pkg/cli"
	"github.com/unkeyed/unkey/pkg/codes"
	"github.com/unkeyed/unkey/pkg/uid"
	"golang.org/x/sys/unix"
)

func TestGuidedProcess(t *testing.T) {
	if os.Getenv("UNKEY_GUIDED_PROCESS") != "1" {
		return
	}
	var args []string
	if err := json.Unmarshal([]byte(os.Getenv("UNKEY_GUIDED_ARGS")), &args); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	command := &cli.Command{Name: "unkey", Commands: append(ResourceCommands(), Cmd())}
	if err := command.Run(context.Background(), append([]string{"unkey"}, args...)); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	os.Exit(0)
}

func TestGuidedTerminalCreateAndCancellation(t *testing.T) {
	requests := make(chan components.V2ProjectsCreateProjectRequestBody, 4)
	response := components.V2ProjectsCreateProjectResponseBody{
		Meta: components.Meta{RequestID: uid.New(uid.RequestPrefix)},
		Data: components.V2ProjectsCreateProjectResponseData{ID: uid.New(uid.ProjectPrefix)},
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var request components.V2ProjectsCreateProjectRequestBody
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Error(err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		requests <- request
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(response); err != nil {
			t.Error(err)
		}
	}))
	t.Cleanup(server.Close)
	flags := []string{"--root-key=" + uid.New(uid.KeyPrefix), "--api-url=" + server.URL}
	for _, mode := range []string{"human", "json"} {
		args := append([]string{"projects", "create", "--output=" + mode}, flags...)
		out, terminal, err := runGuidedTerminal(t, args, []terminalAnswer{
			{expect: "Name: ", answer: "Demo Service\r"},
			{expect: "Slug [demo-service]: ", answer: "\r"},
			{expect: "Create project", answer: "\r"},
		})
		require.NoError(t, err, terminal)
		require.Equal(t, components.V2ProjectsCreateProjectRequestBody{Name: "Demo Service", Slug: "demo-service"}, <-requests)
		require.Contains(t, terminal, "Review request\r\n  Name: Demo Service\r\n  Slug: demo-service")
		require.NotContains(t, terminal, "Edit Name")
		require.NotContains(t, terminal, "Edit Slug")
		require.NotContains(t, terminal, `"name":`)
		require.NotContains(t, out, "Guided input")
		if mode == "json" {
			var got components.V2ProjectsCreateProjectResponseBody
			require.NoError(t, json.Unmarshal([]byte(out), &got))
			require.Equal(t, response, got)
		} else {
			require.Contains(t, out, "Project created.")
			recordGuided(t, "create-after", terminal, out)
		}
	}
	for _, args := range [][]string{
		{"api", "projects", "create-project"},
		{"projects", "create", "--no-input"},
	} {
		out, terminal, err := runGuidedTerminal(t, append(args, flags...), nil)
		require.Error(t, err)
		require.Contains(t, terminal, "required flag missing: name")
		require.Empty(t, out)
		recordGuided(t, "create-before", terminal, out)
	}
	out, terminal, err := runGuidedTerminal(t, append([]string{"projects", "create"}, flags...), []terminalAnswer{
		{expect: "Name: ", answer: "\x04"},
	})
	require.Error(t, err)
	require.Empty(t, out)
	require.Contains(t, terminal, "cancelled")
	out, terminal, err = runGuidedTerminal(t, append([]string{"projects", "update", "--project=demo"}, flags...), []terminalAnswer{
		{expect: "Choose a field to update", answer: "\x03"},
	})
	require.Error(t, err)
	require.Empty(t, out)
	require.Contains(t, terminal, "cancelled")
	out, terminal, err = runGuidedTerminal(t, append([]string{"projects", "delete", "--project=demo", "--no-input"}, flags...), nil)
	require.Error(t, err)
	require.Empty(t, out)
	require.Contains(t, terminal, "confirmation is disabled by --no-input")
	require.Empty(t, requests)
}

func TestGuidedTerminalUpdatePreservesPatch(t *testing.T) {
	requests := make(chan components.V2KeysUpdateKeyRequestBody, 1)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var request components.V2KeysUpdateKeyRequestBody
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Error(err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		requests <- request
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(components.V2KeysUpdateKeyResponseBody{Meta: components.Meta{RequestID: uid.New(uid.RequestPrefix)}}); err != nil {
			t.Error(err)
		}
	}))
	t.Cleanup(server.Close)
	keyID := uid.New(uid.KeyPrefix)
	out, terminal, err := runGuidedTerminal(t, []string{
		"keys", "update", "--key-id=" + keyID, "--enabled=false", "--expires=0", "--roles=", "--interactive",
		"--root-key=" + uid.New(uid.KeyPrefix), "--api-url=" + server.URL,
	}, []terminalAnswer{
		{expect: "Additional settings", answer: "\x1b[B\r"},
		{expect: "Clear (null)", answer: "\x1b[B\r"},
		{expect: "Additional settings", answer: "\r"},
		{expect: "Update key", answer: "\r"},
	})
	require.NoError(t, err, terminal)
	require.Equal(t, components.V2KeysUpdateKeyRequestBody{
		KeyID: keyID, Enabled: new(false), Expires: optionalnullable.From(new(int64(0))), Roles: []string{}, Name: optionalnullable.From[string](nil),
	}, <-requests)
	require.Contains(t, out, "Key updated.")
	recordGuided(t, "update-after", terminal, out)
}

func TestGuidedTerminalSearchAndPlainList(t *testing.T) {
	requests := make(chan components.V2ProjectsListProjectsRequestBody, 2)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var request components.V2ProjectsListProjectsRequestBody
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Error(err)
			return
		}
		requests <- request
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(components.V2ProjectsListProjectsResponseBody{
			Meta: components.Meta{RequestID: uid.New(uid.RequestPrefix)},
			Data: []components.Project{}, Pagination: components.Pagination{HasMore: false},
		}); err != nil {
			t.Error(err)
		}
	}))
	t.Cleanup(server.Close)
	args := []string{"projects", "list", "--root-key=" + uid.New(uid.KeyPrefix), "--api-url=" + server.URL}
	out, terminal, err := runGuidedTerminal(t, args, nil)
	require.NoError(t, err, terminal)
	require.Empty(t, terminal)
	require.Contains(t, out, "No projects found.")
	require.Nil(t, (<-requests).Search)
	out, terminal, err = runGuidedTerminal(t, append(args, "--interactive"), []terminalAnswer{
		{expect: "Additional settings", answer: "\x1b[A\r"},
		{expect: "Search: ", answer: "Demo\r"},
		{expect: "Additional settings", answer: "\r"},
	})
	require.NoError(t, err, terminal)
	require.Equal(t, new("Demo"), (<-requests).Search)
	require.Contains(t, out, "No projects found.")
	require.NotContains(t, terminal, "Review request")
	require.NotContains(t, terminal, "Continue?")
	recordGuided(t, "search-after", terminal, out)
}

func TestGuidedTerminalPickerSearchPaginationAndDeleteCancellation(t *testing.T) {
	requests := make(chan components.V2ProjectsListProjectsRequestBody, 3)
	mutations := make(chan string, 1)
	projectID := uid.New(uid.ProjectPrefix)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v2/projects.listProjects" {
			mutations <- r.URL.Path
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		var request components.V2ProjectsListProjectsRequestBody
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Error(err)
			return
		}
		requests <- request
		response := components.V2ProjectsListProjectsResponseBody{
			Meta: components.Meta{RequestID: uid.New(uid.RequestPrefix)},
			Data: []components.Project{}, Pagination: components.Pagination{HasMore: false},
		}
		if request.Search != nil && request.Cursor == nil {
			response.Pagination = components.Pagination{HasMore: true, Cursor: new("demo-next-page")}
		}
		if request.Cursor != nil {
			response.Data = []components.Project{{ID: projectID, Name: "Demo Service", Slug: "demo-service"}}
		}
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(response); err != nil {
			t.Error(err)
		}
	}))
	t.Cleanup(server.Close)
	out, terminal, err := runGuidedTerminal(t, []string{
		"projects", "delete", "--root-key=" + uid.New(uid.KeyPrefix), "--api-url=" + server.URL,
	}, []terminalAnswer{
		{expect: "Browse existing resources", answer: "\r"},
		{expect: "No resources found", answer: "\r"},
		{expect: "Search: ", answer: "Demo\r"},
		{expect: "Next page", answer: "\x1b[B\r"},
		{expect: "Demo Service (", answer: "\r"},
		{expect: "Delete project", answer: "\r"},
		{expect: "[y/N]: ", answer: "n\r"},
	})
	require.Error(t, err)
	require.Empty(t, out)
	require.Contains(t, terminal, "cancelled")
	require.Nil(t, (<-requests).Search)
	searched := <-requests
	require.Equal(t, new("Demo"), searched.Search)
	require.Nil(t, searched.Cursor)
	page := <-requests
	require.Equal(t, new("Demo"), page.Search)
	require.Equal(t, new("demo-next-page"), page.Cursor)
	require.Empty(t, mutations)
	recordGuided(t, "picker-cancel-after", terminal, out)
}

func TestGuidedTerminalSecretIsHiddenAndPreserved(t *testing.T) {
	requests := make(chan components.V2KeysVerifyKeyRequestBody, 1)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var request components.V2KeysVerifyKeyRequestBody
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Error(err)
			return
		}
		requests <- request
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(components.V2KeysVerifyKeyResponseBody{
			Meta: components.Meta{RequestID: uid.New(uid.RequestPrefix)},
			Data: components.V2KeysVerifyKeyResponseData{Valid: false, Code: components.CodeNotFound},
		}); err != nil {
			t.Error(err)
		}
	}))
	t.Cleanup(server.Close)
	secret := " " + uid.New(uid.KeyPrefix) + " "
	out, terminal, err := runGuidedTerminal(t, []string{
		"keys", "verify", "--root-key=" + uid.New(uid.KeyPrefix), "--api-url=" + server.URL,
	}, []terminalAnswer{
		{expect: "Key (hidden): ", answer: secret + "\r"},
		{expect: "Additional settings", answer: "\r"},
		{expect: "Verify key", answer: "\r"},
	})
	require.NoError(t, err, terminal)
	require.Equal(t, secret, (<-requests).Key)
	require.NotContains(t, terminal, strings.TrimSpace(secret))
	require.NotContains(t, out, strings.TrimSpace(secret))
	require.Contains(t, out, "NOT_FOUND")
	recordGuided(t, "secret-after", terminal, out)
}

func TestGuidedTerminalDeploymentUsesOneOptionalSource(t *testing.T) {
	requests := make(chan components.V3DeploymentsCreateDeploymentRequestBody, 2)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var request components.V3DeploymentsCreateDeploymentRequestBody
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Error(err)
			return
		}
		requests <- request
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		if err := json.NewEncoder(w).Encode(components.V3DeploymentsCreateDeploymentResponseBody{
			Meta: components.Meta{RequestID: uid.New(uid.RequestPrefix)},
			Data: components.V3DeploymentsCreateDeploymentResponseData{DeploymentID: uid.New(uid.DeploymentPrefix)},
		}); err != nil {
			t.Error(err)
		}
	}))
	t.Cleanup(server.Close)
	args := []string{"deployments", "create", "--project=demo", "--app=demo-app", "--environment=preview", "--root-key=" + uid.New(uid.KeyPrefix), "--api-url=" + server.URL}
	_, terminal, err := runGuidedTerminal(t, args, nil)
	require.NoError(t, err, terminal)
	require.Empty(t, terminal)
	require.Nil(t, (<-requests).Oci)
	_, terminal, err = runGuidedTerminal(t, append(args, "--interactive"), []terminalAnswer{
		{expect: "Additional settings", answer: "\x1b[A\r"},
		{expect: "Oci.image: ", answer: "ghcr.io/example/demo:v1\r"},
		{expect: "Create deployment", answer: "\r"},
	})
	require.NoError(t, err, terminal)
	request := <-requests
	require.Equal(t, "ghcr.io/example/demo:v1", request.Oci.Image)
	require.Nil(t, request.Git)
	require.Nil(t, request.Deployment)
}

func TestGuidedTerminalPolicyMergesCompositeFlag(t *testing.T) {
	requests := make(chan components.V2GatewayUpdatePolicyRequestBody, 1)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var request components.V2GatewayUpdatePolicyRequestBody
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Error(err)
			return
		}
		requests <- request
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(components.V2GatewayUpdatePolicyResponseBody{Meta: components.Meta{RequestID: uid.New(uid.RequestPrefix)}}); err != nil {
			t.Error(err)
		}
	}))
	t.Cleanup(server.Close)
	policyID := uid.New(uid.PolicyPrefix)
	policy, err := json.Marshal(struct {
		Enabled bool `json:"enabled"`
	}{Enabled: false})
	require.NoError(t, err)
	_, terminal, err := runGuidedTerminal(t, []string{
		"gateway", "policies", "update", "--project=demo", "--app=demo-app", "--environment=preview", "--policy-id=" + policyID,
		"--policy=" + string(policy), "--interactive", "--root-key=" + uid.New(uid.KeyPrefix), "--api-url=" + server.URL,
	}, []terminalAnswer{
		{expect: "Additional settings", answer: "\r"},
		{expect: "Update policy", answer: "\r"},
	})
	require.NoError(t, err, terminal)
	request := <-requests
	require.Equal(t, policyID, request.PolicyID)
	require.Equal(t, new(false), request.Enabled)
	require.Nil(t, request.Name)
}

func TestGuidedTerminalReviewEditsAndCancellation(t *testing.T) {
	requests := make(chan components.V2ProjectsCreateProjectRequestBody, 2)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var request components.V2ProjectsCreateProjectRequestBody
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Error(err)
			return
		}
		requests <- request
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(components.V2ProjectsCreateProjectResponseBody{
			Meta: components.Meta{RequestID: uid.New(uid.RequestPrefix)},
			Data: components.V2ProjectsCreateProjectResponseData{ID: uid.New(uid.ProjectPrefix)},
		}); err != nil {
			t.Error(err)
		}
	}))
	t.Cleanup(server.Close)
	args := []string{"projects", "create", "--root-key=" + uid.New(uid.KeyPrefix), "--api-url=" + server.URL}
	out, terminal, err := runGuidedTerminal(t, args, []terminalAnswer{
		{expect: "Name: ", answer: "Demo Srevice\r"},
		{expect: "Slug [demo-srevice]: ", answer: "\r"},
		{expect: "Create project", answer: "\x1b[B\r"},
		{expect: "Edit Name", answer: "\x1b[B\r"},
		{expect: "Name [Demo Srevice]: ", answer: "Demo Service\r"},
		{expect: "Edit Slug", answer: "\x1b[B\x1b[B\r"},
		{expect: "Slug [demo-srevice]: ", answer: "demo-service\r"},
		{expect: "Continue", answer: "\r"},
		{expect: "Create project", answer: "\r"},
	})
	require.NoError(t, err, terminal)
	require.Len(t, requests, 1)
	require.Equal(t, components.V2ProjectsCreateProjectRequestBody{Name: "Demo Service", Slug: "demo-service"}, <-requests)
	require.Contains(t, out, "Project created.")
	recordGuided(t, "review-edit-after", terminal, out)
	out, terminal, err = runGuidedTerminal(t, args, []terminalAnswer{
		{expect: "Name: ", answer: "Demo Service\r"},
		{expect: "Slug [demo-service]: ", answer: "\r"},
		{expect: "Create project", answer: "\x03"},
	})
	require.Error(t, err)
	require.Empty(t, out)
	require.Empty(t, requests)
	require.Contains(t, terminal, "cancelled")
	screen := vt100.NewVT100(40, 100)
	_, err = screen.Write([]byte(terminal))
	require.NoError(t, err)
	var visible strings.Builder
	for _, row := range screen.Content {
		visible.WriteString(strings.TrimRight(string(row), " ") + "\n")
	}
	require.Contains(t, visible.String(), "Review request\n  Name: Demo Service\n  Slug: demo-service")
	require.Contains(t, visible.String(), "cancelled")
	require.NotContains(t, visible.String(), "Create project")
	require.NotContains(t, visible.String(), "Edit Name")
	require.NotContains(t, visible.String(), "Edit Slug")
	out, terminal, err = runGuidedTerminal(t, args, []terminalAnswer{
		{expect: "Name: ", answer: "Demo Service\r"},
		{expect: "Slug [demo-service]: ", answer: "\r"},
		{expect: "Create project", answer: "\x1b[A\r"},
	})
	require.Error(t, err)
	require.Empty(t, out)
	require.Empty(t, requests)
	require.Contains(t, terminal, "cancelled")
}

func TestGuidedTerminalValidationCorrectionIsExplicit(t *testing.T) {
	requests := make(chan components.V2ProjectsCreateProjectRequestBody, 4)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var request components.V2ProjectsCreateProjectRequestBody
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Error(err)
			return
		}
		requests <- request
		w.Header().Set("Content-Type", "application/json")
		if request.Slug == "demo-service" {
			w.WriteHeader(http.StatusBadRequest)
			if err := json.NewEncoder(w).Encode(struct {
				Meta  components.Meta                   `json:"meta"`
				Error components.BadRequestErrorDetails `json:"error"`
			}{
				Meta: components.Meta{RequestID: uid.New(uid.RequestPrefix)},
				Error: components.BadRequestErrorDetails{
					Type: codes.App.Validation.InvalidInput.DocsURL(), Status: http.StatusBadRequest,
					Detail: "Invalid input.", Errors: []components.ValidationError{{Location: "body.slug", Message: "Use a different slug."}},
				},
			}); err != nil {
				t.Error(err)
			}
			return
		}
		if err := json.NewEncoder(w).Encode(components.V2ProjectsCreateProjectResponseBody{
			Meta: components.Meta{RequestID: uid.New(uid.RequestPrefix)},
			Data: components.V2ProjectsCreateProjectResponseData{ID: uid.New(uid.ProjectPrefix)},
		}); err != nil {
			t.Error(err)
		}
	}))
	t.Cleanup(server.Close)
	args := []string{"projects", "create", "--root-key=" + uid.New(uid.KeyPrefix), "--api-url=" + server.URL}
	out, terminal, err := runGuidedTerminal(t, args, []terminalAnswer{
		{expect: "Name: ", answer: "Demo Service\r"},
		{expect: "Slug [demo-service]: ", answer: "\r"},
		{expect: "Create project", answer: "\r"},
		{expect: "Edit saved answers", answer: "\x1b[B\r"},
		{expect: "Continue", answer: "\r"},
		{expect: "Change the rejected input", answer: "\x1b[B\x1b[B\r"},
		{expect: "Slug [demo-service]: ", answer: "demo-service-fixed\r"},
		{expect: "Continue", answer: "\r"},
		{expect: "Create project", answer: "\x1b[B\r"},
		{expect: "Edit Slug", answer: "\x1b[B\x1b[B\r"},
		{expect: "Slug [demo-service-fixed]: ", answer: "demo-service\r"},
		{expect: "Continue", answer: "\r"},
		{expect: "Create project", answer: "\r"},
		{expect: "Edit Slug", answer: "\x1b[B\x1b[B\r"},
		{expect: "Slug [demo-service]: ", answer: "demo-service-fixed\r"},
		{expect: "Continue", answer: "\r"},
		{expect: "Create project", answer: "\r"},
	})
	require.NoError(t, err, terminal)
	require.Len(t, requests, 2)
	require.Equal(t, components.V2ProjectsCreateProjectRequestBody{Name: "Demo Service", Slug: "demo-service"}, <-requests)
	require.Equal(t, components.V2ProjectsCreateProjectRequestBody{Name: "Demo Service", Slug: "demo-service-fixed"}, <-requests)
	require.Equal(t, 1, strings.Count(terminal, "\r\nName: "))
	require.Contains(t, out, "Project created.")
	recordGuided(t, "validation-correction-after", terminal, out)
	out, terminal, err = runGuidedTerminal(t, args, []terminalAnswer{
		{expect: "Name: ", answer: "Demo Service\r"},
		{expect: "Slug [demo-service]: ", answer: "\r"},
		{expect: "Create project", answer: "\r"},
		{expect: "Edit saved answers", answer: "\r"},
	})
	require.Error(t, err)
	require.Empty(t, out)
	require.Len(t, requests, 1)
	require.Contains(t, terminal, "Invalid input.")
	<-requests
	out, terminal, err = runGuidedTerminal(t, args, []terminalAnswer{
		{expect: "Name: ", answer: "Demo Service\r"},
		{expect: "Slug [demo-service]: ", answer: "\r"},
		{expect: "Create project", answer: "\r"},
		{expect: "Edit saved answers", answer: "\x1b[B\r"},
		{expect: "Edit Slug", answer: "\x1b[B\x1b[B\r"},
		{expect: "Slug [demo-service]: ", answer: "demo-service-fixed\r"},
		{expect: "Continue", answer: "\r"},
		{expect: "Create project", answer: "\x1b[A\r"},
	})
	require.Error(t, err)
	require.Empty(t, out)
	require.Len(t, requests, 1)
	require.Contains(t, terminal, "cancelled")
	<-requests
	out, terminal, err = runGuidedTerminal(t, append(args, "--name=Demo Service", "--slug=demo-service", "--no-input"), nil)
	require.Error(t, err)
	require.Empty(t, out)
	require.Len(t, requests, 1)
	require.NotContains(t, terminal, "Edit saved answers")
}

func TestGuidedTerminalDoesNotRetryOtherFailures(t *testing.T) {
	for _, status := range []int{http.StatusBadRequest, http.StatusForbidden, http.StatusConflict, http.StatusUnprocessableEntity, http.StatusInternalServerError} {
		t.Run(strconv.Itoa(status), func(t *testing.T) {
			var requests atomic.Int64
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				requests.Add(1)
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(status)
				if err := json.NewEncoder(w).Encode(struct {
					Meta  components.Meta                   `json:"meta"`
					Error components.BadRequestErrorDetails `json:"error"`
				}{
					Meta: components.Meta{RequestID: uid.New(uid.RequestPrefix)},
					Error: components.BadRequestErrorDetails{
						Type: "urn:example:operation-failed", Status: int64(status), Detail: "Request failed.",
						Errors: []components.ValidationError{{Location: "body.slug", Message: "Could not process."}},
					},
				}); err != nil {
					t.Error(err)
				}
			}))
			t.Cleanup(server.Close)
			out, terminal, err := runGuidedTerminal(t, []string{
				"projects", "create", "--name=Demo Service", "--slug=demo-service", "--interactive",
				"--root-key=" + uid.New(uid.KeyPrefix), "--api-url=" + server.URL,
			}, []terminalAnswer{{expect: "Create project", answer: "\r"}})
			require.Error(t, err)
			require.Empty(t, out)
			require.Equal(t, int64(1), requests.Load())
			require.NotContains(t, terminal, "Edit saved answers")
		})
	}
}

type terminalAnswer struct {
	expect string
	answer string
}

func runGuidedTerminal(t *testing.T, args []string, answers []terminalAnswer) (string, string, error) {
	t.Helper()
	master, err := os.OpenFile("/dev/ptmx", os.O_RDWR|unix.O_NOCTTY, 0)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, master.Close()) })
	require.NoError(t, unix.IoctlSetPointerInt(int(master.Fd()), unix.TIOCSPTLCK, 0))
	number, err := unix.IoctlGetInt(int(master.Fd()), unix.TIOCGPTN)
	require.NoError(t, err)
	slave, err := os.OpenFile("/dev/pts/"+strconv.Itoa(number), os.O_RDWR|unix.O_NOCTTY, 0)
	require.NoError(t, err)
	require.NoError(t, unix.IoctlSetWinsize(int(slave.Fd()), unix.TIOCSWINSZ, &unix.Winsize{Col: 100, Row: 40}))
	ctx, cancel := context.WithTimeout(t.Context(), 15*time.Second)
	t.Cleanup(cancel)
	raw, err := json.Marshal(args)
	require.NoError(t, err)
	command := exec.CommandContext(ctx, os.Args[0], "-test.run=^TestGuidedProcess$")
	command.Env = append(os.Environ(), "UNKEY_GUIDED_PROCESS=1", "UNKEY_GUIDED_ARGS="+string(raw), "UNKEY_OUTPUT=human")
	command.Stdin, command.Stderr = slave, slave
	command.SysProcAttr = &syscall.SysProcAttr{Setsid: true, Setctty: true, Ctty: 0}
	var output bytes.Buffer
	command.Stdout = &output
	require.NoError(t, command.Start())
	require.NoError(t, slave.Close())
	chunks := make(chan string, 256)
	go func() {
		defer close(chunks)
		buffer := make([]byte, 4096)
		for {
			count, err := master.Read(buffer)
			if count > 0 {
				chunks <- string(buffer[:count])
			}
			if err != nil {
				return
			}
		}
	}()
	var terminal strings.Builder
	position := 0
	for _, answer := range answers {
		for !strings.Contains(terminal.String()[position:], answer.expect) {
			select {
			case chunk, ok := <-chunks:
				require.True(t, ok, "process ended before %q: %s", answer.expect, terminal.String())
				terminal.WriteString(chunk)
			case <-ctx.Done():
				t.Fatalf("waiting for %q: %s", answer.expect, terminal.String())
			}
		}
		position = terminal.Len()
		_, err := io.WriteString(master, answer.answer)
		require.NoError(t, err)
	}
	err = command.Wait()
	for chunk := range chunks {
		terminal.WriteString(chunk)
	}
	return output.String(), terminal.String(), err
}

func recordGuided(t *testing.T, name, terminal, output string) {
	t.Helper()
	directory := os.Getenv("UNKEY_GUIDED_RECORD_DIR")
	if directory == "" {
		return
	}
	require.NoError(t, os.WriteFile(filepath.Join(directory, name+".log"), []byte("STDERR (terminal)\n"+terminal+"\nSTDOUT\n"+output), 0600))
}
