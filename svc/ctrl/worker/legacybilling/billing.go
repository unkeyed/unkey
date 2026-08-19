package legacybilling

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/big"
	"regexp"
	"strings"
	"time"

	restate "github.com/restatedev/sdk-go"
	stripe "github.com/stripe/stripe-go/v86"
	hydrav1 "github.com/unkeyed/unkey/gen/proto/hydra/v1"
	"github.com/unkeyed/unkey/pkg/clickhouse"
	"github.com/unkeyed/unkey/pkg/logger"
	"github.com/unkeyed/unkey/pkg/restate/restateutil"
	"github.com/unkeyed/unkey/svc/ctrl/internal/db"
)

const (
	// invoiceSource identifies invoices owned by this workflow during reconciliation.
	invoiceSource         = "unkey-billing-tool"
	providerRetryDuration = 15 * time.Minute
)

// decimalPattern matches Stripe decimal prices without converting money through
// binary floating point.
var decimalPattern = regexp.MustCompile(`^\d{1,15}(\.\d{1,12})?$`)

// Handler prepares legacy Stripe draft invoices as durable Restate workflows.
type Handler struct {
	hydrav1.UnimplementedLegacyBillingWorkflowServer
	db     db.Database
	usage  clickhouse.ClickHouse
	stripe *stripe.Client
}

// New creates a legacy billing workflow handler using worker-owned clients.
func New(database db.Database, usage clickhouse.ClickHouse, stripeSecretKey string) *Handler {
	return &Handler{
		UnimplementedLegacyBillingWorkflowServer: hydrav1.UnimplementedLegacyBillingWorkflowServer{},
		db:                                       database,
		usage:                                    usage,
		stripe:                                   stripe.NewClient(stripeSecretKey),
	}
}

// fixedSubscription is the legacy JSON representation of a fixed monthly fee.
type fixedSubscription struct {
	ProductID string `json:"productId"`
	Cents     string `json:"cents"`
}

// billingTier is an inclusive legacy usage range. Nil means an unbounded last
// unit or a free tier.
type billingTier struct {
	FirstUnit    int64   `json:"firstUnit"`
	LastUnit     *int64  `json:"lastUnit,omitempty"`
	CentsPerUnit *string `json:"centsPerUnit,omitempty"`
}

// tieredSubscription is the legacy JSON representation of usage pricing.
type tieredSubscription struct {
	ProductID string        `json:"productId"`
	Tiers     []billingTier `json:"tiers"`
}

// subscriptions contains every legacy product that can contribute invoice lines.
type subscriptions struct {
	Verifications *tieredSubscription `json:"verifications"`
	Ratelimits    *tieredSubscription `json:"ratelimits"`
	Plan          *fixedSubscription  `json:"plan"`
	Support       *fixedSubscription  `json:"support"`
}

// invoiceItem is the exact line specification expected in Stripe.
type invoiceItem struct {
	Description       string
	ProductID         string
	Quantity          int64
	UnitAmountDecimal string
	Key               string
}

// invoiceInput is the validated, dependency-independent specification passed to
// Stripe. Building it completely before mutation keeps invalid runs side-effect free.
type invoiceInput struct {
	workspaceName string
	customerID    string
	period        string
	periodStart   time.Time
	periodEnd     time.Time
	verifications int64
	ratelimits    int64
	items         []invoiceItem
}

// workspaceInput contains the MySQL state required to calculate a legacy bill.
// Its fields are exported because Restate journals values returned from Run.
type workspaceInput struct {
	Name          string `json:"name"`
	CustomerID    string `json:"customer_id"`
	Subscriptions []byte `json:"subscriptions"`
}

