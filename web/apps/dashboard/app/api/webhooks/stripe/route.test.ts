import type Stripe from "stripe";
import { beforeEach, describe, expect, it, vi } from "vitest";

const mocks = vi.hoisted(() => ({
  alertCancellationReason: vi.fn(),
  alertCustomerLifecycle: vi.fn(),
  billingSubscriptionFindFirst: vi.fn(),
  constructEvent: vi.fn(),
  createCtrlClient: vi.fn(),
  customersRetrieve: vi.fn(),
  deployBillingConfig: vi.fn(),
  detectDeployPlan: vi.fn(),
  handlePaymentMethodAttached: vi.fn(),
  pricesRetrieve: vi.fn(),
  productsRetrieve: vi.fn(),
  subscriptionsRetrieve: vi.fn(),
  transaction: vi.fn(),
}));

vi.mock("@/gen/proto/ctrl/v1/deployment_pb", () => ({ DeployService: {} }));
vi.mock("@/lib/audit", () => ({ insertAuditLogs: vi.fn() }));
vi.mock("@/lib/auth/deactivateNonCreatorMemberships", () => ({
  deactivateNonCreatorMemberships: vi.fn(),
}));
vi.mock("@/lib/ctrl-client", () => ({ createCtrlClient: mocks.createCtrlClient }));
vi.mock("@/lib/db", () => ({
  db: {
    query: {
      billingSubscriptions: { findFirst: mocks.billingSubscriptionFindFirst },
    },
    transaction: mocks.transaction,
  },
  eq: vi.fn(),
  schema: {},
}));
vi.mock("@/lib/env", () => ({
  stripeEnv: () => ({
    STRIPE_SECRET_KEY: "sk_test_mock",
    STRIPE_WEBHOOK_SECRET: "whsec_test_mock",
  }),
}));
vi.mock("@/lib/fmt", () => ({
  formatPrice: (amount: number) => `$${(amount / 100).toFixed(2)}`,
}));
vi.mock("@/lib/stripe/billingSubscriptions", () => ({
  deleteBillingSubscription: vi.fn(),
}));
vi.mock("@/lib/stripe/computeAlerts", async (importOriginal) => ({
  ...(await importOriginal<typeof import("@/lib/stripe/computeAlerts")>()),
  computeCreatedAlert: vi.fn(),
  computeUpdatedAlert: vi.fn(),
}));
vi.mock("@/lib/stripe/deployBilling", () => ({
  deployBillingConfig: mocks.deployBillingConfig,
  findApiItem: (_config: unknown, items: unknown[]) => items[0],
}));
vi.mock("@/lib/stripe/computeSignupCredit", () => ({
  handlePaymentMethodAttached: mocks.handlePaymentMethodAttached,
}));
vi.mock("@/lib/stripe/deployCredits", () => ({
  grantDeployCreditsForInvoice: vi.fn(),
}));
vi.mock("@/lib/stripe/deployPlan", () => ({
  deployPlanGrantsTeam: vi.fn(() => false),
  detectDeployPlan: mocks.detectDeployPlan,
  parseDeployPlan: vi.fn(() => null),
}));
vi.mock("@/lib/stripe/linkApiSubscription", () => ({ linkApiSubscription: vi.fn() }));
vi.mock("@/lib/stripe/linkDeploySubscription", () => ({ linkDeploySubscription: vi.fn() }));
vi.mock("@/lib/stripe/paymentUtils", () => ({
  isPaymentRecovery: vi.fn(),
  isPaymentRecoveryUpdate: vi.fn(),
}));
vi.mock("@/lib/stripe/productUtils", () => ({ validateAndParseQuotas: vi.fn() }));
vi.mock("@/lib/stripe/setWorkspaceLimits", () => ({ setWorkspaceLimits: vi.fn() }));
vi.mock("@/lib/stripe/subscriptionUtils", async (importOriginal) => ({
  ...(await importOriginal<typeof import("@/lib/stripe/subscriptionUtils")>()),
  isAutomatedBillingRenewal: vi.fn(),
  isCardUpdateOnly: vi.fn(),
  isPaymentFailureRelatedUpdate: vi.fn(),
  isScheduleUpdateOnly: vi.fn(),
}));
vi.mock("@/lib/utils/slackAlerts", () => ({
  alertCancellationReason: mocks.alertCancellationReason,
  alertCustomerLifecycle: mocks.alertCustomerLifecycle,
  alertInvalidProductQuotaMetadata: vi.fn(),
  alertOrphanedDeploySubscription: vi.fn(),
  alertPaymentFailed: vi.fn(),
  alertPaymentRecovered: vi.fn(),
}));
vi.mock("stripe", () => {
  class StripeMock {
    static errors = { StripeError: class extends Error {} };

    webhooks = { constructEvent: mocks.constructEvent };
    prices = { retrieve: mocks.pricesRetrieve };
    customers = { retrieve: mocks.customersRetrieve };
    products = { retrieve: mocks.productsRetrieve };
    subscriptions = { retrieve: mocks.subscriptionsRetrieve };
  }

  return { default: StripeMock };
});

