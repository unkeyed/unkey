// @vitest-environment node

import { beforeEach, describe, expect, it, vi } from "vitest";

const mocks = vi.hoisted(() => ({
  constructEvent: vi.fn(),
  findSubscription: vi.fn(),
  retrieveCustomer: vi.fn(),
  retrievePrice: vi.fn(),
  retrieveProduct: vi.fn(),
  retrieveSubscription: vi.fn(),
  linkApiSubscription: vi.fn(),
  linkDeploySubscription: vi.fn(),
  alertCustomerLifecycle: vi.fn(),
}));

vi.mock("@/lib/db", () => ({
  db: {
    query: {
      billingSubscriptions: { findFirst: mocks.findSubscription },
    },
  },
  eq: vi.fn(),
  schema: {},
}));

vi.mock("@/lib/auth/deactivateNonCreatorMemberships", () => ({
  deactivateNonCreatorMemberships: vi.fn(),
}));

vi.mock("@/lib/stripe/linkApiSubscription", () => ({
  linkApiSubscription: mocks.linkApiSubscription,
}));

vi.mock("@/lib/stripe/linkDeploySubscription", () => ({
  linkDeploySubscription: mocks.linkDeploySubscription,
}));

vi.mock("@/lib/env", () => ({
  stripeEnv: vi.fn(() => ({
    STRIPE_SECRET_KEY: "sk_test",
    STRIPE_WEBHOOK_SECRET: "whsec_test",
  })),
}));

vi.mock("@/lib/utils/slackAlerts", () => ({
  alertCustomerLifecycle: mocks.alertCustomerLifecycle,
  alertInvalidProductQuotaMetadata: vi.fn(),
  alertOrphanedDeploySubscription: vi.fn(),
  alertPaymentFailed: vi.fn(),
  alertPaymentRecovered: vi.fn(),
}));

vi.mock("stripe", () => ({
  default: class Stripe {
    public readonly webhooks = { constructEvent: mocks.constructEvent };
    public readonly customers = { retrieve: mocks.retrieveCustomer };
    public readonly prices = { retrieve: mocks.retrievePrice };
    public readonly products = { retrieve: mocks.retrieveProduct };
    public readonly subscriptions = { retrieve: mocks.retrieveSubscription };
  },
}));

import { POST } from "./route";

function webhookRequest(): Request {
  return new Request("https://app.unkey.com/api/webhooks/stripe", {
    method: "POST",
    headers: { "stripe-signature": "signed" },
    body: "signed payload",
  });
}

