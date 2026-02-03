package certificate

import (
	"fmt"
	"time"

	restate "github.com/restatedev/sdk-go"
	hydrav1 "github.com/unkeyed/unkey/gen/proto/hydra/v1"
	"github.com/unkeyed/unkey/pkg/db"
)

// RenewExpiringCertificates renews certificates before they expire.
//
// This handler queries for certificates expiring within 30 days (based on the
// acme_challenges table) and triggers [Service.ProcessChallenge] for each one via
// fire-and-forget Restate calls. The ProcessChallenge handler handles actual renewal.
//
// This handler is intended to be called on a schedule via GitHub Actions.
// A 100ms delay is inserted between renewal triggers to avoid overwhelming the system
// when many certificates need renewal simultaneously.
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

	// Send heartbeat to indicate successful completion
	_, err = restate.Run(ctx, func(rc restate.RunContext) (restate.Void, error) {
		return restate.Void{}, s.heartbeat.Ping(rc)
	}, restate.WithName("send heartbeat"))
	if err != nil {
		return nil, fmt.Errorf("send heartbeat: %w", err)
	}

	return &hydrav1.RenewExpiringCertificatesResponse{
		CertificatesChecked: int32(len(challenges)),
		RenewalsTriggered:   renewalsTriggered,
		FailedDomains:       failedDomains,
	}, nil
}
