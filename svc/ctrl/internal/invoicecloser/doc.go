// Package invoicecloser finalizes Stripe invoices for the month-end Deploy
// billing close.
//
// At each period roll, Stripe creates the renewal invoice as a draft and
// would auto-finalize it about an hour later. The close flow finalizes it
// explicitly instead, after pushing the period's final usage, so the metered
// lines bill the complete month rather than whatever the last hourly push
// happened to deliver. This package is the narrow Stripe surface that flow
// needs: list a customer's draft renewal invoices, finalize one.
package invoicecloser
