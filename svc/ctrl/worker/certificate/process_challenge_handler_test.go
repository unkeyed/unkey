package certificate

import (
	"context"
	"crypto"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	legoacme "github.com/go-acme/lego/v4/acme"
	"github.com/go-acme/lego/v4/lego"
	"github.com/go-acme/lego/v4/registration"
	"github.com/prometheus/client_golang/prometheus"
	prometheuspb "github.com/prometheus/client_model/go"
	restate "github.com/restatedev/sdk-go"
	"github.com/restatedev/sdk-go/x/mocks"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	hydrav1 "github.com/unkeyed/unkey/gen/proto/hydra/v1"
	"github.com/unkeyed/unkey/svc/ctrl/internal/db"
	"github.com/unkeyed/unkey/svc/ctrl/pkg/metrics"
)

func TestProcessChallengeReturnsFailureAndMarksChallengeFailed(t *testing.T) {
	ctx := mocks.NewMockContext(t)
	ctx.EXPECT().Log().Return(slog.Default()).Once()
	database := &failedChallengeDB{}
	svc := New(Config{DB: database})
	expectRun(t, ctx, "resolve domain", db.CustomDomain{ID: "domain_one", Domain: "one.example.com"}, nil).Once()
	expectRun(t, ctx, "claim challenge", restate.Void{}, nil).Once()
	expected := restate.TerminalErrorf("ACME authorization failed")
	expectRun(t, ctx, "obtain certificate", certificateAttempt{}, expected,
		mock.Anything, mock.Anything, mock.Anything, mock.Anything).Once()
	ctx.EXPECT().Run(mock.Anything, mock.Anything, restate.WithName("mark failed")).
		RunAndReturn(func(fn func(restate.RunContext) (any, error), _ any, _ ...restate.RunOption) restate.TerminalError {
			_, err := fn(ctx)
			return restate.AsTerminalError(err)
		}).Once()

	counter := metrics.CertificateChallengesTotal.WithLabelValues("failed")
	before := counterValue(t, counter)
	response, err := svc.ProcessChallenge(restate.WithMockContext(ctx), &hydrav1.ProcessChallengeRequest{Domain: "one.example.com"})
	require.Nil(t, response)
	require.ErrorIs(t, err, expected)
	require.True(t, restate.IsTerminalError(err))
	require.Equal(t, "domain_one", database.updated.DomainID)
	require.Equal(t, db.AcmeChallengesStatusFailed, database.updated.Status)
	require.Equal(t, before+1, counterValue(t, counter))
}

func TestProcessChallengeReplaysPersistedCertificateWithoutCountingAgain(t *testing.T) {
	ctx := mocks.NewMockContext(t)
	ctx.EXPECT().Log().Return(slog.Default()).Once()
	expectRun(t, ctx, "resolve domain", db.CustomDomain{ID: "domain_one", Domain: "one.example.com"}, nil).Once()
	expectRun(t, ctx, "claim challenge", restate.Void{}, nil).Once()
	expectRun(t, ctx, "obtain certificate", certificateAttempt{EncryptedCertificate: EncryptedCertificate{CertificateID: "new_id"}}, nil,
		mock.Anything, mock.Anything, mock.Anything, mock.Anything).Once()
	expectRun(t, ctx, "persist certificate", "existing_id", nil).Once()
	expectRun(t, ctx, "mark verified", restate.Void{}, nil).Once()

	svc := New(Config{})
	counter := metrics.CertificateChallengesTotal.WithLabelValues("verified")
	before := counterValue(t, counter)
	response, err := svc.ProcessChallenge(restate.WithMockContext(ctx), &hydrav1.ProcessChallengeRequest{Domain: "one.example.com"})
	require.NoError(t, err)
	require.Equal(t, "existing_id", response.GetCertificateId())
	require.Equal(t, before, counterValue(t, counter))
}

func TestRateLimitResponseJournalsBoundedDelay(t *testing.T) {
	for _, test := range []struct {
		name       string
		retryAfter string
		delay      time.Duration
	}{
		{name: "missing", delay: time.Minute},
		{name: "invalid", retryAfter: "not-a-date", delay: time.Minute},
		{name: "past", retryAfter: time.Now().Add(-time.Hour).UTC().Format(http.TimeFormat), delay: time.Minute},
		{name: "buffer", retryAfter: time.Now().Add(3 * time.Minute).UTC().Format(http.TimeFormat), delay: 4 * time.Minute},
		{name: "cap", retryAfter: time.Now().Add(3 * time.Hour).UTC().Format(http.TimeFormat), delay: 2 * time.Hour},
	} {
		t.Run(test.name, func(t *testing.T) {
			client := acmeClientReturning(t, legoacme.ProblemDetails{
				HTTPStatus: http.StatusTooManyRequests,
				Type:       "urn:ietf:params:acme:error:rateLimited",
				Detail:     "test issuance quota exceeded",
			}, test.retryAfter)
			ctx := mocks.NewMockContext(t)
			ctx.EXPECT().Log().Return(slog.Default()).Once()
			svc := New(Config{})
			counter := metrics.CertificateIssuanceAttemptsTotal.WithLabelValues("rate_limited")
			before := counterValue(t, counter)
			result, err := svc.obtainCertificate(ctx, client, db.CustomDomain{Domain: "one.example.com"})
			require.NoError(t, err)
			require.Empty(t, result.CertificateID)
			require.InDelta(t, test.delay, result.RetryDelay, float64(2*time.Second))
			require.GreaterOrEqual(t, result.RetryDelay, time.Minute)
			require.LessOrEqual(t, result.RetryDelay, 2*time.Hour)
			require.Equal(t, before+1, counterValue(t, counter))

			data, err := json.Marshal(result)
			require.NoError(t, err)
			var replayed certificateAttempt
			require.NoError(t, json.Unmarshal(data, &replayed))
			require.Equal(t, result, replayed)
		})
	}
}

