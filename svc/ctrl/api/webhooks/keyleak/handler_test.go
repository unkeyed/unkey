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
	"net/http/httputil"
	"net/url"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	restate "github.com/restatedev/sdk-go"
	"github.com/restatedev/sdk-go/ingress"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	hydrav1 "github.com/unkeyed/unkey/gen/proto/hydra/v1"
	"github.com/unkeyed/unkey/pkg/auditlog"
	"github.com/unkeyed/unkey/pkg/clock"
	keydb "github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/email"
	"github.com/unkeyed/unkey/pkg/hash"
	"github.com/unkeyed/unkey/pkg/mysql/sqlcomment"
	"github.com/unkeyed/unkey/pkg/testutil/containers"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/pkg/webhook/verifiers/githubsecrets"
	"github.com/unkeyed/unkey/pkg/workos"
	"github.com/unkeyed/unkey/svc/ctrl/integration/seed"
	"github.com/unkeyed/unkey/svc/ctrl/internal/auditlogs"
	"github.com/unkeyed/unkey/svc/ctrl/internal/db"
	"github.com/unkeyed/unkey/svc/ctrl/internal/slack"
	worker "github.com/unkeyed/unkey/svc/ctrl/worker/keyleak"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

func TestKeyLeakDurableNotifications(t *testing.T) {
	ctx := t.Context()
	database, err := db.New(containers.MySQL(t).DSN, sqlcomment.Disabled())
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, database.Close()) })
	auditlogSvc, err := auditlogs.New(auditlogs.Config{DB: database})
	require.NoError(t, err)
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
	service := worker.New(worker.Config{
		DB: database, Auditlogs: auditlogSvc,
		Members: gatedMembers{orgID: workspace.OrgID, ready: membersReady},
		Email:   sender, SlackWebhookURL: slackServer.URL,
	})
	runtime := containers.Restate(t, service.Definition().
		ConfigureHandler("NotifyEmail", restate.WithIngressPrivate(true)).
		ConfigureHandler("NotifySlack", restate.WithIngressPrivate(true)))
	verifier, sign := newSigningFixture(t)
	handler := New(Config{Restate: runtime.IngressClient, Database: database, Verifier: verifier})
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
	require.NoError(t, err, "webhook must acknowledge persistence without waiting for notifications")
	t.Cleanup(func() { require.NoError(t, res.Body.Close()) })
	responseBody, err := io.ReadAll(res.Body)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, res.StatusCode)
	require.Empty(t, responseBody)
	cancel()

	require.EventuallyWithT(t, func(c *assert.CollectT) {
		disabled, err := database.FindKeyByID(ctx, root.KeyID)
		require.NoError(c, err)
		require.False(c, disabled.Enabled, "disable the leaked key before recipient lookup completes")
	}, 20*time.Second, 50*time.Millisecond)

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
	require.False(t, key.DeletedAtM.Valid, "disable the key without deleting it")
	require.Len(t, leakEvents(t, database, workspace.ID, root.KeyID), 1, "duplicate reports must not repeat the disable audit")

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

	t.Run("retry after lost batch acknowledgment", func(t *testing.T) {
		second := seeder.CreateKey(ctx, seed.CreateKeyRequest{
			WorkspaceID: seeder.Resources.RootWorkspace.ID, KeySpaceID: seeder.Resources.RootKeySpace.ID,
			ForWorkspaceID: &workspace.ID,
		})
		reports := []reportedToken{
			{Token: root.Key, Type: "unkey_root_key", Source: "commit", URL: "https://github.com/unkeyed/example/commit/first"},
			{Token: second.Key, Type: "unkey_root_key", Source: "content", URL: "https://github.com/unkeyed/example/blob/second"},
		}
		body, err := json.Marshal(reports)
		require.NoError(t, err)
		target, err := url.Parse(runtime.IngressURL)
		require.NoError(t, err)
		proxy := httputil.NewSingleHostReverseProxy(target)
		var attempts atomic.Int32
		idempotencyKeys := make(chan string, 6)
		interrupted := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			idempotencyKeys <- r.Header.Get("Idempotency-Key")
			if attempts.Add(1) == 1 {
				accepted := httptest.NewRecorder()
				proxy.ServeHTTP(accepted, r)
				if accepted.Code != http.StatusAccepted && accepted.Code != http.StatusOK {
					t.Errorf("enqueue failed: %d", accepted.Code)
				}
				w.WriteHeader(http.StatusServiceUnavailable)
				return
			}
			proxy.ServeHTTP(w, r)
		}))
		t.Cleanup(interrupted.Close)
		handler := New(Config{Restate: ingress.NewClient(interrupted.URL), Database: database, Verifier: verifier})
		res := httptest.NewRecorder()
		handler.ServeHTTP(res, sign(body))
		require.Equal(t, http.StatusServiceUnavailable, res.Code)
		require.Len(t, idempotencyKeys, 1)
		firstKey := <-idempotencyKeys
		require.NotEmpty(t, firstKey)

		for range 2 {
			res := httptest.NewRecorder()
			handler.ServeHTTP(res, sign(body))
			require.Equal(t, http.StatusOK, res.Code, res.Body.String())
			require.Len(t, idempotencyKeys, 1)
			require.Equal(t, firstKey, <-idempotencyKeys)
		}
		var deliveries []string
		for range 4 {
			select {
			case message := <-sender.delivered:
				deliveries = append(deliveries, message.To[0]+":"+message.Variables["URL"])
			case <-time.After(20 * time.Second):
				t.Fatal("retry must deliver both keys' notifications")
			}
		}
		require.ElementsMatch(t, []string{
			"first@example.com:" + reports[0].URL, "second@example.com:" + reports[0].URL,
			"first@example.com:" + reports[1].URL, "second@example.com:" + reports[1].URL,
		}, deliveries)
		for range 2 {
			select {
			case <-slackMessages:
			case <-time.After(20 * time.Second):
				t.Fatal("retry must deliver both Slack alerts")
			}
		}
		require.Empty(t, sender.delivered)
		require.Empty(t, slackMessages)
	})

	t.Run("queue unavailable", func(t *testing.T) {
		unavailable := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusServiceUnavailable)
		}))
		t.Cleanup(unavailable.Close)
		handler := New(Config{Restate: ingress.NewClient(unavailable.URL), Database: database, Verifier: verifier})
		var reports []reportedToken
		var keyIDs []string
		for range 2 {
			key := seeder.CreateKey(ctx, seed.CreateKeyRequest{
				WorkspaceID: seeder.Resources.RootWorkspace.ID, KeySpaceID: seeder.Resources.RootKeySpace.ID,
				ForWorkspaceID: &workspace.ID,
			})
			reports = append(reports, reportedToken{Token: key.Key, Type: "unkey_root_key", Source: "commit"})
			keyIDs = append(keyIDs, key.KeyID)
		}
		body, err := json.Marshal(reports)
		require.NoError(t, err)
		res := httptest.NewRecorder()
		handler.ServeHTTP(res, sign(body))
		require.Equal(t, http.StatusServiceUnavailable, res.Code, "never acknowledge an unqueued leak")
		for _, keyID := range keyIDs {
			key, err := database.FindKeyByID(ctx, keyID)
			require.NoError(t, err)
			require.True(t, key.Enabled, "the webhook must not write before Restate accepts the batch")
			require.Empty(t, leakEvents(t, database, workspace.ID, keyID))
		}

		body, err = json.Marshal([]reportedToken{tokens[1], tokens[2]})
		require.NoError(t, err)
		res = httptest.NewRecorder()
		handler.ServeHTTP(res, sign(body))
		require.Equal(t, http.StatusOK, res.Code, "false positives must not depend on Restate availability")
		require.Empty(t, res.Body.Bytes())
	})

	t.Run("database unavailable", func(t *testing.T) {
		handler := New(Config{Database: &unavailableDatabase{}, Verifier: verifier})
		res := httptest.NewRecorder()
		handler.ServeHTTP(res, sign(body))
		require.Equal(t, http.StatusServiceUnavailable, res.Code, "lookup failure must not enqueue or acknowledge the report")
	})
}

