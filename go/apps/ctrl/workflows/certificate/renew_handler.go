package certificate

import (
	"time"

	restate "github.com/restatedev/sdk-go"
	hydrav1 "github.com/unkeyed/unkey/go/gen/proto/hydra/v1"
	"github.com/unkeyed/unkey/go/pkg/db"
)

const (
	// renewalInterval is how often the certificate renewal check runs
	renewalInterval = 24 * time.Hour

	// renewalKey is the virtual object key for the singleton renewal job
	renewalKey = "global"
)

// RenewExpiringCertificates scans for certificates expiring soon and triggers renewal.
// This is a self-scheduling Restate cron job - after completing, it schedules itself
// to run again after renewalInterval (24 hours).
//
// To start the cron job, call this handler once with key "global". It will then
// automatically reschedule itself forever.
func (s *Service) RenewExpiringCertificates(
	ctx restate.ObjectContext,
	req *hydrav1.RenewExpiringCertificatesRequest,
) (*hydrav1.RenewExpiringCertificatesResponse, error) {
	s.logger.Info("starting certificate renewal check")

	challengeTypes := []db.AcmeChallengesChallengeType{
		db.AcmeChallengesChallengeTypeDNS01,
		db.AcmeChallengesChallengeTypeHTTP01,
	}

	// Find all challenges that need processing (waiting or expiring soon)
	challenges, err := restate.Run(ctx, func(stepCtx restate.RunContext) ([]db.ListExecutableChallengesRow, error) {
		return db.Query.ListExecutableChallenges(stepCtx, s.db.RO(), challengeTypes)
	}, restate.WithName("list expiring certificates"))
	if err != nil {
		return nil, err
	}

	s.logger.Info("found certificates to process", "count", len(challenges))

	var failedDomains []string
	renewalsTriggered := int32(0)

	for _, challenge := range challenges {
		s.logger.Info("triggering certificate renewal",
			"domain", challenge.Domain,
			"workspace_id", challenge.WorkspaceID,
		)

		// Trigger the ProcessChallenge workflow for this domain (fire-and-forget)
		client := hydrav1.NewCertificateServiceClient(ctx, challenge.Domain)
		sendErr := client.ProcessChallenge().Send(&hydrav1.ProcessChallengeRequest{
			WorkspaceId: challenge.WorkspaceID,
			Domain:      challenge.Domain,
		})

		if sendErr != nil {
			s.logger.Warn("failed to trigger renewal",
				"domain", challenge.Domain,
				"error", sendErr,
			)
			failedDomains = append(failedDomains, challenge.Domain)
		} else {
			renewalsTriggered++
		}

		// Small delay between requests to avoid overwhelming the system
		if err := restate.Sleep(ctx, 100*time.Millisecond); err != nil {
			return nil, err
		}
	}

	s.logger.Info("certificate renewal check completed",
		"checked", len(challenges),
		"triggered", renewalsTriggered,
		"failed", len(failedDomains),
	)

	// Schedule next run - this creates the Restate cron pattern
	// The job will run again after renewalInterval
	// Use idempotency key based on the next run date to prevent duplicate schedules
	nextRunDate := time.Now().Add(renewalInterval).Format("2006-01-02")
	selfClient := hydrav1.NewCertificateServiceClient(ctx, renewalKey)
	selfClient.RenewExpiringCertificates().Send(
		&hydrav1.RenewExpiringCertificatesRequest{},
		restate.WithDelay(renewalInterval),
		restate.WithIdempotencyKey("cert-renewal-"+nextRunDate),
	)

	s.logger.Info("scheduled next certificate renewal check", "delay", renewalInterval)

	return &hydrav1.RenewExpiringCertificatesResponse{
		CertificatesChecked: int32(len(challenges)),
		RenewalsTriggered:   renewalsTriggered,
		FailedDomains:       failedDomains,
	}, nil
}
