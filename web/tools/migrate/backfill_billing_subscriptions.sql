-- Backfill billing_subscriptions from the legacy workspace_billing.stripe_subscription_id.
--
-- Ordering: run once AFTER billing_subscriptions exists and the new code is live
-- (the billing-split deploy) and BEFORE dropping workspace_billing.stripe_subscription_id
-- (the workspaces-column-drop deploy). It carries the pre-split API subscriptions
-- into the table so the app, which now reads them from billing_subscriptions,
-- keeps seeing existing customers.
--
-- Only the API product needs backfilling: stripe_subscription_id holds live API
-- subscriptions today, while Deploy subscriptions were never stored on
-- workspace_billing (that column was dropped before it ever shipped) and are
-- written straight to billing_subscriptions by the subscribe/link flows.
--
-- INSERT IGNORE makes it idempotent: the unique(workspace_id, product) turns a
-- re-run, or a row the app already wrote, into a no-op.
INSERT IGNORE INTO `billing_subscriptions`
  (`workspace_id`, `product`, `stripe_subscription_id`, `created_at`)
SELECT `workspace_id`, 'api', `stripe_subscription_id`, UNIX_TIMESTAMP() * 1000
  FROM `workspace_billing`
  WHERE `stripe_subscription_id` IS NOT NULL;