func TestRateLimitRetryUsesJournaledDelayAndRecovers(t *testing.T) {
	ctx := mocks.NewMockContext(t)
	first := expectRun(t, ctx, "obtain certificate", certificateAttempt{RetryDelay: 83 * time.Second}, nil,
		mock.Anything, mock.Anything, mock.Anything, mock.Anything).Once()
	sleep := ctx.EXPECT().Sleep(83 * time.Second).Return(nil).Once()
	second := expectRun(t, ctx, "obtain certificate", certificateAttempt{EncryptedCertificate: EncryptedCertificate{CertificateID: "issued"}}, nil,
		mock.Anything, mock.Anything, mock.Anything, mock.Anything).Once()
	mock.InOrder(first, sleep, second)

	svc := New(Config{})
	cert, err := svc.obtainWithRetry(restate.WithMockContext(ctx), db.CustomDomain{Domain: "one.example.com"})
	require.NoError(t, err)
	require.Equal(t, "issued", cert.CertificateID)
}

func TestRateLimitRetryBudgetReturnsTerminalFailure(t *testing.T) {
	ctx := mocks.NewMockContext(t)
	expectRun(t, ctx, "obtain certificate", certificateAttempt{RetryDelay: time.Minute}, nil,
		mock.Anything, mock.Anything, mock.Anything, mock.Anything).Times(4)
	ctx.EXPECT().Sleep(time.Minute).Return(nil).Times(3)

	svc := New(Config{})
	cert, err := svc.obtainWithRetry(restate.WithMockContext(ctx), db.CustomDomain{Domain: "one.example.com"})
	require.Empty(t, cert.CertificateID)
	require.True(t, restate.IsTerminalError(err))
}

func TestIssuanceServerFailureRemainsRetryable(t *testing.T) {
	client := acmeClientReturning(t, legoacme.ProblemDetails{
		HTTPStatus: http.StatusInternalServerError,
		Type:       "urn:ietf:params:acme:error:serverInternal",
		Detail:     "test CA unavailable",
	}, "")
	svc := New(Config{})
	result, err := svc.obtainCertificate(mocks.NewMockContext(t), client, db.CustomDomain{Domain: "one.example.com"})
	require.Error(t, err)
	require.False(t, restate.IsTerminalError(err))
	require.Zero(t, result.RetryDelay)
}

func TestCertificateAttemptReadsPreviouslyJournaledCertificate(t *testing.T) {
	var result certificateAttempt
	require.NoError(t, json.Unmarshal([]byte(`{"CertificateID":"existing","Certificate":"pem","EncryptedPrivateKey":"encrypted","ExpiresAt":1790742010123}`), &result))
	require.Equal(t, certificateAttempt{EncryptedCertificate: EncryptedCertificate{
		CertificateID: "existing", Certificate: "pem", EncryptedPrivateKey: "encrypted", ExpiresAt: 1790742010123,
	}}, result)
}

func counterValue(t *testing.T, counter prometheus.Counter) float64 {
	t.Helper()
	var metric prometheuspb.Metric
	require.NoError(t, counter.Write(&metric))
	return metric.GetCounter().GetValue()
}

func acmeClientReturning(t *testing.T, problem legoacme.ProblemDetails, retryAfter string) *lego.Client {
	t.Helper()
	mux := http.NewServeMux()
	server := httptest.NewTLSServer(mux)
	t.Cleanup(server.Close)
	mux.HandleFunc("GET /directory", func(w http.ResponseWriter, _ *http.Request) {
		if err := json.NewEncoder(w).Encode(map[string]string{
			"newNonce": server.URL + "/nonce", "newAccount": server.URL + "/account", "newOrder": server.URL + "/order",
		}); err != nil {
			t.Error(err)
		}
	})
	mux.HandleFunc("HEAD /nonce", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Replay-Nonce", "test-nonce")
	})
	mux.HandleFunc("POST /order", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/problem+json")
		w.Header().Set("Retry-After", retryAfter)
		w.WriteHeader(problem.HTTPStatus)
		if err := json.NewEncoder(w).Encode(problem); err != nil {
			t.Error(err)
		}
	})
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)
	config := lego.NewConfig(acmeTestUser{key: key})
	config.CADirURL = server.URL + "/directory"
	config.HTTPClient = server.Client()
	client, err := lego.NewClient(config)
	require.NoError(t, err)
	return client
}

type acmeTestUser struct {
	key crypto.PrivateKey
}

func (acmeTestUser) GetEmail() string { return "test@example.com" }

func (acmeTestUser) GetRegistration() *registration.Resource { return nil }

func (u acmeTestUser) GetPrivateKey() crypto.PrivateKey { return u.key }

type failedChallengeDB struct {
	db.Database
	updated db.UpdateAcmeChallengeStatusParams
}

func (d *failedChallengeDB) UpdateAcmeChallengeStatus(_ context.Context, params db.UpdateAcmeChallengeStatusParams) error {
	d.updated = params
	return nil
}
