package legacybilling

import (
	"encoding/json"
	"net/url"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	stripe "github.com/stripe/stripe-go/v86"
	"github.com/stripe/stripe-go/v86/form"
)

// TestBuildInvoiceItemsUsesFullFixedPricesAndTierUsage protects fixed fees from
// accidental proration while checking that free usage is consumed before paid usage.
func TestBuildInvoiceItemsUsesFullFixedPricesAndTierUsage(t *testing.T) {
	freeEnd := int64(150_000)
	paid := "0.01"

	items, err := buildInvoiceItems(subscriptions{
		Plan: &fixedSubscription{ProductID: "prod_plan", Cents: "2500"},
		Verifications: &tieredSubscription{
			ProductID: "prod_verifications",
			Tiers: []billingTier{
				{FirstUnit: 1, LastUnit: &freeEnd},
				{FirstUnit: 150_001, CentsPerUnit: &paid},
			},
		},
		Support: &fixedSubscription{ProductID: "prod_support", Cents: "10000"},
	}, 175_000, 0)
	require.NoError(t, err)
	require.Equal(t, []invoiceItem{
		{Description: "Pro plan", ProductID: "prod_plan", Quantity: 1, UnitAmountDecimal: "2500", Key: "plan"},
		{Description: "Verifications 150001+", ProductID: "prod_verifications", Quantity: 25_000, UnitAmountDecimal: "0.01", Key: "verifications-tier-2"},
		{Description: "Professional Support", ProductID: "prod_support", Quantity: 1, UnitAmountDecimal: "10000", Key: "support"},
	}, items)
}

func TestTieredInvoiceItemsInclusiveBoundaries(t *testing.T) {
	freeEnd := int64(100)
	paid := "0.25"
	subscription := tieredSubscription{
		ProductID: "prod_verifications",
		Tiers: []billingTier{
			{FirstUnit: 1, LastUnit: &freeEnd},
			{FirstUnit: 101, CentsPerUnit: &paid},
		},
	}
	tests := []struct {
		name         string
		usage        int64
		paidQuantity int64
	}{
		{name: "zero usage", usage: 0, paidQuantity: 0},
		{name: "last free unit", usage: 100, paidQuantity: 0},
		{name: "first paid unit", usage: 101, paidQuantity: 1},
		{name: "multiple paid units", usage: 150, paidQuantity: 50},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			items, err := tieredInvoiceItems("Verifications", "verifications", subscription, test.usage)
			require.NoError(t, err)
			if test.paidQuantity == 0 {
				require.Empty(t, items)
				return
			}
			require.Len(t, items, 1)
			require.Equal(t, test.paidQuantity, items[0].Quantity)
		})
	}
}

func TestTieredInvoiceItemsRejectsIncompletePricing(t *testing.T) {
	last := int64(100)
	price := "1"

	_, err := tieredInvoiceItems("Verifications", "verifications", tieredSubscription{
		ProductID: "prod_verifications",
		Tiers: []billingTier{{
			FirstUnit:    1,
			LastUnit:     &last,
			CentsPerUnit: &price,
		}},
	}, 101)
	require.EqualError(t, err, "verifications final tier must be unbounded")
}

func TestParseSubscriptionsRejectsEmptyConfiguration(t *testing.T) {
	_, err := parseSubscriptions([]byte(`{}`))
	require.EqualError(t, err, "legacy subscriptions are empty")
}

func TestParseSubscriptionsTreatsOmittedNullableTierFieldsAsNull(t *testing.T) {
	subs, err := parseSubscriptions([]byte(`{
		"verifications": {
			"productId": "prod_verifications",
			"tiers": [{"firstUnit": 1}]
		}
	}`))
	require.NoError(t, err)
	items, err := buildInvoiceItems(subs, 1, 0)
	require.NoError(t, err)
	require.Empty(t, items)
}

func TestParseSubscriptionsRejectsUnknownFields(t *testing.T) {
	_, err := parseSubscriptions([]byte(`{"plan":{"productId":"prod_plan","cents":"2500","unknown":true}}`))
	require.ErrorContains(t, err, `unknown field "unknown"`)
}

// TestStripeInvoiceParamsRemainStandaloneDraft guarantees that creating the
// invoice cannot attach it to a subscription or enable automatic finalization.
func TestStripeInvoiceParamsRemainStandaloneDraft(t *testing.T) {
	periodStart := time.Date(2026, time.July, 1, 0, 0, 0, 0, time.UTC)
	params := newStripeInvoiceParams("ws_123", invoiceInput{
		workspaceName: "Acme",
		customerID:    "cus_123",
		period:        "2026-07",
		periodStart:   periodStart,
	})

	require.False(t, *params.AutoAdvance)
	require.Nil(t, params.Subscription)
	require.Equal(t, "exclude", *params.PendingInvoiceItemsBehavior)
	require.Equal(t, string(stripe.InvoiceCollectionMethodChargeAutomatically), *params.CollectionMethod)
	require.Equal(t, "none", params.Metadata["proration"])
}

