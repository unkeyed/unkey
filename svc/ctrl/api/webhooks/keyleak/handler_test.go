package keyleak

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"crypto/x509"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"errors"
	"io"
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
	keydb "github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/email"
	"github.com/unkeyed/unkey/pkg/hash"
	"github.com/unkeyed/unkey/pkg/mysql/sqlcomment"
	"github.com/unkeyed/unkey/pkg/testutil/containers"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/pkg/webhook/verifiers/githubsecrets"
	"github.com/unkeyed/unkey/svc/ctrl/integration/seed"
	"github.com/unkeyed/unkey/svc/ctrl/internal/db"
	"github.com/unkeyed/unkey/svc/ctrl/internal/slack"
	"github.com/unkeyed/unkey/svc/ctrl/internal/workos"
	worker "github.com/unkeyed/unkey/svc/ctrl/worker/keyleak"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
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
	failingDB := &unavailableDatabase{Database: database, attempted: make(chan struct{})}
	service := worker.New(worker.Config{
		DB:      failingDB,
		Members: gatedMembers{orgID: workspace.OrgID, ready: membersReady},
		Email:   sender, SlackWebhookURL: slackServer.URL,
	})
	runtime := containers.Restate(t, service.Definition().
		ConfigureHandler("NotifyEmail", restate.WithIngressPrivate(true)).
		ConfigureHandler("NotifySlack", restate.WithIngressPrivate(true)))
	verifier, sign := newSigningFixture(t)
	handler := New(runtime.IngressClient, verifier)
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)
	tokens := []reportedToken{
		{Token: root.Key, Type: "unkey_root_key", Source: "commit", URL: "https://github.com/unkeyed/example/commit/a"},
		{Token: unknown, Type: "unkey_root_key", Source: "commit"},
		{Token: deleted.Key, Type: "unkey_root_key", Source: "commit"},
		{Token: root.Key, Type: "unkey_root_key", Source: "content", URL: "https://github.com/unkeyed/example/blob/b"},
	}
	body, err := json.Marshal(tokens)
	require.NoError(t, err)
	requestCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	t.Cleanup(cancel)
	req, err := http.NewRequestWithContext(requestCtx, http.MethodPost, server.URL+"/webhooks/github/key-leaks", bytes.NewReader(body))
	require.NoError(t, err)
	req.Header = sign(body).Header
	res, err := server.Client().Do(req)
	require.NoError(t, err, "webhook must acknowledge persistence without waiting for the database")
	t.Cleanup(func() { require.NoError(t, res.Body.Close()) })
	responseBody, err := io.ReadAll(res.Body)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, res.StatusCode)
	require.Empty(t, responseBody)
	cancel()

	select {
	case <-failingDB.attempted:
	case <-time.After(20 * time.Second):
		t.Fatal("report did not reach the database")
	}
	failingDB.recovered.Store(true)

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
	deliveredKeys := make(map[string]bool)
	select {
	case message := <-sender.delivered:
		require.Equal(t, []string{"second@example.com"}, message.To, "a failed recipient must not block others")
		require.Equal(t, "secret-scanning-key-detected", message.TemplateID)
		require.Equal(t, tokens[0].URL, message.Variables["URL"])
		require.Equal(t, tokens[0].Source, message.Variables["SOURCE"])
		require.NotEmpty(t, message.IdempotencyKey)
		deliveredKeys[message.IdempotencyKey] = true
	case <-time.After(20 * time.Second):
		t.Fatal("second recipient did not receive notification")
	}
	sender.recovered.Store(true)
	select {
	case message := <-sender.delivered:
		require.Equal(t, []string{"first@example.com"}, message.To)
		deliveredKeys[message.IdempotencyKey] = true
	case <-time.After(20 * time.Second):
		t.Fatal("failed email did not retry")
	}

	for range 2 {
		res := httptest.NewRecorder()
		handler.ServeHTTP(res, sign(body))
		require.Equal(t, http.StatusOK, res.Code, res.Body.String())
		require.Empty(t, res.Body.Bytes())
	}
	require.Empty(t, sender.delivered, "duplicate reports must not send duplicate emails")
	require.Empty(t, slackMessages, "duplicate reports must not send duplicate Slack alerts")
	key, err := database.FindKeyByID(ctx, root.KeyID)
	require.NoError(t, err)
	require.False(t, key.DeletedAtM.Valid, "migration must not introduce automatic revocation")

	t.Run("distinct leak of the same key", func(t *testing.T) {
		report := tokens[0]
		report.URL = "https://github.com/unkeyed/example/commit/another"
		body, err := json.Marshal([]reportedToken{report})
		require.NoError(t, err)
		res := httptest.NewRecorder()
		handler.ServeHTTP(res, sign(body))
		require.Equal(t, http.StatusOK, res.Code, res.Body.String())
		for range 2 {
			select {
			case message := <-sender.delivered:
				require.Equal(t, report.URL, message.Variables["URL"])
				require.NotContains(t, deliveredKeys, message.IdempotencyKey)
			case <-time.After(20 * time.Second):
				t.Fatal("a distinct report must send new notifications")
			}
		}
		select {
		case message := <-slackMessages:
			encoded, err := json.Marshal(message)
			require.NoError(t, err)
			require.Contains(t, string(encoded), report.URL)
		case <-time.After(20 * time.Second):
			t.Fatal("a distinct report must send a new Slack alert")
		}
	})

	t.Run("queue unavailable", func(t *testing.T) {
		unavailable := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusServiceUnavailable)
		}))
		t.Cleanup(unavailable.Close)
		handler := New(ingress.NewClient(unavailable.URL), verifier)
		res := httptest.NewRecorder()
		handler.ServeHTTP(res, sign(body))
		require.Equal(t, http.StatusServiceUnavailable, res.Code, "never acknowledge an unqueued leak")
	})
}

