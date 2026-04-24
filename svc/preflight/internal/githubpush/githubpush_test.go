package githubpush

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// fakeGitHub serves the endpoints PushFile touches, letting the test
// assert on the App JWT and the update-file payload without talking
// to real GitHub.
//
// Branch state is modeled with branchExists. Default true so both
// PushFile call sites in the existing tests stay on the "branch
// already there" happy path; flip to false to exercise the
// ensureBranch -> createRef flow.
type fakeGitHub struct {
	t              *testing.T
	installations  int // how many times the installation-token endpoint was hit
	lastAppJWT     string
	branchExists   bool   // controls GET /repos/.../branches/{branch}
	initialFileSHA string // if set, lookupFileSHA returns this SHA; if empty, returns 404
	newCommitSHA   string
	createdRef     map[string]any // captured POST /repos/.../git/refs body
	capturedBody   map[string]any
}

func newFakeGitHub(t *testing.T) (*httptest.Server, *fakeGitHub) {
	gh := &fakeGitHub{t: t, branchExists: true, newCommitSHA: "aaaabbbbccccdddd"}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasPrefix(r.URL.Path, "/app/installations/") && strings.HasSuffix(r.URL.Path, "/access_tokens") && r.Method == http.MethodPost:
			gh.installations++
			gh.lastAppJWT = r.Header.Get("Authorization")
			w.Header().Set("Content-Type", "application/json")
			// 50 minutes in the future: comfortably past the 5-minute
			// refresh window we use internally.
			_ = json.NewEncoder(w).Encode(map[string]any{
				"token":      "installtok_xyz",
				"expires_at": time.Now().Add(50 * time.Minute).UTC().Format(time.RFC3339),
			})
		case strings.HasPrefix(r.URL.Path, "/repos/") && strings.Contains(r.URL.Path, "/branches/") && r.Method == http.MethodGet:
			if !gh.branchExists {
				w.WriteHeader(http.StatusNotFound)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{"name": "main"})
		case strings.HasPrefix(r.URL.Path, "/repos/") && strings.Contains(r.URL.Path, "/git/ref/heads/") && r.Method == http.MethodGet:
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{
				"object": map[string]any{"sha": "deadbeefdeadbeefdeadbeefdeadbeefdeadbeef"},
			})
		case strings.HasPrefix(r.URL.Path, "/repos/") && strings.HasSuffix(r.URL.Path, "/git/refs") && r.Method == http.MethodPost:
			body, _ := io.ReadAll(r.Body)
			_ = json.Unmarshal(body, &gh.createdRef)
			w.WriteHeader(http.StatusCreated)
			_ = json.NewEncoder(w).Encode(map[string]any{"ref": gh.createdRef["ref"]})
		case strings.HasPrefix(r.URL.Path, "/repos/") && strings.Contains(r.URL.Path, "/contents/") && r.Method == http.MethodGet:
			if gh.initialFileSHA == "" {
				w.WriteHeader(http.StatusNotFound)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{"sha": gh.initialFileSHA})
		case strings.HasPrefix(r.URL.Path, "/repos/") && strings.Contains(r.URL.Path, "/contents/") && r.Method == http.MethodPut:
			body, _ := io.ReadAll(r.Body)
			_ = json.Unmarshal(body, &gh.capturedBody)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusCreated)
			_ = json.NewEncoder(w).Encode(map[string]any{
				"commit": map[string]any{"sha": gh.newCommitSHA},
			})
		case r.URL.Path == "/repos/owner/repo" && r.Method == http.MethodGet:
			// Used by ensureBranch's defaultBranch lookup when branchExists=false.
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{"default_branch": "main"})
		default:
			// Repo metadata endpoint may be hit with the other repo
			// fixture; emulate the same shape so tests that flip
			// branchExists=false still find a default branch.
			if strings.HasPrefix(r.URL.Path, "/repos/") && r.Method == http.MethodGet &&
				!strings.Contains(r.URL.Path, "/contents/") &&
				!strings.Contains(r.URL.Path, "/branches/") &&
				!strings.Contains(r.URL.Path, "/git/") {
				w.Header().Set("Content-Type", "application/json")
				_ = json.NewEncoder(w).Encode(map[string]any{"default_branch": "main"})
				return
			}
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
			w.WriteHeader(http.StatusTeapot)
		}
	}))
	t.Cleanup(srv.Close)
	return srv, gh
}

// generateTestKey returns a 2048-bit RSA key PEM-encoded so we do not
// need to stash a fixture on disk. 2048 keeps the test under 100ms.
func generateTestKey(t *testing.T) string {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generate rsa key: %v", err)
	}
	der := x509.MarshalPKCS1PrivateKey(key)
	return string(pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: der}))
}