// Run validates source data and prepares or resumes one standalone Stripe draft.
func (h *Handler) Run(ctx restate.Context, req *hydrav1.LegacyBillingRunRequest) (*hydrav1.LegacyBillingRunResponse, error) {
	workspaceID, year, month := strings.TrimSpace(req.GetWorkspaceId()), int(req.GetYear()), int(req.GetMonth())
	now, err := restateutil.Now(ctx)
	if err != nil {
		return nil, fmt.Errorf("get current time: %w", err)
	}
	if err = validateRequestAt(workspaceID, year, month, now); err != nil {
		return nil, restate.TerminalError(err)
	}
	logger.Info("legacy billing workflow started", "workspace_id", workspaceID, "year", year, "month", month)

	workspace, err := restate.Run(ctx, func(rc restate.RunContext) (workspaceInput, error) {
		return loadWorkspaceInput(rc, h.db, workspaceID)
	}, restate.WithName("load billing state"), restate.WithMaxRetryDuration(providerRetryDuration))
	if err != nil {
		return nil, restate.TerminalError(err)
	}
	subs, err := parseSubscriptions(workspace.Subscriptions)
	if err != nil {
		return nil, restate.TerminalError(fmt.Errorf("parse subscriptions for workspace %q: %w", workspaceID, err))
	}
	var input invoiceInput
	input.workspaceName = workspace.Name
	input.customerID = workspace.CustomerID
	if subs.Verifications != nil {
		logger.Info("querying billable verifications", "workspace_id", workspaceID, "year", year, "month", month)
		input.verifications, err = restate.Run(ctx, func(rc restate.RunContext) (int64, error) {
			return h.usage.GetBillableVerifications(rc, workspaceID, year, month)
		}, restate.WithName("get billable verifications"), restate.WithMaxRetryDuration(providerRetryDuration))
		if err != nil {
			return nil, fmt.Errorf("query billable verifications: %w", err)
		}
		logger.Info("loaded billable verifications", "workspace_id", workspaceID, "count", input.verifications)
	}
	if subs.Ratelimits != nil {
		logger.Info("querying billable ratelimits", "workspace_id", workspaceID, "year", year, "month", month)
		input.ratelimits, err = restate.Run(ctx, func(rc restate.RunContext) (int64, error) {
			return h.usage.GetBillableRatelimits(rc, workspaceID, year, month)
		}, restate.WithName("get billable ratelimits"), restate.WithMaxRetryDuration(providerRetryDuration))
		if err != nil {
			return nil, fmt.Errorf("query billable ratelimits: %w", err)
		}
		logger.Info("loaded billable ratelimits", "workspace_id", workspaceID, "count", input.ratelimits)
	}
	input.items, err = buildInvoiceItems(subs, input.verifications, input.ratelimits)
	if err != nil {
		return nil, restate.TerminalError(err)
	}
	if err = validateInvoiceItemCount(input.items); err != nil {
		return nil, restate.TerminalError(err)
	}
	input.period = fmt.Sprintf("%04d-%02d", year, month)
	input.periodStart = time.Date(year, time.Month(month), 1, 0, 0, 0, 0, time.UTC)
	input.periodEnd = input.periodStart.AddDate(0, 1, 0).Add(-time.Second)
	invoiceID, err := restate.Run(ctx, func(rc restate.RunContext) (string, error) {
		invoice, runErr := prepareDraftInvoice(rc, h.stripe, workspaceID, input)
		if runErr != nil {
			return "", runErr
		}
		return invoice.ID, nil
	}, restate.WithName("reconcile Stripe draft"), restate.WithMaxRetryDuration(providerRetryDuration))
	if err != nil {
		return nil, fmt.Errorf("reconcile Stripe draft: %w", err)
	}
	logger.Info("legacy billing workflow completed", "workspace_id", workspaceID, "invoice_id", invoiceID, "item_count", len(input.items))
	return &hydrav1.LegacyBillingRunResponse{InvoiceId: invoiceID, ItemCount: int32(len(input.items)), Verifications: input.verifications, Ratelimits: input.ratelimits}, nil
}