import { POST } from "./route";

beforeEach(() => {
  vi.clearAllMocks();
  mocks.billingSubscriptionFindFirst.mockResolvedValue(null);
  mocks.deployBillingConfig.mockResolvedValue(null);
  mocks.detectDeployPlan.mockReturnValue(null);
  mocks.pricesRetrieve.mockResolvedValue({
    id: "price_api",
    product: "prod_api",
    unit_amount: 2500,
  });
  mocks.customersRetrieve.mockResolvedValue({
    id: "cus_123",
    deleted: false,
    email: "jane@example.com",
    name: "Jane",
    livemode: true,
  });
  mocks.productsRetrieve.mockResolvedValue({ id: "prod_api", name: "API Pro" });
  mocks.alertCustomerLifecycle.mockResolvedValue("sent");
});

function signedRequest(body = "{}"): Request {
  return new Request("https://app.unkey.com/api/webhooks/stripe", {
    method: "POST",
    headers: { "stripe-signature": "signed" },
    body,
  });
}

describe("payment_method.attached", () => {
  it("rejects a missing signature before granting a signup credit", async () => {
    const response = await POST(
      new Request("https://app.unkey.com/api/webhooks/stripe", {
        method: "POST",
        body: "{}",
      }),
    );

    expect(response.status).toBe(400);
    expect(mocks.constructEvent).not.toHaveBeenCalled();
    expect(mocks.handlePaymentMethodAttached).not.toHaveBeenCalled();
  });

  it("rejects an invalid signature before granting a signup credit", async () => {
    mocks.constructEvent.mockImplementation(() => {
      throw new Error("bad signature");
    });

    const response = await POST(signedRequest());

    expect(response.status).toBe(400);
    expect(mocks.handlePaymentMethodAttached).not.toHaveBeenCalled();
  });

  it("grants only after the webhook signature verifies", async () => {
    mocks.constructEvent.mockReturnValue({
      id: "evt_pm",
      type: "payment_method.attached",
      data: { object: { id: "pm_card", object: "payment_method" } },
    });
    mocks.handlePaymentMethodAttached.mockResolvedValue({
      granted: true,
      transactionId: "cbtxn_signup",
      amountCents: 500,
    });

    const response = await POST(signedRequest());

    expect(mocks.handlePaymentMethodAttached).toHaveBeenCalledOnce();
    expect(response.headers.get("X-Unkey-Webhook-Result")).toBe("signup_credit_granted");
    await expect(response.json()).resolves.toMatchObject({
      eventId: "evt_pm",
      result: "signup_credit_granted",
    });
  });
});