func TestPushFile_CreateThenUpdate(t *testing.T) {
	srv, gh := newFakeGitHub(t)

	c, err := New(Config{
		AppID:          12345,
		InstallationID: 67890,
		PrivateKeyPEM:  generateTestKey(t),
		BaseURL:        srv.URL,
		HTTPClient:     srv.Client(),
		Now:            time.Now,
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// First push: file does not exist. PushFile should succeed without
	// sending a `sha` field on the payload.
	sha, err := c.PushFile(ctx, "unkeyed/preflight-test-app", "main", ".preflight-timestamp", []byte("run-1"), "preflight: run-1")
	if err != nil {
		t.Fatalf("PushFile (create): %v", err)
	}
	if sha != gh.newCommitSHA {
		t.Errorf("expected new commit sha %q, got %q", gh.newCommitSHA, sha)
	}
	if _, ok := gh.capturedBody["sha"]; ok {
		t.Error("expected no `sha` field on create, got one")
	}
	if gh.capturedBody["branch"] != "main" {
		t.Errorf("expected branch=main, got %v", gh.capturedBody["branch"])
	}

	// Second push: file now exists. PushFile must include the existing sha.
	gh.initialFileSHA = "existingfileSHA_9999"
	gh.newCommitSHA = "eeefffgggghhhh"
	gh.capturedBody = nil

	sha2, err := c.PushFile(ctx, "unkeyed/preflight-test-app", "main", ".preflight-timestamp", []byte("run-2"), "preflight: run-2")
	if err != nil {
		t.Fatalf("PushFile (update): %v", err)
	}
	if sha2 != "eeefffgggghhhh" {
		t.Errorf("expected second sha, got %q", sha2)
	}
	if gh.capturedBody["sha"] != gh.initialFileSHA {
		t.Errorf("expected update payload to carry existing sha, got %v", gh.capturedBody["sha"])
	}

	// Only one token exchange despite two PushFile calls: the cache
	// should have been reused inside the 5-minute refresh window.
	if gh.installations != 1 {
		t.Errorf("expected 1 installation-token exchange, got %d", gh.installations)
	}
}

func TestPushFile_CreatesMissingBranchFromDefault(t *testing.T) {
	srv, gh := newFakeGitHub(t)
	gh.branchExists = false

	c, err := New(Config{
		AppID:          1,
		InstallationID: 1,
		PrivateKeyPEM:  generateTestKey(t),
		BaseURL:        srv.URL,
		HTTPClient:     srv.Client(),
		Now:            time.Now,
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if _, err := c.PushFile(ctx, "owner/repo", "preflight/region", ".preflight-timestamp", []byte("x"), "msg"); err != nil {
		t.Fatalf("PushFile: %v", err)
	}

	if got := gh.createdRef["ref"]; got != "refs/heads/preflight/region" {
		t.Errorf("createRef ref = %v; want refs/heads/preflight/region", got)
	}
	if got := gh.createdRef["sha"]; got != "deadbeefdeadbeefdeadbeefdeadbeefdeadbeef" {
		t.Errorf("createRef sha = %v; want deadbeef...", got)
	}
}

func TestPushFile_PropagatesServerError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.Contains(r.URL.Path, "access_tokens"):
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{
				"token":      "installtok_xyz",
				"expires_at": time.Now().Add(time.Hour).UTC().Format(time.RFC3339),
			})
		case r.Method == http.MethodGet && strings.Contains(r.URL.Path, "/branches/"):
			// Branch exists; ensureBranch is a no-op so the test can
			// reach the PUT it actually wants to assert on.
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{"name": "main"})
		case r.Method == http.MethodGet:
			// Contents lookup: file does not exist.
			w.WriteHeader(http.StatusNotFound)
		default:
			// PUT contents: 500 — the actual assertion.
			http.Error(w, `{"message":"internal"}`, http.StatusInternalServerError)
		}
	}))
	t.Cleanup(srv.Close)

	c, err := New(Config{
		AppID:          1,
		InstallationID: 1,
		PrivateKeyPEM:  generateTestKey(t),
		BaseURL:        srv.URL,
		HTTPClient:     srv.Client(),
		Now:            time.Now,
	})
	if err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_, err = c.PushFile(ctx, "owner/repo", "main", "file", []byte("x"), "msg")
	if err == nil {
		t.Fatal("expected error from 500, got nil")
	}
	if !strings.Contains(err.Error(), "500") {
		t.Errorf("expected error to mention status 500, got %v", err)
	}
}