func TestKeyLeakClassifiesAllKeyStoresWithoutSlack(t *testing.T) {
	ctx := t.Context()
	database, err := db.New(containers.MySQL(t).DSN, sqlcomment.Disabled())
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, database.Close()) })
	auditlogSvc, err := auditlogs.New(auditlogs.Config{DB: database})
	require.NoError(t, err)
	seeder := seed.New(t, database, nil)
	seeder.Seed(ctx)
	workspace := seeder.Resources.UserWorkspace
	api := seeder.CreateAPI(ctx, seed.CreateApiRequest{WorkspaceID: workspace.ID})
	membersReady := make(chan struct{})
	close(membersReady)
	sender := &recordingSender{delivered: make(chan email.Email, 20)}
	service := worker.New(worker.Config{
		DB: database, Auditlogs: auditlogSvc,
		Members: gatedMembers{orgID: workspace.OrgID, ready: membersReady}, Email: sender,
	})
	runtime := containers.Restate(t, service.Definition())
	verifier, sign := newSigningFixture(t)
	handler := New(Config{Restate: runtime.IngressClient, Database: database, Verifier: verifier})
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
			if !test.matched {
				New(Config{Database: database, Verifier: verifier}).ServeHTTP(res, sign(body))
				require.Equal(t, http.StatusOK, res.Code, res.Body.String())
				require.Empty(t, res.Body.Bytes())
				require.Empty(t, leakEvents(t, database, workspace.ID, keyID))
				return
			}
			handler.ServeHTTP(res, sign(body))
			require.Equal(t, http.StatusOK, res.Code, res.Body.String())
			require.Empty(t, res.Body.Bytes())
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

			if test.legacy || test.customer {
				key, err := database.FindKeyByID(ctx, keyID)
				require.NoError(t, err)
				require.False(t, key.Enabled)
				require.False(t, key.DeletedAtM.Valid)
			}
			if test.newStore {
				key, err := keydb.Query.FindUnkeyRootKeyByID(ctx, database.RW(), keyID)
				require.NoError(t, err)
				require.False(t, key.Enabled)
			}
			logs := leakEvents(t, database, workspace.ID, keyID)
			require.Len(t, logs, 1)
			wantEvent := "rootKey.leaked"
			if test.customer {
				wantEvent = "key.leaked"
			}
			require.Equal(t, wantEvent, logs[0].Event, "classify the key from storage, not the reported type")
			require.Equal(t, "system", logs[0].Actor.Type)
			require.Equal(t, "github_secret_scanning", logs[0].Actor.ID)
			require.Equal(t, map[string]any{"source": "github"}, logs[0].Actor.Meta)
			require.Contains(t, logs[0].Description, "Automatically disabled leaked key")
			require.Len(t, logs[0].Targets, 1)
			require.Equal(t, map[string]any{"source": report.Source, "url": report.URL, "enabled": false}, logs[0].Targets[0].Meta)
			encoded, err := json.Marshal(logs)
			require.NoError(t, err)
			require.NotContains(t, string(encoded), token)
			require.Empty(t, leakEvents(t, database, seeder.Resources.RootWorkspace.ID, keyID), "root-key audits belong to the customer workspace")
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
		Members:         workos.New("", ""),
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
	token := reportedToken{Token: root.Key, Type: "unkey_key", Source: "commit", URL: "https://github.com/unkeyed/example/commit/test"}
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
	handler := New(Config{Restate: ingress.NewClient(server.URL), Database: database, Verifier: verifier})
	var idempotencyKey string
	for range 2 {
		res := httptest.NewRecorder()
		handler.ServeHTTP(res, sign(body))
		require.Equal(t, http.StatusServiceUnavailable, res.Code)
		request := <-submitted
		require.NotContains(t, string(request.body), token.Token)
		var batch hydrav1.KeyLeakBatchRequest
		require.NoError(t, protojson.Unmarshal(request.body, &batch))
		require.Len(t, batch.GetKeys(), 1)
		require.Equal(t, hash.Sha256(root.Key), batch.GetKeys()[0].GetHash())
		require.Equal(t, workspace.ID, batch.GetKeys()[0].GetWorkspaceId())
		require.True(t, batch.GetKeys()[0].GetRootKey())
		notification := batch.GetKeys()[0].GetNotification()
		require.Equal(t, root.KeyID, notification.GetKeyId())
		require.Equal(t, workspace.OrgID, notification.GetOrgId())
		require.Equal(t, workspace.Name, notification.GetWorkspaceName())
		require.Equal(t, token.Type, notification.GetType())
		require.Equal(t, token.Source, notification.GetSource())
		require.Equal(t, token.URL, notification.GetUrl())
		_, err := time.Parse("Mon Jan 02 2006", notification.GetDate())
		require.NoError(t, err)
		require.NotEmpty(t, request.idempotencyKey)
		require.NotContains(t, request.idempotencyKey, token.Token)
		if idempotencyKey == "" {
			idempotencyKey = request.idempotencyKey
		}
		require.Equal(t, idempotencyKey, request.idempotencyKey)
	}
}

