package cron_test

import (
	"context"
	"errors"
	"slices"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/email"
	"github.com/unkeyed/unkey/svc/ctrl/integration/harness"
	"github.com/unkeyed/unkey/svc/ctrl/integration/seed"
)

type quotaAlertAdmins struct {
	mu     sync.Mutex
	orgID  string
	emails []string
}

func (a *quotaAlertAdmins) AdminEmails(_ context.Context, orgID string) ([]string, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if orgID != a.orgID {
		return nil, nil
	}
	return slices.Clone(a.emails), nil
}

func (a *quotaAlertAdmins) SetOrgID(orgID string) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.orgID = orgID
}

type quotaAlertEmails struct {
	mu       sync.Mutex
	messages []email.Email
}

type failingQuotaAlertAdmins struct {
	mu                   sync.Mutex
	resolverFailureOrgID string
	senderFailureOrgID   string
	resolverFailureCalls int
}

func (a *failingQuotaAlertAdmins) AdminEmails(_ context.Context, orgID string) ([]string, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	switch orgID {
	case a.resolverFailureOrgID:
		a.resolverFailureCalls++
		return nil, errors.New("resolve recipients")
	case a.senderFailureOrgID:
		return []string{"send-failure@example.com"}, nil
	default:
		return nil, nil
	}
}

func (a *failingQuotaAlertAdmins) ResolverFailureCalls() int {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.resolverFailureCalls
}

type failingQuotaAlertEmails struct {
	mu        sync.Mutex
	templates []string
}

func (s *failingQuotaAlertEmails) Send(_ context.Context, message email.Email) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if slices.Contains(message.To, "send-failure@example.com") {
		s.templates = append(s.templates, message.TemplateID)
		return errors.New("send email")
	}
	return nil
}

func (s *failingQuotaAlertEmails) Templates() []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return slices.Clone(s.templates)
}

func (s *quotaAlertEmails) Send(_ context.Context, message email.Email) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.messages = append(s.messages, message)
	return nil
}

func (s *quotaAlertEmails) Messages() []email.Email {
	s.mu.Lock()
	defer s.mu.Unlock()
	return slices.Clone(s.messages)
}

func TestRunQuotaCheck_EmailsFreeWorkspaceTwice(t *testing.T) {
	sender := &quotaAlertEmails{}
	admins := &quotaAlertAdmins{emails: []string{"owner@example.com"}}
	h := harness.New(t, harness.WithQuotaAlerts(
		admins,
		sender,
		0,
	))

	now := time.Now()
	billingPeriod := now.Format("2006-01")
	ws := h.Seed.CreateWorkspaceWithQuota(h.Ctx, seed.CreateWorkspaceWithQuotaRequest{
		RequestsPerMonth: 150_000,
	})
	admins.SetOrgID(ws.OrgID)

	h.ClickHouseSeed.InsertVerifications(h.Ctx, ws.ID, 200_000, now, "VALID")
	waitForVerificationCount(t, h.Ctx, h.ClickHouseConn, ws.ID, 200_000, now.Year(), int(now.Month()))

	resp, err := callRunQuotaCheck(h, billingPeriod)
	require.NoError(t, err)
	require.GreaterOrEqual(t, resp.GetNotificationsSent(), int32(1))
	messages := sender.Messages()
	require.Len(t, messages, 1)
	require.Equal(t, "api-usage-exceeded", messages[0].TemplateID)

	resp, err = callRunQuotaCheck(h, billingPeriod)
	require.NoError(t, err)
	require.GreaterOrEqual(t, resp.GetNotificationsSent(), int32(1))
	messages = sender.Messages()
	require.Len(t, messages, 2)
	require.Equal(t, "api-usage-ratelimit-follow-up", messages[1].TemplateID)

	resp, err = callRunQuotaCheck(h, billingPeriod)
	require.NoError(t, err)
	require.GreaterOrEqual(t, resp.GetNotificationsSent(), int32(1))
	messages = sender.Messages()
	require.Len(t, messages, 2)

	require.Equal(t, []string{"owner@example.com"}, messages[0].To)
	require.Equal(t, ws.Name, messages[0].Variables["WORKSPACE_NAME"])
	require.Equal(t, "150,000", messages[0].Variables["LIMIT"])
	require.Equal(t, "200,000", messages[0].Variables["USED"])
	require.Equal(t, "https://app.unkey.com/"+ws.Slug+"/settings/billing", messages[0].Variables["BILLING_URL"])
	require.Equal(t, "quota-alert/"+ws.ID+"/"+billingPeriod+"/1", messages[0].IdempotencyKey)

	require.Equal(t, "https://app.unkey.com/"+ws.Slug+"/settings/billing", messages[1].Variables["BILLING_URL"])
	require.NotContains(t, messages[1].Variables, "RATELIMIT_URL")
	require.NotContains(t, messages[1].Variables, "RATELIMIT_DOCS_URL")
	require.Equal(t, "quota-alert/"+ws.ID+"/"+billingPeriod+"/2", messages[1].IdempotencyKey)
}

func TestRunQuotaCheck_EmailFailuresDoNotBlockOrAdvanceState(t *testing.T) {
	admins := &failingQuotaAlertAdmins{}
	sender := &failingQuotaAlertEmails{}
	h := harness.New(t, harness.WithQuotaAlerts(admins, sender, 0))

	now := time.Now()
	billingPeriod := now.Format("2006-01")
	resolverFailureWorkspace := h.Seed.CreateWorkspaceWithQuota(h.Ctx, seed.CreateWorkspaceWithQuotaRequest{
		RequestsPerMonth: 150_000,
	})
	senderFailureWorkspace := h.Seed.CreateWorkspaceWithQuota(h.Ctx, seed.CreateWorkspaceWithQuotaRequest{
		RequestsPerMonth: 150_000,
	})
	admins.resolverFailureOrgID = resolverFailureWorkspace.OrgID
	admins.senderFailureOrgID = senderFailureWorkspace.OrgID

	h.ClickHouseSeed.InsertVerifications(h.Ctx, resolverFailureWorkspace.ID, 200_000, now, "VALID")
	h.ClickHouseSeed.InsertVerifications(h.Ctx, senderFailureWorkspace.ID, 200_000, now, "VALID")
	waitForVerificationCount(t, h.Ctx, h.ClickHouseConn, resolverFailureWorkspace.ID, 200_000, now.Year(), int(now.Month()))
	waitForVerificationCount(t, h.Ctx, h.ClickHouseConn, senderFailureWorkspace.ID, 200_000, now.Year(), int(now.Month()))

	for range 2 {
		resp, err := callRunQuotaCheck(h, billingPeriod)
		require.NoError(t, err)
		require.GreaterOrEqual(t, resp.GetNotificationsSent(), int32(2))
	}

	// Each invocation retries each failed side effect five times. Repeating the
	// first template proves failed sends did not advance customer email state.
	require.Equal(t, 10, admins.ResolverFailureCalls())
	templates := sender.Templates()
	require.Len(t, templates, 10)
	for _, template := range templates {
		require.Equal(t, "api-usage-exceeded", template)
	}
}