func TestKeyLeakClassifiesAllKeyStoresWithoutSlack(t *testing.T) {
	ctx := t.Context()
	database, err := db.New(containers.MySQL(t).DSN, sqlcomment.Disabled())
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, database.Close()) })
	seeder := seed.New(t, database, nil)
	seeder.Seed(ctx)
	workspace := seeder.Resources.UserWorkspace
	api := seeder.CreateAPI(ctx, seed.CreateApiRequest{WorkspaceID: workspace.ID})
	membersReady := make(chan struct{})
	close(membersReady)
	sender := &recordingSender{delivered: make(chan email.Email, 20)}
	service := worker.New(worker.Config{
		DB: database, Members: gatedMembers{orgID: workspace.OrgID, ready: membersReady}, Email: sender,
	})
	runtime := containers.Restate(t, service.Definition())
	verifier, sign := newSigningFixture(t)
	handler := New(runtime.IngressClient, verifier)
	slackCtx, cancelSlack := context.WithTimeout(ctx, 5*time.Second)
	t.Cleanup(cancelSlack)
	_, err = hydrav1.NewKeyLeakServiceIngressClient(runtime.IngressClient).NotifySlack().Request(slackCtx,
		&hydrav1.KeyLeakNotification{OrgId: workspace.OrgID})
	require.NoError(t, err, "unconfigured Slack must complete without attempting delivery")

	for _, test := range []struct {
		name       string
		legacy     bool
		customer   bool
		newStore   bool
		deletedNew bool
		matched    bool
	}{
		{name: "legacy root", legacy: true, matched: true},
		{name: "customer key", customer: true, matched: true},
		{name: "new root", newStore: true, matched: true},
		{name: "migrated root", legacy: true, newStore: true, matched: true},
		{name: "tombstoned migrated root", legacy: true, newStore: true, deletedNew: true},
		{name: "unknown"},
	} {
		t.Run(test.name, func(t *testing.T) {
			token := uid.New(uid.TestPrefix)
			keyID := uid.New(uid.KeyPrefix)
			if test.legacy || test.customer {
				params := seed.CreateKeyRequest{
					WorkspaceID: seeder.Resources.RootWorkspace.ID,
					KeySpaceID:  seeder.Resources.RootKeySpace.ID, ForWorkspaceID: &workspace.ID,
				}
				if test.customer {
					params.WorkspaceID = workspace.ID
					params.KeySpaceID = api.KeyAuthID.String
					params.ForWorkspaceID = nil
				}
				key := seeder.CreateKey(ctx, params)
				token, keyID = key.Key, key.KeyID
			}
			if test.newStore {
				require.NoError(t, keydb.Query.InsertUnkeyRootKey(ctx, database.RW(), keydb.InsertUnkeyRootKeyParams{
					ID: keyID, WorkspaceID: workspace.ID, Hash: hash.Sha256(token),
					Prefix: "test", Start: token[:8], End: token[len(token)-4:], Enabled: true,
					CreatedAt: time.Now().UnixMilli(),
				}))
				if test.deletedNew {
					_, err := keydb.Query.SoftDeleteUnkeyRootKey(ctx, database.RW(), keydb.SoftDeleteUnkeyRootKeyParams{
						ID: keyID, WorkspaceID: workspace.ID, Now: sql.NullInt64{Int64: time.Now().UnixMilli(), Valid: true},
					})
					require.NoError(t, err)
				}
			}

			report := reportedToken{Token: token, Type: "unkey_key", Source: "commit", URL: "https://github.com/unkeyed/example/commit/" + keyID}
			body, err := json.Marshal([]reportedToken{report})
			require.NoError(t, err)
			res := httptest.NewRecorder()
			handler.ServeHTTP(res, sign(body))
			require.Equal(t, http.StatusOK, res.Code, res.Body.String())
			require.Empty(t, res.Body.Bytes())

			digest := sha256.Sum256([]byte(token))
			hashed := base64.StdEncoding.EncodeToString(digest[:])
			processed, err := hydrav1.NewKeyLeakServiceIngressClient(runtime.IngressClient).Process().Request(ctx,
				&hydrav1.KeyLeakReport{Tokens: []*hydrav1.KeyLeakToken{{
					Hash: hashed, Type: report.Type, Source: report.Source, Url: report.URL,
				}}}, restate.WithIdempotencyKey(hash.Sha256(string(body))))
			require.NoError(t, err)
			if !test.matched {
				require.Empty(t, processed.GetMatchedHashes())
				return
			}
			require.Equal(t, []string{hashed}, processed.GetMatchedHashes())
			recipients := make([]string, 0, 2)
			for range 2 {
				select {
				case message := <-sender.delivered:
					require.Equal(t, report.URL, message.Variables["URL"])
					recipients = append(recipients, message.To...)
				case <-time.After(20 * time.Second):
					t.Fatal("customer email must not depend on Slack configuration")
				}
			}
			require.ElementsMatch(t, []string{"first@example.com", "second@example.com"}, recipients)
		})
	}
	require.Empty(t, sender.delivered)
}