// TestStripeInvoiceItemParamsPreserveDecimalString protects fractional prices
// from monetary corruption through binary floating-point conversion.
func TestStripeInvoiceItemParamsPreserveDecimalString(t *testing.T) {
	periodStart := time.Date(2026, time.July, 1, 0, 0, 0, 0, time.UTC)
	periodEnd := periodStart.AddDate(0, 1, 0).Add(-time.Second)
	params := newStripeInvoiceItemParams("cus_123", "in_123", "ws_123", "2026-07", periodStart, periodEnd, invoiceItem{
		Description:       "Verifications (tier 1)",
		ProductID:         "prod_123",
		Quantity:          42,
		UnitAmountDecimal: "0.00123456789",
		Key:               "verifications:1",
	})

	require.Nil(t, params.PriceData.UnitAmountDecimal)
	require.Equal(t, "0.00123456789", params.Extra.Get("price_data[unit_amount_decimal]"))
	require.False(t, *params.Discountable)
	require.Equal(t, "none", params.Metadata["proration"])
	values := &form.Values{}
	form.AppendTo(values, params)
	encoded, err := url.ParseQuery(values.Encode())
	require.NoError(t, err)
	require.Equal(t, "0.00123456789", encoded.Get("price_data[unit_amount_decimal]"))
}

// TestRoundedLineAmount protects Stripe's nearest-cent, half-up behavior used
// to independently verify the integer amount returned for a decimal price.
func TestRoundedLineAmount(t *testing.T) {
	tests := []struct {
		name     string
		decimal  string
		quantity int64
		amount   int64
	}{
		{name: "below half", decimal: "0.01", quantity: 49, amount: 0},
		{name: "half rounds up", decimal: "0.01", quantity: 50, amount: 1},
		{name: "whole cents", decimal: "2500", quantity: 1, amount: 2500},
		{name: "fractional total", decimal: "0.25", quantity: 10, amount: 3},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			decimal, err := parseStripeDecimal(test.decimal)
			require.NoError(t, err)
			amount, err := roundedLineAmount(decimal, test.quantity)
			require.NoError(t, err)
			require.Equal(t, test.amount, amount)
		})
	}
}

func TestValidateInvoiceItemCount(t *testing.T) {
	require.NoError(t, validateInvoiceItemCount(make([]invoiceItem, 250)))
	require.EqualError(t, validateInvoiceItemCount(make([]invoiceItem, 251)), "invoice has 251 items; Stripe allows at most 250")
}

func TestValidateDraftInvoice(t *testing.T) {
	valid := &stripe.Invoice{
		ID:               "in_123",
		AutoAdvance:      false,
		CollectionMethod: stripe.InvoiceCollectionMethodChargeAutomatically,
		Customer:         &stripe.Customer{ID: "cus_123"},
		Status:           stripe.InvoiceStatusDraft,
		Metadata:         map[string]string{"source": invoiceSource, "workspace_id": "ws_123", "billing_period": "2026-07", "proration": "none"},
	}
	require.NoError(t, validateDraftInvoice(valid, "cus_123", "ws_123", "2026-07"))

	wrongCustomer := *valid
	wrongCustomer.Customer = &stripe.Customer{ID: "cus_other"}
	require.ErrorContains(t, validateDraftInvoice(&wrongCustomer, "cus_123", "ws_123", "2026-07"), "unexpected customer")

	autoAdvance := *valid
	autoAdvance.AutoAdvance = true
	require.ErrorContains(t, validateDraftInvoice(&autoAdvance, "cus_123", "ws_123", "2026-07"), "auto-advance enabled")

	subscription := *valid
	subscription.Parent = &stripe.InvoiceParent{Type: stripe.InvoiceParentTypeSubscriptionDetails}
	require.ErrorContains(t, validateDraftInvoice(&subscription, "cus_123", "ws_123", "2026-07"), "associated with parent")
}

