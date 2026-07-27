import { beforeEach, describe, expect, it, vi } from "vitest";

// Chainable db mock: transaction(cb) runs cb with a tx whose
// update().set().where() and insert().values().onDuplicateKeyUpdate() resolve.
// findFirst returns the workspace row with its `billing` relation (plan) and its
// `billingSubscriptions` rows, since the linker reads the recorded subscription
// from billing_subscriptions and writes plan/customer to workspace_billing.
const h = vi.hoisted(() => {
  const where = vi.fn().mockResolvedValue(undefined);
  const set = vi.fn().mockReturnValue({ where });
  const update = vi.fn().mockReturnValue({ set });
  const onDuplicateKeyUpdate = vi.fn().mockResolvedValue(undefined);
  const values = vi.fn().mockReturnValue({ onDuplicateKeyUpdate });
  const insert = vi.fn().mockReturnValue({ values });
  const findFirst = vi.fn();
  const transaction = vi.fn(async (cb: (tx: unknown) => unknown) => cb({ update, insert }));
  const insertAuditLogs = vi.fn();
  return {
    where,
    set,
    update,
    insert,
    values,
    onDuplicateKeyUpdate,
    findFirst,
    transaction,
    insertAuditLogs,
  };
});

vi.mock("@/lib/db", () => ({
  db: { query: { workspaces: { findFirst: h.findFirst } }, transaction: h.transaction },
  eq: vi.fn(),
  schema: { workspaces: { id: {} }, workspaceBilling: { workspaceId: {} } },
}));
vi.mock("@unkey/db", () => ({
  and: vi.fn(),
  eq: vi.fn(),
  schema: { billingSubscriptions: { workspaceId: {}, product: {} } },
}));
vi.mock("@/lib/audit", () => ({ insertAuditLogs: h.insertAuditLogs }));
vi.mock("@/lib/env", () => ({
  stripeEnv: () => ({
    STRIPE_PRODUCT_IDS_PRO: ["prod_api"],
    STRIPE_PRODUCT_IDS_ENTERPRISE: [],
  }),
}));

import Stripe from "stripe";
import { linkApiSubscription, linkDeploySubscription } from "./linkDeploySubscription";

const WORKSPACE_ID = "ws_1";
const AUDIT = {
  actor: { type: "system" as const, id: "stripe" },
  location: "",
  userAgent: undefined,
};

function session(overrides: Partial<Stripe.Checkout.Session> = {}): Stripe.Checkout.Session {
  return {
    client_reference_id: WORKSPACE_ID,
    status: "complete",
    payment_status: "paid",
    customer: "cus_1",
    subscription: "sub_1",
    ...overrides,
  } as unknown as Stripe.Checkout.Session;
}

function subscription(overrides: Partial<Stripe.Subscription> = {}): Stripe.Subscription {
  return {
    id: "sub_1",
    status: "active",
    items: { data: [{ price: { id: "price_starter", metadata: { plan: "starter" } } }] },
    ...overrides,
  } as unknown as Stripe.Subscription;
}

function apiSession(overrides: Partial<Stripe.Checkout.Session> = {}): Stripe.Checkout.Session {
  return session({ metadata: { unkey_product: "api" }, ...overrides });
}

function apiSubscription(overrides: Partial<Stripe.Subscription> = {}): Stripe.Subscription {
  return subscription({
    items: { data: [{ id: "si_api", price: { id: "price_api", product: "prod_api" } }] },
    ...overrides,
  } as unknown as Partial<Stripe.Subscription>);
}

function apiProduct(overrides: Partial<Stripe.Product> = {}): Stripe.Product {
  return {
    id: "prod_api",
    name: "Pro",
    metadata: {
      quota_requests_per_month: "1000000",
      quota_logs_retention_days: "30",
      quota_audit_logs_retention_days: "90",
    },
    ...overrides,
  } as unknown as Stripe.Product;
}

const customersUpdate = vi.fn(async () => ({}));