func TestKeyLeakRetriesPartiallyQueuedBatches(t *testing.T) {
	ctx := t.Context()
	database, err := db.New(containers.MySQL(t).DSN, sqlcomment.Disabled())
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, database.Close()) })
	auditlogSvc, err := auditlogs.New(auditlogs.Config{DB: database})
	require.NoError(t, err)
	seeder := seed.New(t, database, nil)
	seeder.Seed(ctx)
	workspace := seeder.Resources.UserWorkspace
	var reports []reportedToken
	var keyIDs []string
	for range 101 {
		key := seeder.CreateKey(ctx, seed.CreateKeyRequest{
			WorkspaceID: seeder.Resources.RootWorkspace.ID, KeySpaceID: seeder.Resources.RootKeySpace.ID,
			ForWorkspaceID: &workspace.ID,
		})
		reports = append(reports, reportedToken{Token: key.Key, Type: "unkey_root_key", Source: "commit"})
		keyIDs = append(keyIDs, key.KeyID)
	}
	service := worker.New(worker.Config{
		DB: database, Auditlogs: auditlogSvc, Members: workos.New("", ""), Email: email.NewNoop(),
	})
	runtime := containers.Restate(t, service.Definition())
	target, err := url.Parse(runtime.IngressURL)
	require.NoError(t, err)
	proxy := httputil.NewSingleHostReverseProxy(target)
	var attempts atomic.Int32
	idempotencyKeys := make(chan string, 4)
	interrupted := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		idempotencyKeys <- r.Header.Get("Idempotency-Key")
		if attempts.Add(1) == 2 {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		proxy.ServeHTTP(w, r)
	}))
	t.Cleanup(interrupted.Close)
	verifier, sign := newSigningFixture(t)
	handler := New(Config{Restate: ingress.NewClient(interrupted.URL), Database: database, Verifier: verifier})
	body, err := json.Marshal(reports)
	require.NoError(t, err)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, sign(body))
	require.Equal(t, http.StatusServiceUnavailable, response.Code)
	require.Len(t, idempotencyKeys, 2)
	firstKeys := []string{<-idempotencyKeys, <-idempotencyKeys}
	require.NotEqual(t, firstKeys[0], firstKeys[1])

	response = httptest.NewRecorder()
	handler.ServeHTTP(response, sign(body))
	require.Equal(t, http.StatusOK, response.Code)
	require.Len(t, idempotencyKeys, 2)
	require.Equal(t, firstKeys, []string{<-idempotencyKeys, <-idempotencyKeys})
	require.EventuallyWithT(t, func(c *assert.CollectT) {
		rows, err := database.ListClickhouseOutboxByWorkspace(ctx, workspace.ID)
		require.NoError(c, err)
		require.Len(c, rows, 101)
	}, 20*time.Second, 50*time.Millisecond)
	for _, keyID := range keyIDs {
		key, err := database.FindKeyByID(ctx, keyID)
		require.NoError(t, err)
		require.False(t, key.Enabled)
		require.Len(t, leakEvents(t, database, workspace.ID, keyID), 1)
	}
}