describe("customer.subscription.created", () => {
  it("posts an API signup alert when the event arrives before the checkout link", async () => {
    mocks.constructEvent.mockReturnValue({
      id: "evt_created",
      type: "customer.subscription.created",
      data: {
        object: {
          id: "sub_api",
          customer: "cus_123",
          metadata: { unkey_product: "api", workspace_id: "ws_123" },
          items: { data: [{ id: "si_api", price: { id: "price_api" } }] },
        },
      },
    });

    const response = await POST(
      new Request("https://app.unkey.com/api/webhooks/stripe", {
        method: "POST",
        headers: { "stripe-signature": "signed" },
        body: "{}",
      }),
    );

    expect(mocks.billingSubscriptionFindFirst).toHaveBeenCalledOnce();
    expect(mocks.alertCustomerLifecycle).toHaveBeenCalledWith({
      action: "signup",
      name: "Jane",
      email: "jane@example.com",
      workspaceId: "ws_123",
      workspaceName: undefined,
      product: "API Pro",
      price: "$25.00",
      stripeCustomerId: "cus_123",
      livemode: true,
    });
    expect(response.headers.get("X-Unkey-Webhook-Result")).toBe("api_signup_alert_processed");
    await expect(response.json()).resolves.toMatchObject({
      eventId: "evt_created",
      result: "api_signup_alert_processed",
      details: { workspaceLinked: false, slackDelivery: "sent" },
    });
  });
});

