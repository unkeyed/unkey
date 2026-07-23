package billingreconcile

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCheckTotal(t *testing.T) {
	// oneLine keeps subtotal == sum of lines so only the term under test can
	// break the decomposition.
	oneLine := func(amount int64) []InvoiceLine {
		return []InvoiceLine{planFeeLine("plan.pro", "f", amount, 0)}
	}

	t.Run("no adjustments: total equals subtotal", func(t *testing.T) {
		inv := Invoice{ID: "in", SubtotalCents: 5000, TotalCents: 5000, Lines: oneLine(5000)} //nolint:exhaustruct
		require.Empty(t, checkTotal(inv))
	})

	t.Run("every adjustment type present, exhaustively accounted for", func(t *testing.T) {
		inv := Invoice{ //nolint:exhaustruct
			ID:                  "in",
			SubtotalCents:       5000,
			Lines:               oneLine(5000),
			DiscountAmounts:     []DiscountAmount{{AmountCents: 200}},
			PretaxCreditAmounts: []PretaxCreditAmount{{AmountCents: 300, Type: PretaxCreditBalanceTransaction}},
			Taxes:               []TaxAmount{{AmountCents: 90}},
			AmountShippingCents: 10,
			TotalCents:          4600, // 5000 - 200 - 300 + 10 + 90
		}
		require.Empty(t, checkTotal(inv))
	})

	t.Run("coupon discount mirror is not double-counted", func(t *testing.T) {
		inv := Invoice{ //nolint:exhaustruct
			ID:                  "in",
			SubtotalCents:       5000,
			Lines:               oneLine(5000),
			DiscountAmounts:     []DiscountAmount{{AmountCents: 250}},
			PretaxCreditAmounts: []PretaxCreditAmount{{AmountCents: 250, Type: PretaxCreditDiscount}},
			TotalCents:          4750, // subtotal - 250, the discount counted once
		}
		require.Empty(t, checkTotal(inv))
	})

	t.Run("subtotal does not equal the sum of lines: structural", func(t *testing.T) {
		inv := Invoice{ID: "in", SubtotalCents: 5000, TotalCents: 5000, Lines: oneLine(4900)} //nolint:exhaustruct
		findings := checkTotal(inv)
		require.Len(t, findings, 1)
		require.Contains(t, findings[0].Detail, "lines sum to")
	})

	t.Run("unexplained cent: structural", func(t *testing.T) {
		inv := Invoice{ID: "in", SubtotalCents: 5000, TotalCents: 4999, Lines: oneLine(5000)} //nolint:exhaustruct
		findings := checkTotal(inv)
		require.Len(t, findings, 1)
		require.Contains(t, findings[0].Detail, "unexplained")
	})

	t.Run("unrecognized adjustment type: structural (type + the cent it leaves)", func(t *testing.T) {
		inv := Invoice{ //nolint:exhaustruct
			ID:                  "in",
			SubtotalCents:       5000,
			Lines:               oneLine(5000),
			PretaxCreditAmounts: []PretaxCreditAmount{{AmountCents: 500, Type: "margin"}},
			TotalCents:          4500, // Stripe applied it, so the stored total reflects it
		}
		findings := checkTotal(inv)
		require.Len(t, findings, 2)
		require.Contains(t, findings[0].Detail, `unrecognized adjustment type "margin"`)
	})
}

func TestFoldVerdict(t *testing.T) {
	require.Equal(t, VerdictClean, foldVerdict(nil))
	require.Equal(t, VerdictLateDataUnderbill, foldVerdict([]Finding{
		{Check: CheckQuantity, Class: VerdictLateDataUnderbill}, //nolint:exhaustruct
	}))
	require.Equal(t, VerdictOverbill, foldVerdict([]Finding{
		{Check: CheckQuantity, Class: VerdictLateDataUnderbill}, //nolint:exhaustruct
		{Check: CheckQuantity, Class: VerdictOverbill},          //nolint:exhaustruct
	}))
	require.Equal(t, VerdictStructural, foldVerdict([]Finding{
		{Check: CheckQuantity, Class: VerdictOverbill}, //nolint:exhaustruct
		{Check: CheckTotal, Class: VerdictStructural},  //nolint:exhaustruct
	}))
}
