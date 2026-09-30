import { beforeEach, describe, expect, it, vi } from "vitest";

vi.mock("@/lib/env", () => ({ stripeEnv: vi.fn() }));
vi.mock("@/lib/stripe", () => ({ getStripeClient: vi.fn() }));

import { stripeEnv } from "@/lib/env";
import { getStripeClient } from "@/lib/stripe";
import {
  type DeployBillingConfig,
  deployBillingConfig,
  deployCheckoutLineItems,
  deployResumeSubscriptionParams,
  deploySubscriptionItems,
  findDeployItems,
  findPlanFeeItem,
  planForPlanFeePriceId,
} from "./deployBilling";

const mockedStripeEnv = vi.mocked(stripeEnv);
const mockedGetStripeClient = vi.mocked(getStripeClient);

type StripeEnv = NonNullable<ReturnType<typeof stripeEnv>>;

// lookup_key -> resolved active price id, the mapping Stripe would return.
const ID: Record<string, string> = {
  lk_starter: "price_starter",
  lk_starter_concurrent: "price_starter_concurrent",
  lk_pro: "price_pro",
  lk_business: "price_business",
  lk_cpu: "price_cpu",
  lk_memory: "price_memory",
  lk_egress: "price_egress",
  lk_disk: "price_disk",
  lk_active_keys: "price_active_keys",
};

function envWith(overrides: Partial<StripeEnv> = {}): StripeEnv {
  return {
    STRIPE_SECRET_KEY: "sk_test_x",
    STRIPE_PRODUCT_IDS_PRO: ["prod_pro"],
    STRIPE_PRODUCT_IDS_ENTERPRISE: ["prod_ent"],
    STRIPE_WEBHOOK_SECRET: "whsec_x",
    STRIPE_LOOKUP_DEPLOY_STARTER: "lk_starter",
    STRIPE_LOOKUP_DEPLOY_PRO: "lk_pro",
    STRIPE_LOOKUP_DEPLOY_BUSINESS: "lk_business",
    STRIPE_LOOKUP_DEPLOY_METER_CPU: "lk_cpu",
    STRIPE_LOOKUP_DEPLOY_METER_MEMORY: "lk_memory",
    STRIPE_LOOKUP_DEPLOY_METER_EGRESS: "lk_egress",
    STRIPE_LOOKUP_DEPLOY_METER_DISK: "lk_disk",
    STRIPE_LOOKUP_DEPLOY_METER_ACTIVE_KEYS: "lk_active_keys",
    ...overrides,
  };
}

// Stripe whose prices.list returns the active price for each known lookup_key.
function stubStripe() {
  const list = vi.fn(async ({ lookup_keys }: { lookup_keys: string[] }) => ({
    data: lookup_keys.filter((k) => ID[k]).map((k) => ({ id: ID[k], lookup_key: k })),
  }));
  mockedGetStripeClient.mockReturnValue({
    prices: { list },
    // Test double; only prices.list is exercised here.
  } as unknown as ReturnType<typeof getStripeClient>);
  return list;
}

const CONFIG: DeployBillingConfig = {
  planFeePriceIds: { starter: "price_starter", pro: "price_pro", business: "price_business" },
  meteredPriceIds: ["price_cpu", "price_memory", "price_egress", "price_disk", "price_active_keys"],
  allDeployPriceIds: new Set([
    "price_starter",
    "price_pro",
    "price_business",
    "price_cpu",
    "price_memory",
    "price_egress",
    "price_disk",
    "price_active_keys",
  ]),
};

function item(id: string, priceId: string) {
  return { id, price: { id: priceId } };
}

describe("deployBillingConfig", () => {
  let listPrices: ReturnType<typeof stubStripe>;

  beforeEach(() => {
    mockedStripeEnv.mockReset();
    mockedGetStripeClient.mockReset();
    listPrices = stubStripe();
  });

  it("resolves lookup_keys to active price ids when fully configured", async () => {
    mockedStripeEnv.mockReturnValue(envWith());
    const c = await deployBillingConfig();
    expect(c?.planFeePriceIds).toEqual({
      starter: "price_starter",
      pro: "price_pro",
      business: "price_business",
    });
    expect(c?.meteredPriceIds).toEqual([
      "price_cpu",
      "price_memory",
      "price_egress",
      "price_disk",
      "price_active_keys",
    ]);
    expect(c?.allDeployPriceIds.size).toBe(8);
  });

  it("returns null when Stripe is not configured", async () => {
    mockedStripeEnv.mockReturnValue(null);
    expect(await deployBillingConfig()).toBeNull();
  });

  it("returns null when a plan-fee lookup_key is missing (all-or-nothing)", async () => {
    mockedStripeEnv.mockReturnValue(envWith({ STRIPE_LOOKUP_DEPLOY_BUSINESS: undefined }));
    expect(await deployBillingConfig()).toBeNull();
  });

  it("returns null when a metered lookup_key is missing (all-or-nothing)", async () => {
    mockedStripeEnv.mockReturnValue(envWith({ STRIPE_LOOKUP_DEPLOY_METER_DISK: undefined }));
    expect(await deployBillingConfig()).toBeNull();
  });

  it("returns null when a lookup_key resolves to no active price", async () => {
    // A distinct key set, so the earlier success is not served from cache; the
    // stub knows every key except this archived one.
    mockedStripeEnv.mockReturnValue(
      envWith({ STRIPE_LOOKUP_DEPLOY_BUSINESS: "lk_business_archived" }),
    );
    expect(await deployBillingConfig()).toBeNull();
  });

  it("coalesces concurrent Stripe catalog resolutions", async () => {
    // Use a distinct key so this test cannot hit the module's successful cache.
    mockedStripeEnv.mockReturnValue(
      envWith({ STRIPE_LOOKUP_DEPLOY_STARTER: "lk_starter_concurrent" }),
    );

    const [first, second, third] = await Promise.all([
      deployBillingConfig(),
      deployBillingConfig(),
      deployBillingConfig(),
    ]);

    expect(listPrices).toHaveBeenCalledTimes(1);
    expect(first).toBe(second);
    expect(second).toBe(third);
  });
});