function stubStripe(opts: {
  session?: Stripe.Checkout.Session;
  sub?: Stripe.Subscription;
  /** Per-id overrides for subscriptions.retrieve; an Error value is thrown. */
  subsById?: Record<string, Stripe.Subscription | Error>;
  customer?: {
    deleted?: boolean;
    invoice_settings?: { default_payment_method?: string | null };
    default_source?: string | null;
  };
  product?: Stripe.Product;
  sessionError?: unknown;
}): Stripe {
  return {
    checkout: {
      sessions: {
        retrieve: vi.fn(async () => {
          if (opts.sessionError !== undefined) {
            throw opts.sessionError;
          }
          return opts.session ?? session();
        }),
      },
    },
    subscriptions: {
      retrieve: vi.fn(async (id: string) => {
        const byId = opts.subsById?.[id];
        if (byId instanceof Error) {
          throw byId;
        }
        return byId ?? opts.sub ?? subscription();
      }),
    },
    customers: {
      retrieve: vi.fn(
        async () =>
          opts.customer ?? {
            deleted: false,
            invoice_settings: { default_payment_method: null },
            default_source: null,
          },
      ),
      update: customersUpdate,
    },
    products: {
      retrieve: vi.fn(async () => opts.product ?? apiProduct()),
    },
  } as unknown as Stripe;
}

// A real Stripe "resource_missing" error (the only shape that means the session
// genuinely does not exist). instanceof Stripe.errors.StripeError must hold.
const RESOURCE_MISSING = new Stripe.errors.StripeInvalidRequestError({
  type: "invalid_request_error",
  code: "resource_missing",
  message: "No such checkout session",
});