// loadWorkspaceInput rejects deleted, unbillable, or subscription-billed
// workspaces before their legacy pricing can reach Stripe.
func loadWorkspaceInput(ctx context.Context, database db.Database, workspaceID string) (workspaceInput, error) {
	workspace, err := database.FindWorkspaceByID(ctx, workspaceID)
	if err != nil {
		if db.IsNotFound(err) {
			return workspaceInput{}, restate.TerminalError(fmt.Errorf("workspace %q does not exist", workspaceID))
		}
		return workspaceInput{}, fmt.Errorf("load workspace %q: %w", workspaceID, err)
	}
	if workspace.DeletedAtM.Valid {
		return workspaceInput{}, restate.TerminalError(fmt.Errorf("workspace %q is deleted", workspaceID))
	}

	billing, err := database.FindWorkspaceBillingByWorkspaceID(ctx, workspaceID)
	if err != nil {
		if db.IsNotFound(err) {
			return workspaceInput{}, restate.TerminalError(fmt.Errorf("workspace %q has no billing configuration", workspaceID))
		}
		return workspaceInput{}, fmt.Errorf("load billing configuration for workspace %q: %w", workspaceID, err)
	}
	if billing.DeletedAtM.Valid {
		return workspaceInput{}, restate.TerminalError(fmt.Errorf("billing configuration for workspace %q is deleted", workspaceID))
	}
	if !billing.StripeCustomerID.Valid || strings.TrimSpace(billing.StripeCustomerID.String) == "" {
		return workspaceInput{}, restate.TerminalError(fmt.Errorf("workspace %q has no Stripe customer ID", workspaceID))
	}
	if billing.StripeSubscriptionID.Valid && strings.TrimSpace(billing.StripeSubscriptionID.String) != "" {
		return workspaceInput{}, restate.TerminalError(fmt.Errorf("workspace %q has an active API Stripe subscription", workspaceID))
	}
	if billing.StripeDeploySubscriptionID.Valid && strings.TrimSpace(billing.StripeDeploySubscriptionID.String) != "" {
		return workspaceInput{}, restate.TerminalError(fmt.Errorf("workspace %q has an active Compute Stripe subscription", workspaceID))
	}
	return workspaceInput{
		Name:          workspace.Name,
		CustomerID:    billing.StripeCustomerID.String,
		Subscriptions: workspace.Subscriptions,
	}, nil
}

// prepareDraftInvoice creates or resumes one draft and verifies its complete
// state after all missing lines have been added.
func prepareDraftInvoice(ctx context.Context, client *stripe.Client, workspaceID string, input invoiceInput) (*stripe.Invoice, error) {
	logger.Info("preparing Stripe draft invoice")
	invoice, err := findInvoice(ctx, client, workspaceID, input.period)
	if err != nil {
		return nil, err
	}
	if invoice == nil {
		logger.Info("creating Stripe draft invoice")
		invoiceParams := newStripeInvoiceParams(workspaceID, input)
		invoice, err = client.V1Invoices.Create(ctx, invoiceParams)
		if err != nil {
			return nil, fmt.Errorf("create draft Stripe invoice: %w", err)
		}
	} else {
		logger.Info("resuming Stripe draft invoice", "invoice_id", invoice.ID)
	}
	if err = validateDraftInvoice(invoice, input.customerID, workspaceID, input.period); err != nil {
		return nil, err
	}

	missingItems, err := reconcileInvoiceLines(ctx, client, invoice.ID, workspaceID, input.period, input.periodStart, input.periodEnd, input.items)
	if err != nil {
		return nil, err
	}
	logger.Info("adding missing Stripe invoice items", "count", len(missingItems))
	for _, item := range missingItems {
		params := newStripeInvoiceItemParams(input.customerID, invoice.ID, workspaceID, input.period, input.periodStart, input.periodEnd, item)
		if _, err = client.V1InvoiceItems.Create(ctx, params); err != nil {
			return nil, fmt.Errorf("add %q to draft invoice %q: %w", item.Description, invoice.ID, err)
		}
	}
	logger.Info("verifying Stripe draft invoice")
	invoiceID := invoice.ID
	invoice, err = client.V1Invoices.Retrieve(ctx, invoiceID, nil)
	if err != nil {
		return nil, fmt.Errorf("re-read draft invoice %q: %w", invoiceID, err)
	}
	if err = validateDraftInvoice(invoice, input.customerID, workspaceID, input.period); err != nil {
		return nil, err
	}
	missingItems, err = reconcileInvoiceLines(ctx, client, invoice.ID, workspaceID, input.period, input.periodStart, input.periodEnd, input.items)
	if err != nil {
		return nil, err
	}
	if len(missingItems) != 0 {
		return nil, fmt.Errorf("draft invoice %q is still missing %d item(s) after reconciliation", invoice.ID, len(missingItems))
	}
	return invoice, nil
}

