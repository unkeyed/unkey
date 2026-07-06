// Package enduserbillingpush closes a billing period for end-user billing:
// it reads each Stripe-connected workspace's per-identity billable
// quantities, resolves and pins the rate card per identity (R18), prices the
// usage with the shared resolver (R19), and dispatches the priced records to
// the workspace's connected account (R12).
//
// It shares the monthly billing-period tick with the Deploy billing push —
// the proto toolchain required to mint a dedicated CronService RPC
// (protoc-gen-go-restate) is not available in this change, so the handler
// invokes this component as an optional, independently-configured second
// phase of the same period key.
package enduserbillingpush

import (
	"context"
	"errors"
	"fmt"

	"github.com/unkeyed/unkey/internal/services/billing"
	"github.com/unkeyed/unkey/pkg/assert"
	"github.com/unkeyed/unkey/pkg/clickhouse"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/fault"
	"github.com/unkeyed/unkey/pkg/logger"
	"github.com/unkeyed/unkey/pkg/ratecard"
	"github.com/unkeyed/unkey/svc/ctrl/internal/enduserbilling"
)

// pageSize bounds each MySQL page of workspaces and identities.
const pageSize = 100

// UsageReader reads per-identity billable quantities from ClickHouse, scoped to
// the keyspaces and ratelimit namespaces the workspace has enabled for billing.
type UsageReader interface {
	GetBillableUsagePerIdentity(ctx context.Context, workspaceID string, year, month int, enabledKeyspaces, enabledNamespaces []string) ([]clickhouse.IdentityBillableUsage, error)
}

// Decrypter decrypts the workspace's Vault-encrypted connected account
// reference.
type Decrypter interface {
	Decrypt(ctx context.Context, keyring, encrypted string) (string, error)
}

// Config holds the component's dependencies.
type Config struct {
	// DB is the application database (pkg/db). Must not be nil.
	DB db.Database
	// Usage reads the per-identity rollup. Must not be nil.
	Usage UsageReader
	// Vault decrypts connected-account references. Must not be nil.
	Vault Decrypter
	// Pusher dispatches priced records to the billing provider. Must not be
	// nil; use enduserbilling.NewNoop() to disable pushing.
	Pusher enduserbilling.MeterPusher
}

// PeriodClose orchestrates one end-user billing period close.
type PeriodClose struct {
	database db.Database
	usage    UsageReader
	vault    Decrypter
	pusher   enduserbilling.MeterPusher
	resolver *billing.Resolver
}

// New constructs a PeriodClose.
func New(cfg Config) (*PeriodClose, error) {
	if err := assert.All(
		assert.NotNil(cfg.DB, "DB must not be nil"),
		assert.NotNil(cfg.Usage, "Usage must not be nil"),
		assert.NotNil(cfg.Vault, "Vault must not be nil"),
		assert.NotNil(cfg.Pusher, "Pusher must not be nil; use enduserbilling.NewNoop()"),
	); err != nil {
		return nil, err
	}
	return &PeriodClose{
		database: cfg.DB,
		usage:    cfg.Usage,
		vault:    cfg.Vault,
		pusher:   cfg.Pusher,
		resolver: billing.NewResolver(cfg.DB),
	}, nil
}

// Summary reports what one run did. Errors are collected per workspace and
// identity rather than aborting the whole run: one customer's
// misconfiguration must not block another customer's billing.
type Summary struct {
	Workspaces    int
	RecordsPushed int
	Errors        []string
}

