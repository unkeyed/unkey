import { beforeEach, describe, expect, it, vi } from "vitest";

const mocks = vi.hoisted(() => ({
  alertCustomerLifecycle: vi.fn(),
  billingSubscriptionFindFirst: vi.fn(),
  constructEvent: vi.fn(),
  customersRetrieve: vi.fn(),
  deployBillingConfig: vi.fn(),
  detectDeployPlan: vi.fn(),
  pricesRetrieve: vi.fn(),
  productsRetrieve: vi.fn(),
}));

vi.mock("@/gen/proto/ctrl/v1/deployment_pb", () => ({ DeployService: {} }));
vi.mock("@/lib/audit", () => ({ insertAuditLogs: vi.fn() }));
vi.mock("@/lib/auth/deactivateNonCreatorMemberships", () => ({
  deactivateNonCreatorMemberships: vi.fn(),
}));
vi.mock("@/lib/ctrl-client", () => ({ createCtrlClient: vi.fn() }));
vi.mock("@/lib/db", () => ({
  db: {
    query: {
      billingSubscriptions: { findFirst: mocks.billingSubscriptionFindFirst },
    },
    transaction: vi.fn(),
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
vi.mock("@/lib/stripe/computeAlerts", () => ({
  computeCreatedAlert: vi.fn(),
  computeUpdatedAlert: vi.fn(),
}));
vi.mock("@/lib/stripe/deployBilling", () => ({
  deployBillingConfig: mocks.deployBillingConfig,
  findApiItem: (_config: unknown, items: unknown[]) => items[0],
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
vi.mock("@/lib/stripe/subscriptionUtils", () => ({
  isAutomatedBillingRenewal: vi.fn(),
  isCardUpdateOnly: vi.fn(),
  isPaymentFailureRelatedUpdate: vi.fn(),
  isScheduleUpdateOnly: vi.fn(),
}));
vi.mock("@/lib/utils/slackAlerts", () => ({
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