func TestKeyLeakNoopProvidersStillDeliverSlack(t *testing.T) {
	messages := make(chan slack.Payload, 1)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var message slack.Payload
		if err := json.NewDecoder(r.Body).Decode(&message); err != nil {
			t.Error(err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		messages <- message
	}))
	t.Cleanup(server.Close)
	service := worker.New(worker.Config{
		DB:              nil,
		Members:         workos.New(""),
		Email:           email.NewResend("", ""),
		SlackWebhookURL: server.URL,
	})
	runtime := containers.Restate(t, service.Definition())
	client := hydrav1.NewKeyLeakServiceIngressClient(runtime.IngressClient)
	leak := &hydrav1.KeyLeakNotification{
		KeyId:         uid.New(uid.KeyPrefix),
		OrgId:         uid.New(uid.TestPrefix),
		WorkspaceName: "test workspace",
		Source:        "commit",
		Url:           "https://github.com/unkeyed/example/commit/test",
		Type:          "unkey_key",
		Date:          "Mon Jan 02 2006",
	}

	ctx, cancel := context.WithTimeout(t.Context(), 20*time.Second)
	t.Cleanup(cancel)
	_, err := client.Notify().Request(ctx, leak)
	require.NoError(t, err)
	_, err = client.NotifyEmail().Request(ctx, &hydrav1.KeyLeakEmailRequest{Leak: leak, Email: "test@example.com"})
	require.NoError(t, err)

	select {
	case message := <-messages:
		encoded, err := json.Marshal(message)
		require.NoError(t, err)
		require.Contains(t, string(encoded), leak.GetKeyId())
		require.Contains(t, string(encoded), leak.GetUrl())
	case <-ctx.Done():
		t.Fatal("Slack alert must not depend on email provider configuration")
	}

	leak.OrgId = ""
	for _, test := range []struct {
		name    string
		handler string
		request proto.Message
	}{
		{name: "notification requires organization", handler: "Notify", request: leak},
		{name: "Slack notification requires organization", handler: "NotifySlack", request: leak},
		{name: "email notification requires organization", handler: "NotifyEmail", request: &hydrav1.KeyLeakEmailRequest{Leak: leak, Email: "test@example.com"}},
		{name: "email notification requires leak", handler: "NotifyEmail", request: &hydrav1.KeyLeakEmailRequest{Email: "test@example.com"}},
	} {
		t.Run(test.name, func(t *testing.T) {
			body, err := protojson.Marshal(test.request)
			require.NoError(t, err)
			req, err := http.NewRequestWithContext(ctx, http.MethodPost,
				runtime.IngressURL+"/hydra.v1.KeyLeakService/"+test.handler, bytes.NewReader(body))
			require.NoError(t, err)
			req.Header.Set("Content-Type", "application/json")
			response, err := http.DefaultClient.Do(req)
			require.NoError(t, err)
			t.Cleanup(func() { require.NoError(t, response.Body.Close()) })
			require.Equal(t, http.StatusBadRequest, response.StatusCode)
		})
	}
	require.Empty(t, messages)
}