// Run closes the given period for every Stripe-connected workspace. The
// period MUST be already closed (a past month): the caller passes the most
// recently closed month, never the open one, because the additive Stripe
// invoice items are only correct once a period's usage is final. Re-running a
// closed period is safe and cheap: each identity is stamped pushed on success
// (billing_period_rate_cards.pushed_at), so subsequent hourly ticks skip the
// already-billed identities and only retry the ones that failed.
func (p *PeriodClose) Run(ctx context.Context, year, month int) (Summary, error) {
	summary := Summary{Workspaces: 0, RecordsPushed: 0, Errors: nil}

	for offset := 0; ; offset += pageSize {
		settings, err := db.Query.ListStripeConnectedWorkspaces(ctx, p.database.RO(), db.ListStripeConnectedWorkspacesParams{
			Limit:  pageSize,
			Offset: int32(offset),
		})
		if err != nil {
			return summary, fault.Wrap(err, fault.Internal("failed to list stripe-connected workspaces"))
		}
		if len(settings) == 0 {
			break
		}

		for _, ws := range settings {
			summary.Workspaces++
			pushed, wsErr := p.runWorkspace(ctx, ws, year, month)
			summary.RecordsPushed += pushed
			if wsErr != nil {
				summary.Errors = append(summary.Errors, fmt.Sprintf("workspace %s: %s", ws.WorkspaceID, wsErr.Error()))
			}
		}

		if len(settings) < pageSize {
			break
		}
	}

	logger.Info("end-user billing period close finished",
		"year", year, "month", month,
		"workspaces", summary.Workspaces,
		"records_pushed", summary.RecordsPushed,
		"errors", len(summary.Errors),
	)
	// Money-movement failures must not be reduced to an Info-level count.
	// Surface each collected error at Error level with its workspace/identity
	// context so monitoring and on-call see per-customer billing failures.
	// Run still returns nil so one customer's failure does not abort billing
	// for the rest; the caller decides how to act on a non-empty summary.
	for _, e := range summary.Errors {
		logger.Error("end-user billing failure during period close",
			"year", year, "month", month, "detail", e,
		)
	}
	return summary, nil
}

