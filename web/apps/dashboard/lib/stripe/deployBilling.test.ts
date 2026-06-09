import { beforeEach, describe, expect, it, vi } from "vitest";

vi.mock("@/lib/env", () => ({ stripeEnv: vi.fn() }));
vi.mock("@/lib/stripe", () => ({ getStripeClient: vi.fn() }));

import { stripeEnv } from "@/lib/env";
import { getStripeClient } from "@/lib/stripe";
import {
  type DeployBillingConfig,
  deployBillingConfig,
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
  lk_pro: "price_pro",
  lk_business: "price_business",
  lk_cpu: "price_cpu",
  lk_memory: "price_memory",
  lk_egress: "price_egress",
  lk_disk: "price_disk",
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
    ...overrides,
  };
}

// Stripe whose prices.list returns the active price for each known lookup_key.
function stubStripe() {
  mockedGetStripeClient.mockReturnValue({
    prices: {
      list: vi.fn(async ({ lookup_keys }: { lookup_keys: string[] }) => ({
        data: lookup_keys.filter((k) => ID[k]).map((k) => ({ id: ID[k], lookup_key: k })),
      })),
    },
    // Test double; only prices.list is exercised here.
  } as unknown as ReturnType<typeof getStripeClient>);
}

const CONFIG: DeployBillingConfig = {
  planFeePriceIds: { starter: "price_starter", pro: "price_pro", business: "price_business" },
  meteredPriceIds: ["price_cpu", "price_memory", "price_egress", "price_disk"],
  allDeployPriceIds: new Set([
    "price_starter",
    "price_pro",
    "price_business",
    "price_cpu",
    "price_memory",
    "price_egress",
    "price_disk",
  ]),
};

function item(id: string, priceId: string) {
  return { id, price: { id: priceId } };
}

describe("deployBillingConfig", () => {
  beforeEach(() => {
    mockedStripeEnv.mockReset();
    mockedGetStripeClient.mockReset();
    stubStripe();
  });

  it("resolves lookup_keys to active price ids when fully configured", async () => {
    mockedStripeEnv.mockReturnValue(envWith());
    const c = await deployBillingConfig();
    expect(c?.planFeePriceIds).toEqual({
      starter: "price_starter",
      pro: "price_pro",
      business: "price_business",
    });
    expect(c?.meteredPriceIds).toEqual(["price_cpu", "price_memory", "price_egress", "price_disk"]);
    expect(c?.allDeployPriceIds.size).toBe(7);
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
});

describe("deploySubscriptionItems", () => {
  it("builds the plan-fee for the tier plus the shared metered prices", () => {
    expect(deploySubscriptionItems(CONFIG, "pro")).toEqual([
      { price: "price_pro" },
      { price: "price_cpu" },
      { price: "price_memory" },
      { price: "price_egress" },
      { price: "price_disk" },
    ]);
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