func TestKeyLeakRejectsInvalidReports(t *testing.T) {
	verifier, sign := newSigningFixture(t)
	handler := New(Config{Verifier: verifier})
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
	body, err := json.Marshal([]reportedToken{{Token: uid.New(uid.TestPrefix), Type: "unkey_key", Source: "commit"}})
	require.NoError(t, err)
	req := sign(body)
	req.Header.Set("Github-Public-Key-Signature", "forged")
	res = httptest.NewRecorder()
	handler.ServeHTTP(res, req)
	require.Equal(t, http.StatusUnauthorized, res.Code)
	res = httptest.NewRecorder()
	handler.ServeHTTP(res, sign([]byte("[]")))
	require.Equal(t, http.StatusOK, res.Code)
	require.Empty(t, res.Body.Bytes())
}

func leakEvents(t *testing.T, database db.Database, workspaceID, keyID string) []auditlog.Event {
	t.Helper()
	rows, err := database.ListClickhouseOutboxByWorkspace(t.Context(), workspaceID)
	require.NoError(t, err)
	var events []auditlog.Event
	for _, row := range rows {
		var event auditlog.Event
		require.NoError(t, json.Unmarshal(row.Payload, &event))
		for _, target := range event.Targets {
			if target.ID == keyID {
				events = append(events, event)
			}
		}
	}
	return events
}

type unavailableDatabase struct {
	db.Database
}

func (d *unavailableDatabase) FindLeakedKeysByHashes(context.Context, db.FindLeakedKeysByHashesParams) ([]db.FindLeakedKeysByHashesRow, error) {
	return nil, errors.New("database unavailable")
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