func (p *PeriodClose) runWorkspace(ctx context.Context, ws db.WorkspaceBillingSetting, year, month int) (int, error) {
	if !ws.StripeConnectEncrypted.Valid {
		return 0, nil
	}
	connectedAccountID, err := p.vault.Decrypt(ctx, ws.WorkspaceID, ws.StripeConnectEncrypted.String)
	if err != nil {
		return 0, fault.Wrap(err, fault.Internal("failed to decrypt connected account reference"))
	}

	// Only keyspaces/namespaces the workspace enabled for billing contribute
	// usage. With nothing enabled there is nothing to bill — skip before the
	// ClickHouse read.
	enabledKeyspaces, enabledNamespaces, err := billing.LoadEnabledResources(ctx, p.database, ws.WorkspaceID)
	if err != nil {
		return 0, err
	}
	if len(enabledKeyspaces) == 0 && len(enabledNamespaces) == 0 {
		// A Stripe-connected workspace that has enabled nothing bills nothing.
		// Log it: connecting Stripe but enabling no resources is a plausible
		// misconfiguration (the customer expects usage to bill), and a silent
		// skip would leave that invisible.
		logger.Info("end-user billing skipped: workspace has no billable resources enabled",
			"workspace_id", ws.WorkspaceID, "year", year, "month", month,
		)
		return 0, nil
	}

	usage, err := p.usage.GetBillableUsagePerIdentity(ctx, ws.WorkspaceID, year, month, enabledKeyspaces, enabledNamespaces)
	if err != nil {
		return 0, fault.Wrap(err, fault.Internal("failed to read per-identity usage"))
	}
	usageByIdentity := make(map[string]clickhouse.IdentityBillableUsage, len(usage))
	for _, u := range usage {
		usageByIdentity[u.IdentityID] = u
	}

	pushedTotal := 0
	var errs []error

	// One batch resolver per workspace: it reads the workspace billing
	// settings once and caches each rate card, so resolving N identities costs
	// O(1) settings + O(distinct cards) reads instead of O(N) of each.
	resolver := p.resolver.NewBatch(ws.WorkspaceID)

	for offset := 0; ; offset += pageSize {
		identities, listErr := db.Query.ListBillingBoundIdentities(ctx, p.database.RO(), db.ListBillingBoundIdentitiesParams{
			WorkspaceID:     ws.WorkspaceID,
			BillingProvider: db.IdentitiesBillingProviderStripeConnect,
			Limit:           pageSize,
			Offset:          int32(offset),
		})
		if listErr != nil {
			errs = append(errs, fault.Wrap(listErr, fault.Internal("failed to list billing-bound identities")))
			break
		}
		if len(identities) == 0 {
			break
		}

		for _, identity := range identities {
			u, hasUsage := usageByIdentity[identity.ID]
			if !hasUsage {
				continue
			}

			resolved, resolveErr := resolver.ResolveAndRecord(ctx, identity, year, month)
			if resolveErr != nil {
				if errors.Is(resolveErr, billing.ErrNoRateCard) {
					errs = append(errs, fault.New("identity "+identity.ID+" has usage but no rate card resolves"))
					continue
				}
				errs = append(errs, resolveErr)
				continue
			}

			// Run-once: a period already pushed for this identity is skipped.
			// The push targets a closed period, so its usage is final; billing
			// it once (rather than on every hourly re-tick) is what stops the
			// additive invoice item from double-billing after Stripe's 24h
			// idempotency window lapses.
			if resolved.AlreadyPushed {
				continue
			}

			amounts, priceErr := resolved.Price(u.Verifications, u.SpentCredits, u.RatelimitsPassed)
			if priceErr != nil {
				errs = append(errs, priceErr)
				continue
			}

			providerCustomerID := ""
			if identity.BillingExternalCustomerID.Valid {
				providerCustomerID = identity.BillingExternalCustomerID.String
			}

			// Push one identity at a time so the pushed marker is exact: an
			// identity is stamped only after its own push succeeds, so a
			// partial failure retries just the failed identities next tick
			// (a batch marker would either skip retries or re-push peers).
			pushed, pushErr := p.pusher.Push(ctx, enduserbilling.PushRequest{
				WorkspaceID:        ws.WorkspaceID,
				ConnectedAccountID: connectedAccountID,
				Year:               year,
				Month:              month,
				Records: []enduserbilling.UsageRecord{{
					IdentityID:         identity.ID,
					ExternalID:         identity.ExternalID,
					ProviderCustomerID: providerCustomerID,
					RateCardID:         resolved.Card.ID,
					Verifications:      u.Verifications,
					SpentCredits:       u.SpentCredits,
					RatelimitsPassed:   u.RatelimitsPassed,
					VerificationsCents: ratecard.RoundedCents(amounts.VerificationsCents),
					CreditsCents:       ratecard.RoundedCents(amounts.CreditsCents),
					RatelimitsCents:    ratecard.RoundedCents(amounts.RatelimitsCents),
					Currency:           resolved.Card.Currency,
				}},
			})
			pushedTotal += pushed
			if pushErr != nil {
				errs = append(errs, pushErr)
				continue
			}
			if markErr := resolver.MarkPushed(ctx, identity.ID, year, month); markErr != nil {
				// Money-critical: the usage was already pushed to the provider,
				// but recording the run-once marker failed. The next tick sees
				// AlreadyPushed=false and re-pushes; the provider's idempotency
				// key dedups within its window, but a marker write that stays
				// failed past that window risks a double charge. Surface it
				// loudly at the site (not just in the aggregated tail) so on-call
				// sees the exact identity at risk.
				logger.Error("pushed to provider but failed to record pushed marker; identity will be re-pushed next tick",
					"workspace_id", ws.WorkspaceID,
					"identity_id", identity.ID,
					"year", year,
					"month", month,
					"error", markErr.Error(),
				)
				errs = append(errs, markErr)
			}
		}

		if len(identities) < pageSize {
			break
		}
	}

	return pushedTotal, errors.Join(errs...)
}
