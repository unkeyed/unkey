// Package billing provides customer billing functionality that enables Unkey
// customers to bill their end users through Stripe based on API usage.
//
// The billing system uses Stripe Connect to facilitate payments between
// customers and their end users, with Unkey acting as the platform that
// tracks usage and generates invoices.
//
// Key components:
//   - StripeConnectService: Handles OAuth flow for connecting Stripe accounts
//   - PricingModelService: Manages pricing configurations
//   - EndUserService: Manages end user records and Stripe customers
//   - BillingService: Generates invoices and processes payments
//   - UsageAggregator: Aggregates usage data from ClickHouse
package billing
