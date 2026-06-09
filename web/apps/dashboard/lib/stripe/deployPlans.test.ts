import { beforeEach, describe, expect, it, vi } from "vitest";

vi.mock("@/lib/env", () => ({
  stripeEnv: vi.fn(),
}));

import { stripeEnv } from "@/lib/env";
import {
  type DeployBillingConfig,
  deployBillingConfig,
  deploySubscriptionItems,
  findDeployItems,
  findPlanFeeItem,
  planForPlanFeePriceId,
} from "./deployPlans";

const mockedStripeEnv = vi.mocked(stripeEnv);

type StripeEnv = NonNullable<ReturnType<typeof stripeEnv>>;

const PRICE = {
  starter: "price_starter",
  pro: "price_pro",
  business: "price_business",
  cpu: "price_cpu",
  memory: "price_memory",
  egress: "price_egress",
  disk: "price_disk",
};

function envWith(overrides: Partial<StripeEnv> = {}): StripeEnv {
  return {
    STRIPE_SECRET_KEY: "sk_test_x",
    STRIPE_PRODUCT_IDS_PRO: ["prod_pro"],
    STRIPE_PRODUCT_IDS_ENTERPRISE: ["prod_ent"],
    STRIPE_WEBHOOK_SECRET: "whsec_x",
    STRIPE_PRICE_DEPLOY_STARTER: PRICE.starter,
    STRIPE_PRICE_DEPLOY_PRO: PRICE.pro,
    STRIPE_PRICE_DEPLOY_BUSINESS: PRICE.business,
    STRIPE_PRICE_DEPLOY_METER_CPU: PRICE.cpu,
    STRIPE_PRICE_DEPLOY_METER_MEMORY: PRICE.memory,
    STRIPE_PRICE_DEPLOY_METER_EGRESS: PRICE.egress,
    STRIPE_PRICE_DEPLOY_METER_DISK: PRICE.disk,
    ...overrides,
  };
}

// deployBillingConfig with the fully-configured env, for the pure helpers.
function config(): DeployBillingConfig {
  mockedStripeEnv.mockReturnValue(envWith());
  const c = deployBillingConfig();
  if (!c) {
    throw new Error("expected config");
  }
  return c;
}

function item(id: string, priceId: string) {
  return { id, price: { id: priceId } };
}

describe("deployBillingConfig", () => {
  beforeEach(() => mockedStripeEnv.mockReset());

  it("resolves all plan-fee and metered price ids when fully configured", () => {
    mockedStripeEnv.mockReturnValue(envWith());
    const c = deployBillingConfig();
    expect(c?.planFeePriceIds).toEqual({
      starter: PRICE.starter,
      pro: PRICE.pro,
      business: PRICE.business,
    });
    expect(c?.meteredPriceIds).toEqual([PRICE.cpu, PRICE.memory, PRICE.egress, PRICE.disk]);
    expect(c?.allDeployPriceIds.size).toBe(7);
  });

  it("returns null when Stripe is not configured", () => {
    mockedStripeEnv.mockReturnValue(null);
    expect(deployBillingConfig()).toBeNull();
  });

  it("returns null when a plan-fee price id is missing (all-or-nothing)", () => {
    mockedStripeEnv.mockReturnValue(envWith({ STRIPE_PRICE_DEPLOY_BUSINESS: undefined }));
    expect(deployBillingConfig()).toBeNull();
  });

  it("returns null when a metered price id is missing (all-or-nothing)", () => {
    mockedStripeEnv.mockReturnValue(envWith({ STRIPE_PRICE_DEPLOY_METER_DISK: undefined }));
    expect(deployBillingConfig()).toBeNull();
  });
});

describe("deploySubscriptionItems", () => {
  it("builds the plan-fee for the tier plus the shared metered prices", () => {
    expect(deploySubscriptionItems(config(), "pro")).toEqual([
      { price: PRICE.pro },
      { price: PRICE.cpu },
      { price: PRICE.memory },
      { price: PRICE.egress },
      { price: PRICE.disk },
    ]);
  });
});

describe("planForPlanFeePriceId", () => {
  it("maps a plan-fee price id back to its plan", () => {
    expect(planForPlanFeePriceId(config(), PRICE.business)).toBe("business");
  });
  it("returns undefined for a metered or unknown price id", () => {
    expect(planForPlanFeePriceId(config(), PRICE.cpu)).toBeUndefined();
    expect(planForPlanFeePriceId(config(), "price_unknown")).toBeUndefined();
  });
});

describe("findDeployItems", () => {
  it("returns every Deploy item (plan-fee + metered), ignoring API items", () => {
    const c = config();
    const found = findDeployItems(c, [
      item("si_api", "price_api"),
      item("si_fee", PRICE.pro),
      item("si_cpu", PRICE.cpu),
      item("si_disk", PRICE.disk),
    ]);
    expect(found).toEqual([
      { id: "si_fee", priceId: PRICE.pro },
      { id: "si_cpu", priceId: PRICE.cpu },
      { id: "si_disk", priceId: PRICE.disk },
    ]);
  });

  it("returns empty when no Deploy items are present", () => {
    expect(findDeployItems(config(), [item("si_api", "price_api")])).toEqual([]);
  });
});

describe("findPlanFeeItem", () => {
  it("finds the plan-fee item and its plan among other items", () => {
    const found = findPlanFeeItem(config(), [
      item("si_cpu", PRICE.cpu),
      item("si_fee", PRICE.business),
    ]);
    expect(found).toEqual({ id: "si_fee", plan: "business" });
  });

  it("returns undefined when there is no plan-fee item (metered only)", () => {
    expect(findPlanFeeItem(config(), [item("si_cpu", PRICE.cpu)])).toBeUndefined();
  });
});