describe("linkDeploySubscription", () => {
  beforeEach(() => {
    h.findFirst.mockReset();
    h.transaction.mockClear();
    h.insertAuditLogs.mockClear();
    h.update.mockClear();
    h.set.mockClear();
    h.where.mockClear();
    h.insert.mockClear();
    h.values.mockClear();
    h.onDuplicateKeyUpdate.mockClear();
    customersUpdate.mockClear();
  });

  it("rejects a session belonging to another workspace, without writing", async () => {
    const stripe = stubStripe({ session: session({ client_reference_id: "ws_other" }) });
    const result = await linkDeploySubscription(stripe, {
      sessionId: "cs_1",
      expectedWorkspaceId: WORKSPACE_ID,
      audit: AUDIT,
    });
    expect(result).toEqual({ ok: false, reason: "forbidden", message: expect.any(String) });
    expect(h.transaction).not.toHaveBeenCalled();
  });

  it("rejects an unpaid/incomplete session (entitlement-bypass guard)", async () => {
    const stripe = stubStripe({ session: session({ payment_status: "unpaid", status: "open" }) });
    const result = await linkDeploySubscription(stripe, {
      sessionId: "cs_1",
      expectedWorkspaceId: WORKSPACE_ID,
      audit: AUDIT,
    });
    expect(result).toMatchObject({ ok: false, reason: "not_paid" });
    expect(h.transaction).not.toHaveBeenCalled();
  });

  it("rejects a subscription that is not active/trialing", async () => {
    const stripe = stubStripe({ sub: subscription({ status: "incomplete" }) });
    const result = await linkDeploySubscription(stripe, {
      sessionId: "cs_1",
      expectedWorkspaceId: WORKSPACE_ID,
      audit: AUDIT,
    });
    expect(result).toMatchObject({ ok: false, reason: "not_active" });
    expect(h.transaction).not.toHaveBeenCalled();
  });

  it("rejects a subscription with no recognized Compute plan", async () => {
    const stripe = stubStripe({
      sub: subscription({
        items: { data: [{ price: { id: "price_api", metadata: {} } }] },
      } as unknown as Partial<Stripe.Subscription>),
    });
    const result = await linkDeploySubscription(stripe, {
      sessionId: "cs_1",
      expectedWorkspaceId: WORKSPACE_ID,
      audit: AUDIT,
    });
    expect(result).toMatchObject({ ok: false, reason: "no_deploy_plan" });
    expect(h.transaction).not.toHaveBeenCalled();
  });

  it("returns session_not_found when Stripe reports the session is missing", async () => {
    const stripe = stubStripe({ sessionError: RESOURCE_MISSING });
    const result = await linkDeploySubscription(stripe, {
      sessionId: "cs_missing",
      expectedWorkspaceId: WORKSPACE_ID,
      audit: AUDIT,
    });
    expect(result).toMatchObject({ ok: false, reason: "session_not_found" });
  });

  it("rethrows a transient Stripe error so the webhook retries (does not orphan)", async () => {
    // A non-resource_missing failure (network/429/5xx) must propagate, not be
    // swallowed as session_not_found — otherwise the webhook acks and never retries.
    const stripe = stubStripe({ sessionError: new Error("network blip") });
    await expect(
      linkDeploySubscription(stripe, {
        sessionId: "cs_1",
        expectedWorkspaceId: WORKSPACE_ID,
        audit: AUDIT,
      }),
    ).rejects.toThrow("network blip");
    expect(h.transaction).not.toHaveBeenCalled();
  });

  it("refuses to link a subscription set to cancel, without writing", async () => {
    // A checkout.session.completed redelivered after the user cancelled would
    // otherwise resurrect the plan the cancel cleared.
    h.findFirst.mockResolvedValue({
      id: WORKSPACE_ID,
      orgId: "org_1",
      billing: { plan: null },
      billingSubscriptions: [],
    });
    const stripe = stubStripe({
      sub: subscription({ cancel_at_period_end: true } as Partial<Stripe.Subscription>),
    });
    const result = await linkDeploySubscription(stripe, {
      sessionId: "cs_1",
      expectedWorkspaceId: WORKSPACE_ID,
      audit: AUDIT,
    });
    expect(result).toMatchObject({ ok: false, reason: "not_active" });
    expect(h.transaction).not.toHaveBeenCalled();
  });

  it("writes customer + subscription + plan for a paid, active, unlinked workspace", async () => {
    h.findFirst.mockResolvedValue({
      id: WORKSPACE_ID,
      orgId: "org_1",
      billing: { plan: null },
      billingSubscriptions: [],
    });
    const stripe = stubStripe({});
    const result = await linkDeploySubscription(stripe, {
      sessionId: "cs_1",
      expectedWorkspaceId: WORKSPACE_ID,
      audit: AUDIT,
    });
    expect(result).toEqual({ ok: true, plan: "starter", alreadyLinked: false });
    expect(h.transaction).toHaveBeenCalledOnce();
    expect(h.set).toHaveBeenCalledWith({
      stripeCustomerId: "cus_1",
      plan: "starter",
    });
    expect(h.values).toHaveBeenCalledWith({
      workspaceId: WORKSPACE_ID,
      product: "compute",
      stripeSubscriptionId: "sub_1",
    });
    expect(h.insertAuditLogs).toHaveBeenCalledOnce();
  });

  it("is an idempotent no-op when the same subscription+plan is already linked", async () => {
    h.findFirst.mockResolvedValue({
      id: WORKSPACE_ID,
      orgId: "org_1",
      billing: { plan: "starter" },
      billingSubscriptions: [{ product: "compute", stripeSubscriptionId: "sub_1" }],
    });
    const stripe = stubStripe({});
    const result = await linkDeploySubscription(stripe, {
      sessionId: "cs_1",
      expectedWorkspaceId: WORKSPACE_ID,
      audit: AUDIT,
    });
    expect(result).toEqual({ ok: true, plan: "starter", alreadyLinked: true });
    expect(h.transaction).not.toHaveBeenCalled();
  });

  it("hard-fails rather than repoint a workspace with a different LIVE subscription", async () => {
    h.findFirst.mockResolvedValue({
      id: WORKSPACE_ID,
      orgId: "org_1",
      billing: { plan: "pro" },
      billingSubscriptions: [{ product: "compute", stripeSubscriptionId: "sub_other" }],
    });
    // The default stub returns an active subscription for any id, so the
    // recorded sub_other reads as live.
    const stripe = stubStripe({});
    const result = await linkDeploySubscription(stripe, {
      sessionId: "cs_1",
      expectedWorkspaceId: WORKSPACE_ID,
      audit: AUDIT,
    });
    expect(result).toMatchObject({ ok: false, reason: "subscription_conflict" });
    expect(h.transaction).not.toHaveBeenCalled();
  });

  it("repoints a workspace whose recorded subscription is dead (cancel-then-resubscribe)", async () => {
    h.findFirst.mockResolvedValue({
      id: WORKSPACE_ID,
      orgId: "org_1",
      billing: { plan: null },
      billingSubscriptions: [{ product: "compute", stripeSubscriptionId: "sub_dead" }],
    });
    const stripe = stubStripe({
      subsById: { sub_dead: subscription({ id: "sub_dead", status: "canceled" }) },
    });
    const result = await linkDeploySubscription(stripe, {
      sessionId: "cs_1",
      expectedWorkspaceId: WORKSPACE_ID,
      audit: AUDIT,
    });
    expect(result).toEqual({ ok: true, plan: "starter", alreadyLinked: false });
    expect(h.set).toHaveBeenCalledWith({
      stripeCustomerId: "cus_1",
      plan: "starter",
    });
    expect(h.values).toHaveBeenCalledWith({
      workspaceId: WORKSPACE_ID,
      product: "compute",
      stripeSubscriptionId: "sub_1",
    });
  });

  it("mirrors the checkout card onto a customer with no default payment method", async () => {
    h.findFirst.mockResolvedValue({
      id: WORKSPACE_ID,
      orgId: "org_1",
      billing: { plan: null },
      billingSubscriptions: [],
    });
    const stripe = stubStripe({
      sub: subscription({ default_payment_method: "pm_1" } as Partial<Stripe.Subscription>),
    });
    const result = await linkDeploySubscription(stripe, {
      sessionId: "cs_1",
      expectedWorkspaceId: WORKSPACE_ID,
      audit: AUDIT,
    });
    expect(result).toEqual({ ok: true, plan: "starter", alreadyLinked: false });
    expect(customersUpdate).toHaveBeenCalledWith("cus_1", {
      invoice_settings: { default_payment_method: "pm_1" },
    });
  });

  it("leaves an existing customer default payment method untouched", async () => {
    h.findFirst.mockResolvedValue({
      id: WORKSPACE_ID,
      orgId: "org_1",
      billing: { plan: null },
      billingSubscriptions: [],
    });
    const stripe = stubStripe({
      sub: subscription({ default_payment_method: "pm_1" } as Partial<Stripe.Subscription>),
      customer: {
        deleted: false,
        invoice_settings: { default_payment_method: "pm_existing" },
        default_source: null,
      },
    });
    const result = await linkDeploySubscription(stripe, {
      sessionId: "cs_1",
      expectedWorkspaceId: WORKSPACE_ID,
      audit: AUDIT,
    });
    expect(result).toEqual({ ok: true, plan: "starter", alreadyLinked: false });
    expect(customersUpdate).not.toHaveBeenCalled();
  });

  it("repoints when the recorded subscription no longer exists on Stripe", async () => {
    h.findFirst.mockResolvedValue({
      id: WORKSPACE_ID,
      orgId: "org_1",
      billing: { plan: null },
      billingSubscriptions: [{ product: "compute", stripeSubscriptionId: "sub_gone" }],
    });
    const stripe = stubStripe({ subsById: { sub_gone: RESOURCE_MISSING } });
    const result = await linkDeploySubscription(stripe, {
      sessionId: "cs_1",
      expectedWorkspaceId: WORKSPACE_ID,
      audit: AUDIT,
    });
    expect(result).toEqual({ ok: true, plan: "starter", alreadyLinked: false });
    expect(h.transaction).toHaveBeenCalledOnce();
  });
});

