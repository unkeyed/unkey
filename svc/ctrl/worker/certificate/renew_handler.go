package certificate

import (
	"fmt"
	"time"

	restate "github.com/restatedev/sdk-go"
	hydrav1 "github.com/unkeyed/unkey/gen/proto/hydra/v1"
	"github.com/unkeyed/unkey/pkg/logger"
	"github.com/unkeyed/unkey/svc/ctrl/internal/db"
)

// RenewExpiringCertificates submits issuance and renewal work for eligible domains.
func (s *Service) RenewExpiringCertificates(
	ctx restate.ObjectContext,
	req *hydrav1.RenewExpiringCertificatesRequest,
) (*hydrav1.RenewExpiringCertificatesResponse, error) {
	logger.Info("starting certificate renewal check")

	challengeTypes := []db.AcmeChallengesChallengeType{
		db.AcmeChallengesChallengeTypeDNS01,
		db.AcmeChallengesChallengeTypeHTTP01,
	}

	challenges, err := restate.Run(ctx, func(stepCtx restate.RunContext) ([]db.ListExecutableChallengesRow, error) {
		return s.db.ListExecutableChallenges(stepCtx, challengeTypes)
	}, restate.WithName("list expiring certificates"))
	if err != nil {
		return nil, err
	}

	logger.Info("found certificates to process", "count", len(challenges))

	renewalsTriggered := int32(0)

	for _, challenge := range challenges {
		logger.Info("triggering certificate renewal",
			"domain", challenge.Domain,
			"workspace_id", challenge.WorkspaceID,
		)

		client := hydrav1.NewCertificateServiceClient(ctx, challenge.Domain)
		invocation := client.ProcessChallenge().Send(&hydrav1.ProcessChallengeRequest{
			WorkspaceId: challenge.WorkspaceID,
			Domain:      challenge.Domain,
		})

		logger.Info("certificate renewal submitted",
			"domain", challenge.Domain,
			"invocation_id", invocation.GetInvocationId(),
		)
		renewalsTriggered++

		if err := restate.Sleep(ctx, 100*time.Millisecond); err != nil {
			return nil, err
		}
	}

	logger.Info("certificate renewal check completed",
		"checked", len(challenges),
		"triggered", renewalsTriggered,
	)

	_, err = restate.Run(ctx, func(rc restate.RunContext) (restate.Void, error) {
		return restate.Void{}, s.heartbeat.Ping(rc)
	}, restate.WithName("send heartbeat"))
	if err != nil {
		return nil, fmt.Errorf("send heartbeat: %w", err)
	}

	return &hydrav1.RenewExpiringCertificatesResponse{
		CertificatesChecked: int32(len(challenges)),
		RenewalsTriggered:   renewalsTriggered,
	}, nil
}