describe("deploySubscriptionItems", () => {
  it("builds the plan-fee for the tier plus the shared metered prices", () => {
    expect(deploySubscriptionItems(CONFIG, "pro")).toEqual([
      { price: "price_pro" },
      { price: "price_cpu" },
      { price: "price_memory" },
      { price: "price_egress" },
      { price: "price_disk" },
      { price: "price_active_keys" },
    ]);
  });
});

describe("deployResumeSubscriptionParams", () => {
  it("atomically resumes a cancelled Starter subscription on Pro", () => {
    expect(
      deployResumeSubscriptionParams(CONFIG, { id: "si_plan", plan: "starter" }, "pro"),
    ).toEqual({
      cancel_at_period_end: false,
      items: [{ id: "si_plan", price: "price_pro" }],
      proration_behavior: "always_invoice",
      payment_behavior: "error_if_incomplete",
    });
  });

  it("resumes the same tier without invoicing it again", () => {
    expect(
      deployResumeSubscriptionParams(CONFIG, { id: "si_plan", plan: "starter" }, "starter"),
    ).toEqual({
      cancel_at_period_end: false,
      proration_behavior: "none",
      payment_behavior: "error_if_incomplete",
    });
  });

  it("changes to a lower tier without refunding the paid period", () => {
    expect(
      deployResumeSubscriptionParams(CONFIG, { id: "si_plan", plan: "pro" }, "starter"),
    ).toEqual({
      cancel_at_period_end: false,
      items: [{ id: "si_plan", price: "price_starter" }],
      proration_behavior: "none",
      payment_behavior: "error_if_incomplete",
    });
  });
});

describe("deployCheckoutLineItems", () => {
  it("gives the plan-fee quantity 1 and omits quantity on metered items", () => {
    const items = deployCheckoutLineItems(CONFIG, "pro");
    expect(items[0]).toEqual({ price: "price_pro", quantity: 1 });
    // Metered prices must not carry a quantity — Stripe Checkout rejects it.
    for (const item of items.slice(1)) {
      expect(item).not.toHaveProperty("quantity");
    }
  });

  it("puts the plan-fee first, followed by exactly the metered prices", () => {
    const items = deployCheckoutLineItems(CONFIG, "starter");
    expect(items.map((i) => i.price)).toEqual(["price_starter", ...CONFIG.meteredPriceIds]);
  });

  it("changes only the plan-fee price id when the plan changes", () => {
    const starter = deployCheckoutLineItems(CONFIG, "starter");
    const business = deployCheckoutLineItems(CONFIG, "business");
    expect(starter[0].price).toBe("price_starter");
    expect(business[0].price).toBe("price_business");
    expect(starter.slice(1)).toEqual(business.slice(1));
  });
});

describe("planForPlanFeePriceId", () => {
  it("maps a plan-fee price id back to its plan", () => {
    expect(planForPlanFeePriceId(CONFIG, "price_business")).toBe("business");
  });
  it("returns undefined for a metered or unknown price id", () => {
    expect(planForPlanFeePriceId(CONFIG, "price_cpu")).toBeUndefined();
    expect(planForPlanFeePriceId(CONFIG, "price_unknown")).toBeUndefined();
  });
});

describe("findDeployItems", () => {
  it("returns every Deploy item (plan-fee + metered), ignoring API items", () => {
    const found = findDeployItems(CONFIG, [
      item("si_api", "price_api"),
      item("si_fee", "price_pro"),
      item("si_cpu", "price_cpu"),
      item("si_disk", "price_disk"),
    ]);
    expect(found).toEqual([
      { id: "si_fee", priceId: "price_pro" },
      { id: "si_cpu", priceId: "price_cpu" },
      { id: "si_disk", priceId: "price_disk" },
    ]);
  });

  it("returns empty when no Deploy items are present", () => {
    expect(findDeployItems(CONFIG, [item("si_api", "price_api")])).toEqual([]);
  });
});

describe("findPlanFeeItem", () => {
  it("finds the plan-fee item and its plan among other items", () => {
    const found = findPlanFeeItem(CONFIG, [
      item("si_cpu", "price_cpu"),
      item("si_fee", "price_business"),
    ]);
    expect(found).toEqual({ id: "si_fee", plan: "business" });
  });

  it("returns undefined when there is no plan-fee item (metered only)", () => {
    expect(findPlanFeeItem(CONFIG, [item("si_cpu", "price_cpu")])).toBeUndefined();
  });
});
