// Package invoicecloser lists and finalizes Stripe draft invoices for month-end close.
//
// Stripe creates renewal drafts at period roll and would auto-finalize ~1h later.
// The close pushes final usage first, then finalizes here.
package invoicecloser