// newStripeInvoiceParams creates a standalone, non-advancing invoice request
// that excludes unrelated pending customer items.
func newStripeInvoiceParams(workspaceID string, input invoiceInput) *stripe.InvoiceCreateParams {
	params := &stripe.InvoiceCreateParams{
		AutoAdvance:                 stripe.Bool(false),
		CollectionMethod:            stripe.String(string(stripe.InvoiceCollectionMethodChargeAutomatically)),
		Customer:                    stripe.String(input.customerID),
		PendingInvoiceItemsBehavior: stripe.String("exclude"),
		CustomFields: []*stripe.InvoiceCreateCustomFieldParams{
			{Name: stripe.String("Workspace"), Value: stripe.String(input.workspaceName)},
			{Name: stripe.String("Billing Period"), Value: stripe.String(input.periodStart.Format("January 2006"))},
		},
		Metadata: map[string]string{
			"source":         invoiceSource,
			"workspace_id":   workspaceID,
			"billing_period": input.period,
			"proration":      "none",
		},
	}
	params.SetIdempotencyKey(fmt.Sprintf("%s:invoice:%s:%s", invoiceSource, workspaceID, input.period))
	return params
}

func validateRequestAt(workspaceID string, year, month int, now time.Time) error {
	if strings.TrimSpace(workspaceID) == "" {
		return errors.New("workspace_id is required")
	}
	if year < 2000 || year > 9999 {
		return errors.New("year must be between 2000 and 9999")
	}
	if month < 1 || month > 12 {
		return errors.New("month must be between 1 and 12")
	}
	periodEnd := time.Date(year, time.Month(month), 1, 0, 0, 0, 0, time.UTC).AddDate(0, 1, 0)
	if now.Before(periodEnd) {
		return errors.New("billing month must be complete")
	}
	return nil
}

// newStripeInvoiceItemParams preserves the exact decimal price and attaches a
// deterministic specification used to reconcile interrupted runs.
func newStripeInvoiceItemParams(customerID, invoiceID, workspaceID, period string, periodStart, periodEnd time.Time, item invoiceItem) *stripe.InvoiceItemCreateParams {
	params := &stripe.InvoiceItemCreateParams{
		Currency:     stripe.String(string(stripe.CurrencyUSD)),
		Customer:     stripe.String(customerID),
		Description:  stripe.String(item.Description),
		Discountable: stripe.Bool(false),
		Invoice:      stripe.String(invoiceID),
		Period: &stripe.InvoiceItemCreatePeriodParams{
			Start: stripe.Int64(periodStart.Unix()),
			End:   stripe.Int64(periodEnd.Unix()),
		},
		PriceData: &stripe.InvoiceItemCreatePriceDataParams{
			Currency: stripe.String(string(stripe.CurrencyUSD)),
			Product:  stripe.String(item.ProductID),
		},
		Quantity: stripe.Int64(item.Quantity),
		Metadata: map[string]string{
			"source":         invoiceSource,
			"workspace_id":   workspaceID,
			"billing_period": period,
			"charge":         item.Key,
			"charge_spec":    invoiceItemSpec(item, periodStart, periodEnd),
			"proration":      "none",
		},
	}
	// stripe-go models this value as float64. AddExtra preserves the validated
	// decimal string exactly instead of rounding a monetary value through a float.
	params.AddExtra("price_data[unit_amount_decimal]", item.UnitAmountDecimal)
	params.SetIdempotencyKey(fmt.Sprintf("%s:item:%s:%s:%s", invoiceSource, workspaceID, period, item.Key))
	return params
}

