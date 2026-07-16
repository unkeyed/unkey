import { beforeEach, describe, expect, it, vi } from "vitest";

// Chainable db mock: transaction(cb) runs cb with a tx whose
// update().set().where() resolves. findFirst returns the workspace row.
const h = vi.hoisted(() => {
  const where = vi.fn().mockResolvedValue(undefined);
  const set = vi.fn().mockReturnValue({ where });
  const update = vi.fn().mockReturnValue({ set });
  const findFirst = vi.fn();
  const transaction = vi.fn(async (cb: (tx: unknown) => unknown) => cb({ update }));
  const insertAuditLogs = vi.fn();
  return { where, set, update, findFirst, transaction, insertAuditLogs };
});

vi.mock("@/lib/db", () => ({
  db: { query: { workspaces: { findFirst: h.findFirst } }, transaction: h.transaction },
  eq: vi.fn(),
  schema: { workspaces: { id: {} } },
}));
vi.mock("@/lib/audit", () => ({ insertAuditLogs: h.insertAuditLogs }));

import Stripe from "stripe";
import { linkDeploySubscription } from "./linkDeploySubscription";

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

  it("writes customer + subscription + plan for a paid, active, unlinked workspace", async () => {
    h.findFirst.mockResolvedValue({
      id: WORKSPACE_ID,
      orgId: "org_1",
      stripeSubscriptionId: null,
      deployPlan: null,
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
      stripeSubscriptionId: "sub_1",
      deployPlan: "starter",
    });
    expect(h.insertAuditLogs).toHaveBeenCalledOnce();
  });

  it("is an idempotent no-op when the same subscription+plan is already linked", async () => {
    h.findFirst.mockResolvedValue({
      id: WORKSPACE_ID,
      orgId: "org_1",
      stripeSubscriptionId: "sub_1",
      deployPlan: "starter",
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
      stripeSubscriptionId: "sub_other",
      deployPlan: "pro",
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
      stripeSubscriptionId: "sub_dead",
      deployPlan: null,
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
      stripeSubscriptionId: "sub_1",
      deployPlan: "starter",
    });
  });

  it("mirrors the checkout card onto a customer with no default payment method", async () => {
    h.findFirst.mockResolvedValue({
      id: WORKSPACE_ID,
      orgId: "org_1",
      stripeSubscriptionId: null,
      deployPlan: null,
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
      stripeSubscriptionId: null,
      deployPlan: null,
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
      stripeSubscriptionId: "sub_gone",
      deployPlan: null,
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