func TestKeyLeakNeverSubmitsRawTokensToRestate(t *testing.T) {
	token := reportedToken{Token: uid.New(uid.TestPrefix), Type: "unkey_key", Source: "commit"}
	body, err := json.Marshal([]reportedToken{token})
	require.NoError(t, err)
	type submission struct {
		body           []byte
		idempotencyKey string
	}
	submitted := make(chan submission, 2)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		payload, err := io.ReadAll(r.Body)
		if err != nil {
			t.Error(err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		submitted <- submission{body: payload, idempotencyKey: r.Header.Get("Idempotency-Key")}
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	t.Cleanup(server.Close)
	verifier, sign := newSigningFixture(t)
	handler := New(ingress.NewClient(server.URL), verifier)
	for range 2 {
		res := httptest.NewRecorder()
		handler.ServeHTTP(res, sign(body))
		require.Equal(t, http.StatusServiceUnavailable, res.Code)
		request := <-submitted
		require.NotContains(t, string(request.body), token.Token)
		var report hydrav1.KeyLeakReport
		require.NoError(t, json.Unmarshal(request.body, &report))
		require.Len(t, report.GetTokens(), 1)
		digest := sha256.Sum256([]byte(token.Token))
		require.Equal(t, base64.StdEncoding.EncodeToString(digest[:]), report.GetTokens()[0].GetHash())
		digest = sha256.Sum256(body)
		require.Equal(t, base64.StdEncoding.EncodeToString(digest[:]), request.idempotencyKey)
	}
}

func TestKeyLeakRejectsInvalidReports(t *testing.T) {
	verifier, sign := newSigningFixture(t)
	handler := New(nil, verifier)
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
	for _, body := range [][]byte{[]byte("["), nil} {
		res := httptest.NewRecorder()
		handler.ServeHTTP(res, sign(body))
		require.Equal(t, http.StatusBadRequest, res.Code)
	}
	res := httptest.NewRecorder()
	handler.ServeHTTP(res, sign([]byte(strings.Repeat("x", bodyBytesMax+1))))
	require.Equal(t, http.StatusRequestEntityTooLarge, res.Code)
	res = httptest.NewRecorder()
	handler.ServeHTTP(res, httptest.NewRequest(http.MethodPost, "/", strings.NewReader("[]")))
	require.Equal(t, http.StatusBadRequest, res.Code)
	req := sign([]byte("[]"))
	req.Header.Set("Github-Public-Key-Signature", "forged")
	res = httptest.NewRecorder()
	handler.ServeHTTP(res, req)
	require.Equal(t, http.StatusUnauthorized, res.Code)
	res = httptest.NewRecorder()
	handler.ServeHTTP(res, sign([]byte("[]")))
	require.Equal(t, http.StatusOK, res.Code)
	require.Empty(t, res.Body.Bytes())
}

type unavailableDatabase struct {
	db.Database
	attempted chan struct{}
	once      sync.Once
	recovered atomic.Bool
}

func (d *unavailableDatabase) FindLeakedKeysByHashes(ctx context.Context, params db.FindLeakedKeysByHashesParams) ([]db.FindLeakedKeysByHashesRow, error) {
	recovered := d.recovered.Load()
	d.once.Do(func() { close(d.attempted) })
	if !recovered {
		return nil, errors.New("database unavailable")
	}
	return d.Database.FindLeakedKeysByHashes(ctx, params)
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

func newSigningFixture(t *testing.T) (*githubsecrets.Verifier, func([]byte) *http.Request) {
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
	verifier, err := githubsecrets.New(server.URL, clock.New())
	require.NoError(t, err)
	t.Cleanup(verifier.Close)
	return verifier, func(body []byte) *http.Request {
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