describe("linkApiSubscription", () => {
  beforeEach(() => {
    h.findFirst.mockReset();
    h.transaction.mockClear();
    h.insertAuditLogs.mockClear();
    h.update.mockClear();
    h.set.mockClear();
    h.where.mockClear();
    h.insert.mockClear();
    h.values.mockClear();
    h.onDuplicateKeyUpdate.mockClear();
    customersUpdate.mockClear();
  });

  it("rejects an API checkout belonging to another workspace", async () => {
    const stripe = stubStripe({
      session: apiSession({ client_reference_id: "ws_other" }),
      sub: apiSubscription(),
    });
    const result = await linkApiSubscription(stripe, {
      sessionId: "cs_api",
      expectedWorkspaceId: WORKSPACE_ID,
      audit: AUDIT,
    });

    expect(result).toMatchObject({ ok: false, reason: "forbidden" });
    expect(h.transaction).not.toHaveBeenCalled();
  });

  it("rejects an unpaid API checkout without granting the tier", async () => {
    const stripe = stubStripe({
      session: apiSession({ status: "open", payment_status: "unpaid" }),
      sub: apiSubscription(),
    });
    const result = await linkApiSubscription(stripe, {
      sessionId: "cs_api",
      expectedWorkspaceId: WORKSPACE_ID,
      audit: AUDIT,
    });

    expect(result).toMatchObject({ ok: false, reason: "not_paid" });
    expect(h.transaction).not.toHaveBeenCalled();
  });

  it("rejects a cancelling API subscription so webhook redelivery cannot restore it", async () => {
    const stripe = stubStripe({
      session: apiSession(),
      sub: apiSubscription({ cancel_at_period_end: true }),
    });
    const result = await linkApiSubscription(stripe, {
      sessionId: "cs_api",
      expectedWorkspaceId: WORKSPACE_ID,
      audit: AUDIT,
    });

    expect(result).toMatchObject({ ok: false, reason: "not_active" });
    expect(h.transaction).not.toHaveBeenCalled();
  });

  it("rejects a subscription whose product is outside the API allow-list", async () => {
    const stripe = stubStripe({
      session: apiSession(),
      sub: apiSubscription({
        items: { data: [{ id: "si_bad", price: { id: "price_bad", product: "prod_bad" } }] },
      } as unknown as Partial<Stripe.Subscription>),
    });
    const result = await linkApiSubscription(stripe, {
      sessionId: "cs_api",
      expectedWorkspaceId: WORKSPACE_ID,
      audit: AUDIT,
    });

    expect(result).toMatchObject({ ok: false, reason: "invalid_api_product" });
    expect(h.transaction).not.toHaveBeenCalled();
  });

  it("atomically links a paid API subscription and grants its configured quotas", async () => {
    h.findFirst.mockResolvedValue({
      id: WORKSPACE_ID,
      orgId: "org_1",
      billing: { tier: "Free" },
      billingSubscriptions: [],
    });
    const stripe = stubStripe({ session: apiSession(), sub: apiSubscription() });
    const result = await linkApiSubscription(stripe, {
      sessionId: "cs_api",
      expectedWorkspaceId: WORKSPACE_ID,
      audit: AUDIT,
    });

    expect(result).toEqual({ ok: true, productName: "Pro", alreadyLinked: false });
    expect(h.transaction).toHaveBeenCalledOnce();
    expect(h.set).toHaveBeenCalledWith({ stripeCustomerId: "cus_1", tier: "Pro" });
    expect(h.values).toHaveBeenCalledWith({
      workspaceId: WORKSPACE_ID,
      product: "api",
      stripeSubscriptionId: "sub_1",
    });
    expect(h.values).toHaveBeenCalledWith({
      workspaceId: WORKSPACE_ID,
      requestsPerMonth: 1_000_000,
      logsRetentionDays: 30,
      auditLogsRetentionDays: 90,
      team: true,
    });
    expect(h.insertAuditLogs).toHaveBeenCalledOnce();
  });

  it("does not replace a different live API subscription", async () => {
    h.findFirst.mockResolvedValue({
      id: WORKSPACE_ID,
      orgId: "org_1",
      billing: { tier: "Pro" },
      billingSubscriptions: [{ product: "api", stripeSubscriptionId: "sub_existing" }],
    });
    const stripe = stubStripe({
      session: apiSession(),
      sub: apiSubscription(),
      subsById: { sub_existing: apiSubscription({ id: "sub_existing" }) },
    });
    const result = await linkApiSubscription(stripe, {
      sessionId: "cs_api",
      expectedWorkspaceId: WORKSPACE_ID,
      audit: AUDIT,
    });

    expect(result).toMatchObject({ ok: false, reason: "subscription_conflict" });
    expect(h.transaction).not.toHaveBeenCalled();
  });

  it("replaces a dead API subscription on resubscribe", async () => {
    h.findFirst.mockResolvedValue({
      id: WORKSPACE_ID,
      orgId: "org_1",
      billing: { tier: "Free" },
      billingSubscriptions: [{ product: "api", stripeSubscriptionId: "sub_dead" }],
    });
    const stripe = stubStripe({
      session: apiSession(),
      sub: apiSubscription(),
      subsById: { sub_dead: apiSubscription({ id: "sub_dead", status: "canceled" }) },
    });
    const result = await linkApiSubscription(stripe, {
      sessionId: "cs_api",
      expectedWorkspaceId: WORKSPACE_ID,
      audit: AUDIT,
    });

    expect(result).toEqual({ ok: true, productName: "Pro", alreadyLinked: false });
    expect(h.transaction).toHaveBeenCalledOnce();
  });

  it("is an idempotent no-op when the same API subscription is already linked", async () => {
    h.findFirst.mockResolvedValue({
      id: WORKSPACE_ID,
      orgId: "org_1",
      billing: { tier: "Pro" },
      billingSubscriptions: [{ product: "api", stripeSubscriptionId: "sub_1" }],
    });
    const stripe = stubStripe({ session: apiSession(), sub: apiSubscription() });
    const result = await linkApiSubscription(stripe, {
      sessionId: "cs_api",
      expectedWorkspaceId: WORKSPACE_ID,
      audit: AUDIT,
    });

    expect(result).toEqual({ ok: true, productName: "Pro", alreadyLinked: true });
    expect(h.transaction).not.toHaveBeenCalled();
  });
});