// TestReconcileExistingInvoiceLinesResumesPartialDraft guarantees that a retry
// adds absent charges without duplicating or accepting changed charges.
func TestReconcileExistingInvoiceLinesResumesPartialDraft(t *testing.T) {
	periodStart := time.Date(2026, time.July, 1, 0, 0, 0, 0, time.UTC)
	periodEnd := periodStart.AddDate(0, 1, 0).Add(-time.Second)
	expected := []invoiceItem{
		{Description: "Pro plan", ProductID: "prod_plan", Quantity: 1, UnitAmountDecimal: "2500", Key: "plan"},
		{Description: "Verifications 1+", ProductID: "prod_usage", Quantity: 42, UnitAmountDecimal: "0.01", Key: "verifications-tier-1"},
	}
	existing := expectedInvoiceLine(t, "il_123", "in_123", "ws_123", "2026-07", periodStart, periodEnd, expected[0])

	missing, err := reconcileExistingInvoiceLines("in_123", "ws_123", "2026-07", periodStart, periodEnd, expected, []*stripe.InvoiceLineItem{existing})
	require.NoError(t, err)
	require.Equal(t, []invoiceItem{expected[1]}, missing)

	existing.Metadata["charge_spec"] = "changed"
	_, err = reconcileExistingInvoiceLines("in_123", "ws_123", "2026-07", periodStart, periodEnd, expected, []*stripe.InvoiceLineItem{existing})
	require.ErrorContains(t, err, "does not match the expected specification")

	existing.Metadata["charge_spec"] = invoiceItemSpec(expected[0], periodStart, periodEnd)
	existing.LastResponse.RawJSON = []byte(`{"pricing":{"unit_amount_decimal":"5000"}}`)
	_, err = reconcileExistingInvoiceLines("in_123", "ws_123", "2026-07", periodStart, periodEnd, expected, []*stripe.InvoiceLineItem{existing})
	require.ErrorContains(t, err, `unit amount is "5000", expected "2500"`)
}

// expectedInvoiceLine creates the exact standalone Stripe line shape accepted
// by reconciliation tests.
func expectedInvoiceLine(t *testing.T, id, invoiceID, workspaceID, period string, periodStart, periodEnd time.Time, item invoiceItem) *stripe.InvoiceLineItem {
	t.Helper()
	rawJSON, err := json.Marshal(map[string]any{"pricing": map[string]string{"unit_amount_decimal": item.UnitAmountDecimal}})
	require.NoError(t, err)
	unitAmount, err := parseStripeDecimal(item.UnitAmountDecimal)
	require.NoError(t, err)
	amount, err := roundedLineAmount(unitAmount, item.Quantity)
	require.NoError(t, err)
	return &stripe.InvoiceLineItem{
		APIResource:  stripe.APIResource{LastResponse: &stripe.APIResponse{RawJSON: rawJSON}},
		ID:           id,
		Amount:       amount,
		Subtotal:     amount,
		Invoice:      invoiceID,
		Description:  item.Description,
		Discountable: false,
		Quantity:     item.Quantity,
		Period:       &stripe.Period{Start: periodStart.Unix(), End: periodEnd.Unix()},
		Currency:     stripe.CurrencyUSD,
		Pricing:      &stripe.InvoiceLineItemPricing{PriceDetails: &stripe.InvoiceLineItemPricingPriceDetails{Product: item.ProductID}},
		Parent:       &stripe.InvoiceLineItemParent{Type: stripe.InvoiceLineItemParentTypeInvoiceItemDetails, InvoiceItemDetails: &stripe.InvoiceLineItemParentInvoiceItemDetails{}},
		Metadata: map[string]string{"source": invoiceSource, "workspace_id": workspaceID, "billing_period": period, "charge": item.Key,
			"charge_spec": invoiceItemSpec(item, periodStart, periodEnd), "proration": "none"},
	}
}

func TestValidateConfig(t *testing.T) {
	now := time.Date(2026, time.August, 1, 0, 0, 0, 0, time.UTC)
	require.NoError(t, validateRequestAt("ws_123", 2026, 7, now))
	require.EqualError(t, validateRequestAt("ws_123", 2026, 13, now), "month must be between 1 and 12")
	require.EqualError(t, validateRequestAt("ws_123", 2026, 8, now), "billing month must be complete")
}

// TestWorkspaceInputIsJournalSerializable protects billing state from being
// lost when Restate records and replays the MySQL step result.
func TestWorkspaceInputIsJournalSerializable(t *testing.T) {
	expected := workspaceInput{Name: "Acme", CustomerID: "cus_123", Subscriptions: []byte(`{"plan":true}`)}

	encoded, err := json.Marshal(expected)
	require.NoError(t, err)
	var actual workspaceInput
	require.NoError(t, json.Unmarshal(encoded, &actual))
	require.Equal(t, expected, actual)
}
