package keyleak

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	restate "github.com/restatedev/sdk-go"
	"github.com/restatedev/sdk-go/ingress"
	"github.com/stretchr/testify/require"
	hydrav1 "github.com/unkeyed/unkey/gen/proto/hydra/v1"
	"github.com/unkeyed/unkey/pkg/clock"
	"github.com/unkeyed/unkey/pkg/email"
	"github.com/unkeyed/unkey/pkg/mysql/sqlcomment"
	"github.com/unkeyed/unkey/pkg/testutil/containers"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/ctrl/integration/seed"
	"github.com/unkeyed/unkey/svc/ctrl/internal/db"
	"github.com/unkeyed/unkey/svc/ctrl/internal/githubsignature"
	"github.com/unkeyed/unkey/svc/ctrl/internal/slack"
	worker "github.com/unkeyed/unkey/svc/ctrl/worker/keyleak"
)

func TestKeyLeakDurableNotifications(t *testing.T) {
	ctx := t.Context()
	database, err := db.New(containers.MySQL(t).DSN, sqlcomment.Disabled())
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, database.Close()) })
	seeder := seed.New(t, database, nil)
	seeder.Seed(ctx)
	workspace := seeder.Resources.UserWorkspace
	root := seeder.CreateKey(ctx, seed.CreateKeyRequest{
		WorkspaceID: seeder.Resources.RootWorkspace.ID, KeySpaceID: seeder.Resources.RootKeySpace.ID,
		ForWorkspaceID: &workspace.ID,
	})
	deleted := seeder.CreateKey(ctx, seed.CreateKeyRequest{
		WorkspaceID: seeder.Resources.RootWorkspace.ID, KeySpaceID: seeder.Resources.RootKeySpace.ID,
		ForWorkspaceID: &workspace.ID, Deleted: true,
	})
	ordinary := seeder.CreateKey(ctx, seed.CreateKeyRequest{
		WorkspaceID: seeder.Resources.RootWorkspace.ID, KeySpaceID: seeder.Resources.RootKeySpace.ID,
	})
	unknown := uid.New(uid.TestPrefix)

	var slackAttempts atomic.Int32
	slackMessages := make(chan slack.Payload, 10)
	slackServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if slackAttempts.Add(1) == 1 {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		var payload slack.Payload
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Error(err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		slackMessages <- payload
	}))
	t.Cleanup(slackServer.Close)
	membersReady := make(chan struct{})
	releaseMembers := sync.OnceFunc(func() { close(membersReady) })
	t.Cleanup(releaseMembers)
	sender := &recordingSender{delivered: make(chan email.Email, 10), failedRecipient: "first@example.com"}
	service := worker.New(worker.Config{
		Members: gatedMembers{orgID: workspace.OrgID, ready: membersReady},
		Email:   sender, SlackWebhookURL: slackServer.URL,
	})
	runtime := containers.Restate(t, hydrav1.NewKeyLeakServiceServer(service).
		ConfigureHandler("NotifyEmail", restate.WithIngressPrivate(true)).
		ConfigureHandler("NotifySlack", restate.WithIngressPrivate(true)))
	verifier, sign := newSigningFixture(t)
	handler := New(database, runtime.IngressClient, verifier, clock.New())
	tokens := []reportedToken{
		{Token: root.Key, Type: "unkey_root_key", Source: "commit", URL: "https://github.com/unkeyed/example/commit/a"},
		{Token: unknown, Type: "unkey_root_key", Source: "commit"},
		{Token: deleted.Key, Type: "unkey_root_key", Source: "commit"},
		{Token: root.Key, Type: "unkey_root_key", Source: "content", URL: "https://github.com/unkeyed/example/blob/b"},
		{Token: ordinary.Key, Type: "unkey_root_key", Source: "commit"},
	}
	body, err := json.Marshal(tokens)
	require.NoError(t, err)
	for range 2 {
		res := httptest.NewRecorder()
		handler.ServeHTTP(res, sign(body))
		require.Equal(t, http.StatusCreated, res.Code, res.Body.String())
		var got []result
		require.NoError(t, json.Unmarshal(res.Body.Bytes(), &got))
		want := make([]result, len(tokens))
		for i, token := range tokens {
			if i == 0 || i == 3 {
				want[i] = result{TokenRaw: root.Key, TokenType: token.Type, Label: "true_positive"}
			} else {
				digest := sha256.Sum256([]byte(token.Token))
				want[i] = result{TokenHash: base64.StdEncoding.EncodeToString(digest[:]), TokenType: token.Type, Label: "false_positive"}
			}
		}
		require.Equal(t, want, got)
	}
	select {
	case message := <-slackMessages:
		encoded, err := json.Marshal(message)
		require.NoError(t, err)
		require.Contains(t, string(encoded), root.KeyID)
		require.Contains(t, string(encoded), workspace.Name)
		require.Contains(t, string(encoded), workspace.OrgID)
		require.NotContains(t, string(encoded), root.Key)
	case <-time.After(20 * time.Second):
		t.Fatal("Slack must retry and deliver while recipient lookup is blocked")
	}
	releaseMembers()
	select {
	case message := <-sender.delivered:
		require.Equal(t, []string{"second@example.com"}, message.To, "a failed recipient must not block others")
		require.Equal(t, "secret-scanning-key-detected", message.TemplateID)
		require.Equal(t, tokens[0].URL, message.Variables["URL"])
		require.Equal(t, tokens[0].Source, message.Variables["SOURCE"])
		require.NotEmpty(t, message.IdempotencyKey)
	case <-time.After(20 * time.Second):
		t.Fatal("second recipient did not receive notification")
	}
	sender.recovered.Store(true)
	select {
	case message := <-sender.delivered:
		require.Equal(t, []string{"first@example.com"}, message.To)
	case <-time.After(20 * time.Second):
		t.Fatal("failed email did not retry")
	}
	_, err = hydrav1.NewKeyLeakServiceIngressClient(runtime.IngressClient).Notify().Request(ctx,
		&hydrav1.KeyLeakNotification{}, restate.WithIdempotencyKey("leaked-key:"+root.KeyID))
	require.NoError(t, err)
	require.Empty(t, sender.delivered, "duplicate reports must not send duplicate emails")
	require.Empty(t, slackMessages, "duplicate reports must not send duplicate Slack alerts")
	key, err := database.FindKeyByID(ctx, root.KeyID)
	require.NoError(t, err)
	require.False(t, key.DeletedAtM.Valid, "migration must not introduce automatic revocation")

	t.Run("queue unavailable", func(t *testing.T) {
		unavailable := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusServiceUnavailable)
		}))
		t.Cleanup(unavailable.Close)
		handler := New(database, ingress.NewClient(unavailable.URL), verifier, clock.New())
		res := httptest.NewRecorder()
		handler.ServeHTTP(res, sign(body))
		require.Equal(t, http.StatusServiceUnavailable, res.Code, "never acknowledge an unqueued leak")
	})
}