// parseSubscriptions strictly decodes the legacy pricing document so malformed
// pricing cannot silently produce a partial bill.
func parseSubscriptions(raw []byte) (subscriptions, error) {
	var subs subscriptions
	if len(raw) == 0 || string(raw) == "null" || string(raw) == "{}" {
		return subs, errors.New("legacy subscriptions are empty")
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&subs); err != nil {
		return subs, err
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return subs, errors.New("legacy subscriptions contain trailing JSON data")
	}
	if subs.Plan == nil && subs.Support == nil && subs.Verifications == nil && subs.Ratelimits == nil {
		return subs, errors.New("legacy subscriptions contain no billable products")
	}
	return subs, nil
}

// buildInvoiceItems converts fixed prices and usage into the complete Stripe
// line specification without performing external writes.
func buildInvoiceItems(subs subscriptions, verifications, ratelimits int64) ([]invoiceItem, error) {
	items := make([]invoiceItem, 0, 4)
	if subs.Plan != nil {
		item, err := fixedInvoiceItem("Pro plan", "plan", *subs.Plan)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	if subs.Verifications != nil {
		tierItems, err := tieredInvoiceItems("Verifications", "verifications", *subs.Verifications, verifications)
		if err != nil {
			return nil, err
		}
		items = append(items, tierItems...)
	}
	if subs.Ratelimits != nil {
		tierItems, err := tieredInvoiceItems("Ratelimits", "ratelimits", *subs.Ratelimits, ratelimits)
		if err != nil {
			return nil, err
		}
		items = append(items, tierItems...)
	}
	if subs.Support != nil {
		item, err := fixedInvoiceItem("Professional Support", "support", *subs.Support)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, nil
}

// validateInvoiceItemCount enforces Stripe's per-invoice line limit before the
// draft exists.
func validateInvoiceItemCount(items []invoiceItem) error {
	if len(items) > 250 {
		return fmt.Errorf("invoice has %d items; Stripe allows at most 250", len(items))
	}
	return nil
}

// fixedInvoiceItem charges the full configured monthly amount with quantity one.
func fixedInvoiceItem(description, key string, sub fixedSubscription) (invoiceItem, error) {
	if strings.TrimSpace(sub.ProductID) == "" {
		return invoiceItem{}, fmt.Errorf("%s product ID is empty", key)
	}
	if !decimalPattern.MatchString(sub.Cents) {
		return invoiceItem{}, fmt.Errorf("%s cents %q is not a valid Stripe decimal", key, sub.Cents)
	}
	return invoiceItem{Description: description, ProductID: sub.ProductID, Quantity: 1, UnitAmountDecimal: sub.Cents, Key: key}, nil
}

// tieredInvoiceItems consumes each inclusive tier in order and emits lines only
// for paid quantities.
func tieredInvoiceItems(name, key string, sub tieredSubscription, usage int64) ([]invoiceItem, error) {
	if usage < 0 {
		return nil, fmt.Errorf("%s usage cannot be negative", key)
	}
	if err := validateTieredSubscription(key, sub); err != nil {
		return nil, err
	}

	remaining := usage
	items := make([]invoiceItem, 0, len(sub.Tiers))
	for i, tier := range sub.Tiers {
		quantity := remaining
		if tier.LastUnit != nil && quantity > *tier.LastUnit-tier.FirstUnit+1 {
			quantity = *tier.LastUnit - tier.FirstUnit + 1
		}
		remaining -= quantity
		if quantity > 0 && tier.CentsPerUnit != nil {
			lastUnit := "+"
			if tier.LastUnit != nil {
				lastUnit = fmt.Sprintf("-%d", *tier.LastUnit)
			}
			items = append(items, invoiceItem{
				Description:       fmt.Sprintf("%s %d%s", name, tier.FirstUnit, lastUnit),
				ProductID:         sub.ProductID,
				Quantity:          quantity,
				UnitAmountDecimal: *tier.CentsPerUnit,
				Key:               fmt.Sprintf("%s-tier-%d", key, i+1),
			})
		}
	}
	if remaining > 0 {
		return nil, fmt.Errorf("%s has %d unpriced units", key, remaining)
	}
	return items, nil
}

// validateTieredSubscription proves that every unit belongs to one contiguous
// tier ending in an unbounded range.
func validateTieredSubscription(key string, sub tieredSubscription) error {
	if strings.TrimSpace(sub.ProductID) == "" {
		return fmt.Errorf("%s product ID is empty", key)
	}
	if len(sub.Tiers) == 0 {
		return fmt.Errorf("%s tiers are empty", key)
	}
	if sub.Tiers[0].FirstUnit != 1 {
		return fmt.Errorf("%s first tier must start at unit 1", key)
	}

	for i, tier := range sub.Tiers {
		if i > 0 {
			previous := sub.Tiers[i-1]
			if previous.LastUnit == nil || tier.FirstUnit != *previous.LastUnit+1 {
				return fmt.Errorf("%s tier %d is not contiguous", key, i+1)
			}
		}
		if tier.FirstUnit < 1 {
			return fmt.Errorf("%s tier %d has an invalid first unit", key, i+1)
		}
		if tier.LastUnit != nil && *tier.LastUnit < tier.FirstUnit {
			return fmt.Errorf("%s tier %d ends before it starts", key, i+1)
		}
		if tier.LastUnit == nil && i != len(sub.Tiers)-1 {
			return fmt.Errorf("%s only the final tier may be unbounded", key)
		}
		if tier.CentsPerUnit != nil && !decimalPattern.MatchString(*tier.CentsPerUnit) {
			return fmt.Errorf("%s tier %d cents %q is not a valid Stripe decimal", key, i+1, *tier.CentsPerUnit)
		}
	}
	if sub.Tiers[len(sub.Tiers)-1].LastUnit != nil {
		return fmt.Errorf("%s final tier must be unbounded", key)
	}
	return nil
}

// findInvoice scans the whole Stripe account because customer IDs can change
// while the workspace and billing-period uniqueness requirement remains.
func findInvoice(ctx context.Context, client *stripe.Client, workspaceID, period string) (*stripe.Invoice, error) {
	list := client.V1Invoices.List(ctx, &stripe.InvoiceListParams{
		ListParams: stripe.ListParams{Limit: stripe.Int64(100)},
	})
	var match *stripe.Invoice
	for invoice, err := range list.All(ctx) {
		if err != nil {
			return nil, fmt.Errorf("check existing Stripe invoices: %w", err)
		}
		if invoice.Metadata["source"] == invoiceSource &&
			invoice.Metadata["workspace_id"] == workspaceID &&
			invoice.Metadata["billing_period"] == period {
			if match != nil {
				return nil, fmt.Errorf("multiple invoices (%q and %q) exist for workspace %q and period %s", match.ID, invoice.ID, workspaceID, period)
			}
			match = invoice
		}
	}
	return match, nil
}

// validateDraftInvoice fails closed unless an existing invoice remains the
// exact standalone, non-advancing draft owned by this billing run.
func validateDraftInvoice(invoice *stripe.Invoice, customerID, workspaceID, period string) error {
	if invoice == nil {
		return errors.New("Stripe returned an empty invoice")
	}
	if invoice.Customer == nil || invoice.Customer.ID != customerID {
		return fmt.Errorf("invoice %q belongs to an unexpected customer", invoice.ID)
	}
	if invoice.Status != stripe.InvoiceStatusDraft {
		return fmt.Errorf("invoice %q has status %q, expected draft", invoice.ID, invoice.Status)
	}
	if invoice.AutoAdvance {
		return fmt.Errorf("invoice %q has auto-advance enabled", invoice.ID)
	}
	if invoice.CollectionMethod != stripe.InvoiceCollectionMethodChargeAutomatically {
		return fmt.Errorf("invoice %q has unexpected collection method %q", invoice.ID, invoice.CollectionMethod)
	}
	if invoice.Parent != nil {
		return fmt.Errorf("invoice %q is associated with parent type %q", invoice.ID, invoice.Parent.Type)
	}
	if invoice.Metadata["source"] != invoiceSource || invoice.Metadata["workspace_id"] != workspaceID ||
		invoice.Metadata["billing_period"] != period || invoice.Metadata["proration"] != "none" {
		return fmt.Errorf("invoice %q has unexpected billing metadata", invoice.ID)
	}
	return nil
}

// reconcileInvoiceLines loads every attached line before deciding which expected
// charges are safe to create.
func reconcileInvoiceLines(
	ctx context.Context,
	client *stripe.Client,
	invoiceID, workspaceID, period string,
	periodStart, periodEnd time.Time,
	expected []invoiceItem,
) ([]invoiceItem, error) {
	lines := client.V1Invoices.ListLines(ctx, &stripe.InvoiceListLinesParams{
		ListParams: stripe.ListParams{Limit: stripe.Int64(100)},
		Invoice:    stripe.String(invoiceID),
	})
	existing := make([]*stripe.InvoiceLineItem, 0, len(expected))
	for line, err := range lines.All(ctx) {
		if err != nil {
			return nil, fmt.Errorf("list lines for draft invoice %q: %w", invoiceID, err)
		}
		existing = append(existing, line)
	}
	return reconcileExistingInvoiceLines(invoiceID, workspaceID, period, periodStart, periodEnd, expected, existing)
}

// reconcileExistingInvoiceLines rejects changed, duplicate, prorated, or
// subscription-derived lines and returns only absent expected charges.
func reconcileExistingInvoiceLines(
	invoiceID, workspaceID, period string,
	periodStart, periodEnd time.Time,
	expected []invoiceItem,
	existing []*stripe.InvoiceLineItem,
) ([]invoiceItem, error) {
	expectedByKey := make(map[string]invoiceItem, len(expected))
	for _, item := range expected {
		if _, exists := expectedByKey[item.Key]; exists {
			return nil, fmt.Errorf("duplicate expected invoice item key %q", item.Key)
		}
		expectedByKey[item.Key] = item
	}
	seen := make(map[string]struct{}, len(expected))
	for _, line := range existing {
		key := line.Metadata["charge"]
		item, exists := expectedByKey[key]
		if !exists || line.Metadata["source"] != invoiceSource || line.Metadata["workspace_id"] != workspaceID || line.Metadata["billing_period"] != period {
			return nil, fmt.Errorf("draft invoice %q contains unexpected line %q", invoiceID, line.ID)
		}
		if _, exists = seen[key]; exists {
			return nil, fmt.Errorf("draft invoice %q contains duplicate charge %q", invoiceID, key)
		}
		if priceErr := validateInvoiceLinePrice(line, item); priceErr != nil {
			return nil, fmt.Errorf("draft invoice %q charge %q has unexpected pricing: %w", invoiceID, key, priceErr)
		}
		if line.Metadata["charge_spec"] != invoiceItemSpec(item, periodStart, periodEnd) ||
			line.Invoice != invoiceID || line.Description != item.Description || line.Quantity != item.Quantity || line.Discountable ||
			line.Currency != stripe.CurrencyUSD || line.Pricing == nil || line.Pricing.PriceDetails == nil ||
			line.Pricing.PriceDetails.Product != item.ProductID ||
			line.Period == nil || line.Period.Start != periodStart.Unix() || line.Period.End != periodEnd.Unix() ||
			line.Parent == nil || line.Parent.Type != stripe.InvoiceLineItemParentTypeInvoiceItemDetails ||
			line.Parent.InvoiceItemDetails == nil || line.Parent.InvoiceItemDetails.Proration ||
			line.Parent.InvoiceItemDetails.Subscription != "" {
			return nil, fmt.Errorf("draft invoice %q charge %q does not match the expected specification", invoiceID, key)
		}
		seen[key] = struct{}{}
	}

	missing := make([]invoiceItem, 0, len(expected)-len(seen))
	for _, item := range expected {
		if _, exists := seen[item.Key]; !exists {
			missing = append(missing, item)
		}
	}
	return missing, nil
}

// validateInvoiceLinePrice reads Stripe's raw decimal string because the typed
// SDK field uses float64 and cannot preserve financial precision.
func validateInvoiceLinePrice(line *stripe.InvoiceLineItem, item invoiceItem) error {
	if line.LastResponse == nil || len(line.LastResponse.RawJSON) == 0 {
		return errors.New("Stripe response is missing raw line JSON")
	}
	var raw struct {
		Pricing struct {
			UnitAmountDecimal string `json:"unit_amount_decimal"`
		} `json:"pricing"`
	}
	if err := json.Unmarshal(line.LastResponse.RawJSON, &raw); err != nil {
		return fmt.Errorf("decode raw Stripe line: %w", err)
	}
	actual, err := parseStripeDecimal(raw.Pricing.UnitAmountDecimal)
	if err != nil {
		return fmt.Errorf("parse Stripe unit amount: %w", err)
	}
	expected, err := parseStripeDecimal(item.UnitAmountDecimal)
	if err != nil {
		return fmt.Errorf("parse expected unit amount: %w", err)
	}
	if actual.Cmp(expected) != 0 {
		return fmt.Errorf("unit amount is %q, expected %q", raw.Pricing.UnitAmountDecimal, item.UnitAmountDecimal)
	}
	expectedAmount, err := roundedLineAmount(expected, item.Quantity)
	if err != nil {
		return err
	}
	if line.Amount != expectedAmount || line.Subtotal != expectedAmount {
		return fmt.Errorf("amount is %d and subtotal is %d, expected %d", line.Amount, line.Subtotal, expectedAmount)
	}
	return nil
}

// parseStripeDecimal parses the workflow's non-negative fixed-point format
// into an exact rational value.
func parseStripeDecimal(value string) (*big.Rat, error) {
	if !decimalPattern.MatchString(value) {
		return nil, fmt.Errorf("invalid decimal %q", value)
	}
	decimal, ok := new(big.Rat).SetString(value)
	if !ok {
		return nil, fmt.Errorf("invalid decimal %q", value)
	}
	return decimal, nil
}

// roundedLineAmount reproduces Stripe's nearest-unit rounding for a positive
// decimal unit amount multiplied by an integer quantity.
func roundedLineAmount(unitAmount *big.Rat, quantity int64) (int64, error) {
	if quantity < 0 || unitAmount.Sign() < 0 {
		return 0, errors.New("line amount inputs must be non-negative")
	}
	totalNumerator := new(big.Int).Mul(unitAmount.Num(), big.NewInt(quantity))
	quotient, remainder := new(big.Int), new(big.Int)
	quotient.QuoRem(totalNumerator, unitAmount.Denom(), remainder)
	if new(big.Int).Lsh(remainder, 1).Cmp(unitAmount.Denom()) >= 0 {
		quotient.Add(quotient, big.NewInt(1))
	}
	if !quotient.IsInt64() {
		return 0, errors.New("rounded line amount exceeds int64")
	}
	return quotient.Int64(), nil
}

// invoiceItemSpec hashes fields that stripe-go cannot compare exactly after it
// decodes decimal prices through float64.
func invoiceItemSpec(item invoiceItem, periodStart, periodEnd time.Time) string {
	canonical := fmt.Sprintf("%s\x00%s\x00%d\x00%s\x00%d\x00%d", item.Description, item.ProductID, item.Quantity, item.UnitAmountDecimal, periodStart.Unix(), periodEnd.Unix())
	return fmt.Sprintf("%x", sha256.Sum256([]byte(canonical)))
}