describe("customer.subscription.updated on a cancelling subscription", () => {
  const cancellingSub = {
    id: "sub_api",
    customer: "cus_123",
    cancel_at: 1_900_000_000,
    cancel_at_period_end: true,
    items: { data: [{ id: "si_api", price: { id: "price_api", product: "prod_api" } }] },
  };
  const clearedDetails = { feedback: null, comment: null, reason: null };
  const recordedDetails = {
    feedback: "too_expensive",
    comment: "Too pricey for us",
    reason: "cancellation_requested",
  } as const;
  const withFeedback = { ...cancellingSub, cancellation_details: recordedDetails };

  function linkWorkspace(product: "api" | "compute") {
    mocks.billingSubscriptionFindFirst.mockResolvedValue({
      workspaceId: "ws_123",
      product,
      workspace: {
        id: "ws_123",
        orgId: "org_123",
        name: "Acme",
        deletedAtM: null,
        billing: { workspaceId: "ws_123", plan: "pro", planOverride: null, tier: "Pro" },
      },
    });
  }

  beforeEach(() => {
    linkWorkspace("api");
    mocks.subscriptionsRetrieve.mockResolvedValue(cancellingSub);
  });

  function updatedEvent(
    previousAttributes: Partial<Stripe.Subscription>,
    object: typeof cancellingSub &
      Partial<Pick<Stripe.Subscription, "cancellation_details">> = cancellingSub,
  ) {
    return {
      id: "evt_updated",
      type: "customer.subscription.updated",
      data: { object, previous_attributes: previousAttributes },
    };
  }

  const janeReason = {
    reason: "too_expensive",
    comment: "Too pricey for us",
    name: "Jane",
    email: "jane@example.com",
    workspaceId: "ws_123",
    workspaceName: "Acme",
    stripeCustomer: { id: "cus_123", livemode: true },
  };

  it("posts one reason alert for a feedback-only API update and skips the API reconcile", async () => {
    mocks.constructEvent.mockReturnValue(
      updatedEvent({ cancellation_details: clearedDetails }, withFeedback),
    );

    const response = await POST(signedRequest());

    expect(response.status).toBe(200);
    expect(mocks.alertCustomerLifecycle).not.toHaveBeenCalled();
    expect(mocks.alertCancellationReason).toHaveBeenCalledOnce();
    expect(mocks.alertCancellationReason).toHaveBeenCalledWith({
      ...janeReason,
      product: "API Pro",
    });
    expect(mocks.pricesRetrieve).not.toHaveBeenCalled();
    expect(mocks.productsRetrieve).toHaveBeenCalledOnce();
    expect(mocks.transaction).not.toHaveBeenCalled();
  });

  it("posts one reason alert for a feedback-only Compute update and skips teardown and mirror", async () => {
    linkWorkspace("compute");
    mocks.detectDeployPlan.mockReturnValue("pro");
    mocks.constructEvent.mockReturnValue(
      updatedEvent({ cancellation_details: clearedDetails }, withFeedback),
    );

    const response = await POST(signedRequest());

    expect(response.status).toBe(200);
    expect(mocks.alertCancellationReason).toHaveBeenCalledOnce();
    expect(mocks.alertCancellationReason).toHaveBeenCalledWith({
      ...janeReason,
      product: "Compute Pro",
    });
    expect(mocks.createCtrlClient).not.toHaveBeenCalled();
    expect(mocks.transaction).not.toHaveBeenCalled();
    expect(mocks.productsRetrieve).not.toHaveBeenCalled();
  });

  it("posts no alert when the feedback did not change", async () => {
    mocks.constructEvent.mockReturnValue(
      updatedEvent({ cancellation_details: recordedDetails }, withFeedback),
    );

    const response = await POST(signedRequest());

    expect(response.status).toBe(200);
    expect(mocks.alertCancellationReason).not.toHaveBeenCalled();
  });

  it("posts no alert when the feedback was cleared", async () => {
    mocks.constructEvent.mockReturnValue(updatedEvent({ cancellation_details: clearedDetails }));

    const response = await POST(signedRequest());

    expect(response.status).toBe(200);
    expect(mocks.alertCustomerLifecycle).not.toHaveBeenCalled();
    expect(mocks.alertCancellationReason).not.toHaveBeenCalled();
  });

  it("posts the reason with unknown contact details when the customer has no email", async () => {
    mocks.customersRetrieve.mockResolvedValue({
      id: "cus_123",
      deleted: false,
      email: null,
      name: null,
      livemode: false,
    });
    mocks.constructEvent.mockReturnValue(
      updatedEvent({ cancellation_details: clearedDetails }, withFeedback),
    );

    await POST(signedRequest());

    expect(mocks.alertCancellationReason).toHaveBeenCalledWith({
      ...janeReason,
      name: "Unknown",
      email: "Unknown",
      product: "API Pro",
      stripeCustomer: { id: "cus_123", livemode: false },
    });
  });

  it("posts the reason without a Stripe link when the customer lookup fails", async () => {
    mocks.customersRetrieve.mockRejectedValue(new Error("stripe is down"));
    mocks.constructEvent.mockReturnValue(
      updatedEvent({ cancellation_details: clearedDetails }, withFeedback),
    );

    await POST(signedRequest());

    expect(mocks.alertCancellationReason).toHaveBeenCalledWith({
      ...janeReason,
      name: "Unknown",
      email: "Unknown",
      product: "API Pro",
      stripeCustomer: null,
    });
  });

  it("posts the cancelling alert, then one reason alert, when a cancel carries feedback", async () => {
    mocks.constructEvent.mockReturnValue(
      updatedEvent(
        { cancel_at_period_end: false, cancellation_details: clearedDetails },
        withFeedback,
      ),
    );

    const response = await POST(signedRequest());

    expect(response.status).toBe(200);
    expect(mocks.alertCustomerLifecycle).toHaveBeenCalledOnce();
    expect(mocks.alertCancellationReason).toHaveBeenCalledOnce();
    expect(mocks.alertCancellationReason).toHaveBeenCalledWith({
      ...janeReason,
      product: "API Pro",
    });
    expect(mocks.alertCustomerLifecycle.mock.invocationCallOrder[0]).toBeLessThan(
      mocks.alertCancellationReason.mock.invocationCallOrder[0],
    );
  });

  it("posts a cancelling alert when the cancel was just scheduled", async () => {
    mocks.constructEvent.mockReturnValue(updatedEvent({ cancel_at_period_end: false }));

    const response = await POST(signedRequest());

    expect(response.status).toBe(200);
    expect(mocks.alertCancellationReason).not.toHaveBeenCalled();
    expect(mocks.alertCustomerLifecycle).toHaveBeenCalledWith({
      action: "cancelling",
      name: "Jane",
      email: "jane@example.com",
      workspaceId: "ws_123",
      workspaceName: "Acme",
      product: "API Pro",
      price: "$25.00",
      stripeCustomerId: "cus_123",
      livemode: true,
    });
  });
});