describe("POST /api/webhooks/stripe", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mocks.findSubscription.mockResolvedValue(undefined);
    mocks.retrievePrice.mockResolvedValue({
      id: "price_api",
      unit_amount: 2500,
      product: "prod_api",
    });
    mocks.retrieveCustomer.mockResolvedValue({
      id: "cus_1",
      deleted: false,
      email: "owner@acme.test",
      name: "Acme Owner",
      livemode: true,
    });
    mocks.retrieveProduct.mockResolvedValue({ id: "prod_api", name: "Pro" });
  });

  it("does not announce an incomplete Checkout subscription from its created event", async () => {
    mocks.constructEvent.mockReturnValue({
      id: "evt_1",
      type: "customer.subscription.created",
      data: {
        object: {
          id: "sub_1",
          status: "incomplete",
          customer: "cus_1",
          metadata: { unkey_product: "api", workspace_id: "ws_1" },
          items: {
            data: [
              {
                id: "si_1",
                price: { id: "price_api", metadata: { pricing_kind: "api" } },
              },
            ],
          },
        },
      },
    });

    const response = await POST(webhookRequest());

    expect(response.status).toBe(200);
    expect(mocks.retrieveSubscription).not.toHaveBeenCalled();
    expect(mocks.findSubscription).not.toHaveBeenCalled();
    expect(mocks.alertCustomerLifecycle).not.toHaveBeenCalled();
  });

  it("announces an API signup after Checkout links the active subscription", async () => {
    mocks.constructEvent.mockReturnValue({
      id: "evt_2",
      type: "checkout.session.completed",
      data: {
        object: {
          id: "cs_1",
          mode: "subscription",
          subscription: "sub_1",
          client_reference_id: "ws_1",
          metadata: { unkey_product: "api" },
        },
      },
    });
    mocks.linkApiSubscription.mockResolvedValue({
      ok: true,
      productName: "Pro",
      alreadyLinked: false,
    });
    mocks.findSubscription.mockResolvedValue({
      product: "api",
      workspace: {
        id: "ws_1",
        name: "Acme",
        orgId: "org_1",
        deletedAtM: null,
        billing: {},
      },
    });
    mocks.retrieveSubscription.mockResolvedValue({
      id: "sub_1",
      status: "active",
      cancel_at: null,
      cancel_at_period_end: false,
      customer: "cus_1",
      items: {
        data: [{ id: "si_1", price: { id: "price_api" } }],
      },
    });

    const response = await POST(webhookRequest());

    expect(response.status).toBe(200);
    expect(mocks.linkApiSubscription).toHaveBeenCalledWith(
      expect.anything(),
      expect.objectContaining({ sessionId: "cs_1", expectedWorkspaceId: "ws_1" }),
    );
    expect(mocks.alertCustomerLifecycle).toHaveBeenCalledWith({
      action: "signup",
      name: "Acme Owner",
      email: "owner@acme.test",
      workspaceId: "ws_1",
      workspaceName: "Acme",
      product: "Pro",
      price: "$25.00",
      stripeCustomerId: "cus_1",
      livemode: true,
    });
  });

  it("does not announce a Checkout whose linker rejects the current state", async () => {
    mocks.constructEvent.mockReturnValue({
      id: "evt_3",
      type: "checkout.session.completed",
      data: {
        object: {
          id: "cs_1",
          mode: "subscription",
          subscription: "sub_1",
          client_reference_id: "ws_1",
          metadata: { unkey_product: "api" },
        },
      },
    });
    mocks.linkApiSubscription.mockResolvedValue({
      ok: false,
      reason: "not_paid",
      message: "Checkout session is not paid.",
    });

    const response = await POST(webhookRequest());

    expect(response.status).toBe(200);
    expect(mocks.findSubscription).not.toHaveBeenCalled();
    expect(mocks.alertCustomerLifecycle).not.toHaveBeenCalled();
  });

  it("announces Compute after Checkout links it, including when the success page linked first", async () => {
    mocks.constructEvent.mockReturnValue({
      id: "evt_compute_checkout",
      type: "checkout.session.completed",
      data: {
        object: {
          id: "cs_compute",
          mode: "subscription",
          subscription: "sub_compute",
          client_reference_id: "ws_1",
          metadata: { unkey_product: "compute" },
        },
      },
    });
    mocks.linkDeploySubscription.mockResolvedValue({
      ok: true,
      plan: "pro",
      alreadyLinked: true,
    });
    mocks.findSubscription.mockResolvedValue({
      product: "compute",
      workspace: {
        id: "ws_1",
        name: "Acme",
        orgId: "org_1",
        deletedAtM: null,
        billing: { plan: "pro", tier: "Free" },
      },
    });
    mocks.retrieveSubscription.mockResolvedValue({
      id: "sub_compute",
      status: "active",
      cancel_at: null,
      cancel_at_period_end: false,
      customer: "cus_1",
      items: {
        data: [
          {
            id: "si_pro",
            price: { id: "price_pro", unit_amount: 4900, metadata: { plan: "pro" } },
          },
        ],
      },
    });

    const response = await POST(webhookRequest());

    expect(response.status).toBe(200);
    expect(mocks.linkDeploySubscription).toHaveBeenCalledOnce();
    expect(mocks.alertCustomerLifecycle).toHaveBeenCalledWith(
      expect.objectContaining({
        action: "signup",
        workspaceId: "ws_1",
        product: "Compute Pro",
        price: "$49.00",
      }),
    );
  });

  it("uses current Stripe state for a delayed direct Compute created event", async () => {
    mocks.constructEvent.mockReturnValue({
      id: "evt_4",
      type: "customer.subscription.created",
      data: {
        object: {
          id: "sub_compute",
          status: "active",
          metadata: {},
          items: {
            data: [
              {
                id: "si_starter",
                price: { id: "price_starter", metadata: { plan: "starter" } },
              },
            ],
          },
        },
      },
    });
    mocks.retrieveSubscription.mockResolvedValue({
      id: "sub_compute",
      status: "active",
      cancel_at: null,
      cancel_at_period_end: false,
      customer: "cus_1",
      metadata: {},
      items: {
        data: [
          {
            id: "si_pro",
            price: { id: "price_pro", unit_amount: 4900, metadata: { plan: "pro" } },
          },
        ],
      },
    });
    mocks.findSubscription.mockResolvedValue({
      product: "compute",
      workspace: {
        id: "ws_1",
        name: "Acme",
        orgId: "org_1",
        deletedAtM: null,
        billing: { plan: "pro", tier: "Free" },
      },
    });

    const response = await POST(webhookRequest());

    expect(response.status).toBe(200);
    expect(mocks.retrieveSubscription).toHaveBeenCalledWith("sub_compute");
    expect(mocks.alertCustomerLifecycle).toHaveBeenCalledWith(
      expect.objectContaining({
        action: "signup",
        product: "Compute Pro",
        price: "$49.00",
      }),
    );
  });
});
