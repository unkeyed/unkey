package certificate

import (
	"fmt"
	"time"

	restate "github.com/restatedev/sdk-go"
	restateIngress "github.com/restatedev/sdk-go/ingress"
	ctrlrestate "github.com/unkeyed/unkey/go/apps/ctrl/restate"
	certificateworkflow "github.com/unkeyed/unkey/go/apps/ctrl/workflows/certificate"
	"github.com/unkeyed/unkey/go/pkg/db"
	"github.com/unkeyed/unkey/go/pkg/otel/logging"
)

// CertificateCron is a service that periodically triggers certificate challenges
type CertificateCron struct {
	db                  db.Database
	logger              logging.Logger
	supportedTypes      []db.AcmeChallengesType
	checkIntervalMins   int
	restateClient       *ctrlrestate.Client
	certificateWorkflow *certificateworkflow.CertificateChallenge
}

type CertificateCronConfig struct {
	DB                  db.Database
	Logger              logging.Logger
	SupportedTypes      []db.AcmeChallengesType
	CheckIntervalMins   int
	RestateClient       *ctrlrestate.Client
	CertificateWorkflow *certificateworkflow.CertificateChallenge
}

func NewCertificateCron(cfg CertificateCronConfig) *CertificateCron {
	return &CertificateCron{
		db:                  cfg.DB,
		logger:              cfg.Logger,
		supportedTypes:      cfg.SupportedTypes,
		checkIntervalMins:   cfg.CheckIntervalMins,
		restateClient:       cfg.RestateClient,
		certificateWorkflow: cfg.CertificateWorkflow,
	}
}

func (c *CertificateCron) Name() string {
	return "certificate_cron"
}

type CronTriggerRequest struct {
	Timestamp int64 `json:"timestamp"`
}

// Run checks for executable challenges and schedules workflows for them
func (c *CertificateCron) Run(ctx restate.Context, req CronTriggerRequest) error {
	c.logger.Info("certificate cron triggered", "timestamp", req.Timestamp)

	// Query for executable challenges
	executableChallenges, err := restate.Run(ctx, func(stepCtx restate.RunContext) ([]db.ListExecutableChallengesRow, error) {
		return db.Query.ListExecutableChallenges(stepCtx, c.db.RO(), c.supportedTypes)
	}, restate.WithName("listing executable challenges"))
	if err != nil {
		c.logger.Error("failed to list executable challenges", "error", err)
		return err
	}

	c.logger.Info("found executable challenges",
		"count", len(executableChallenges),
		"supported_types", c.supportedTypes)

	// Start a workflow for each challenge
	for _, challenge := range executableChallenges {
		_, err := restate.Run(ctx, func(stepCtx restate.RunContext) (restate.Void, error) {
			// Send workflow request to certificate_challenge workflow
			workflowReq := certificateworkflow.CertificateChallengeRequest{
				WorkspaceID: challenge.WorkspaceID,
				Domain:      challenge.Domain,
			}

			// Use workflow's Start method
			err := c.certificateWorkflow.Start(stepCtx, c.restateClient.Raw(), workflowReq)
			if err != nil {
				return restate.Void{}, fmt.Errorf("failed to start certificate workflow: %w", err)
			}

			c.logger.Info("started certificate challenge workflow",
				"domain", challenge.Domain,
				"workspace_id", challenge.WorkspaceID)

			return restate.Void{}, nil
		}, restate.WithName(fmt.Sprintf("starting workflow for %s", challenge.Domain)))

		if err != nil {
			c.logger.Error("failed to start workflow",
				"domain", challenge.Domain,
				"error", err)
			// Continue with other challenges even if one fails
		}
	}

	// Schedule next execution
	nextRun := time.Now().Add(time.Duration(c.checkIntervalMins) * time.Minute)
	_, err = restate.Run(ctx, func(stepCtx restate.RunContext) (restate.Void, error) {
		// Sleep until next execution time
		restate.Sleep(ctx, time.Duration(c.checkIntervalMins)*time.Minute)

		// Trigger the next run
		invocation := restateIngress.ServiceSend[CronTriggerRequest](c.restateClient.Raw(), "certificate_cron", "Run").Send(stepCtx, CronTriggerRequest{
			Timestamp: nextRun.UnixMilli(),
		})

		if invocation.Error != nil {
			return restate.Void{}, fmt.Errorf("failed to schedule next cron run: %w", invocation.Error)
		}

		c.logger.Info("scheduled next cron run")

		return restate.Void{}, nil
	}, restate.WithName("scheduling next run"))

	if err != nil {
		c.logger.Error("failed to schedule next run", "error", err)
		return err
	}

	c.logger.Info("certificate cron completed", "next_run", nextRun)
	return nil
}