func TestKeyLeakRejectsInvalidReports(t *testing.T) {
	verifier, sign := newSigningFixture(t)
	handler := New(nil, nil, verifier, clock.New())
	for _, test := range []struct {
		name string
		body any
	}{
		{name: "not an array", body: reportedToken{Token: "token"}},
		{name: "null", body: nil},
		{name: "null member", body: []*reportedToken{nil}},
		{name: "missing token", body: []reportedToken{{Type: "type", Source: "commit"}}},
		{name: "numeric token", body: []map[string]any{{"token": 42, "type": "type", "source": "commit"}}},
	} {
		t.Run(test.name, func(t *testing.T) {
			body, err := json.Marshal(test.body)
			require.NoError(t, err)
			res := httptest.NewRecorder()
			handler.ServeHTTP(res, sign(body))
			require.Equal(t, http.StatusBadRequest, res.Code)
		})
	}
	for _, body := range [][]byte{[]byte("["), nil, []byte(strings.Repeat("x", bodyBytesMax+1))} {
		res := httptest.NewRecorder()
		handler.ServeHTTP(res, sign(body))
		require.Equal(t, http.StatusBadRequest, res.Code)
	}
	res := httptest.NewRecorder()
	handler.ServeHTTP(res, httptest.NewRequest(http.MethodPost, "/", strings.NewReader("[]")))
	require.Equal(t, http.StatusBadRequest, res.Code)
	req := sign([]byte("[]"))
	req.Header.Set("Github-Public-Key-Signature", "forged")
	res = httptest.NewRecorder()
	handler.ServeHTTP(res, req)
	require.Equal(t, http.StatusUnauthorized, res.Code)
	res = httptest.NewRecorder()
	handler.ServeHTTP(res, sign([]byte("[]")))
	require.Equal(t, http.StatusCreated, res.Code)
	var results []result
	require.NoError(t, json.Unmarshal(res.Body.Bytes(), &results))
	require.NotNil(t, results)
	require.Empty(t, results)
}

type gatedMembers struct {
	orgID string
	ready <-chan struct{}
}

func (m gatedMembers) MemberEmails(ctx context.Context, orgID string) ([]string, error) {
	if orgID != m.orgID {
		return nil, errors.New("wrong organization")
	}
	select {
	case <-m.ready:
		return []string{"first@example.com", "second@example.com"}, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

type recordingSender struct {
	delivered       chan email.Email
	failedRecipient string
	recovered       atomic.Bool
}

func (s *recordingSender) Send(_ context.Context, message email.Email) error {
	if message.To[0] == s.failedRecipient && !s.recovered.Load() {
		return errors.New("temporary email failure")
	}
	s.delivered <- message
	return nil
}

func newSigningFixture(t *testing.T) (*githubsignature.Verifier, func([]byte) *http.Request) {
	t.Helper()
	privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)
	publicKey, err := x509.MarshalPKIXPublicKey(&privateKey.PublicKey)
	require.NoError(t, err)
	keyID := uid.New(uid.TestPrefix)
	keys := struct {
		PublicKeys []struct {
			KeyIdentifier string `json:"key_identifier"`
			Key           string `json:"key"`
		} `json:"public_keys"`
	}{}
	keys.PublicKeys = append(keys.PublicKeys, struct {
		KeyIdentifier string `json:"key_identifier"`
		Key           string `json:"key"`
	}{KeyIdentifier: keyID, Key: string(pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: publicKey}))})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		if err := json.NewEncoder(w).Encode(keys); err != nil {
			t.Error(err)
		}
	}))
	t.Cleanup(server.Close)
	return githubsignature.New(server.URL, clock.New()), func(body []byte) *http.Request {
		t.Helper()
		digest := sha256.Sum256(body)
		signature, err := ecdsa.SignASN1(rand.Reader, privateKey, digest[:])
		require.NoError(t, err)
		req := httptest.NewRequest(http.MethodPost, "/webhooks/github/key-leaks", bytes.NewReader(body))
		req.Header.Set("Github-Public-Key-Identifier", keyID)
		req.Header.Set("Github-Public-Key-Signature", base64.StdEncoding.EncodeToString(signature))
		return req
	}
}
