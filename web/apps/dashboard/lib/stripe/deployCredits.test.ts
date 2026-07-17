import type Stripe from "stripe";
import { afterEach, describe, expect, it, vi } from "vitest";
import type { DeployBillingConfig } from "./deployBilling";
import { grantDeployCreditsForInvoice, netDeployFee } from "./deployCredits";

vi.mock("./deployBilling", async (importOriginal) => {
  const actual = await importOriginal<typeof import("./deployBilling")>();
  return {
    ...actual,
    deployBillingConfig: vi.fn(),
  };
});

import { deployBillingConfig } from "./deployBilling";

const config: DeployBillingConfig = {
  planFeePriceIds: {
    starter: "price_fee_starter",
    pro: "price_fee_pro",
    business: "price_fee_business",
  },
  meteredPriceIds: ["price_cpu", "price_mem", "price_egress", "price_disk"],
  allDeployPriceIds: new Set([
    "price_fee_starter",
    "price_fee_pro",
    "price_fee_business",
    "price_cpu",
    "price_mem",
    "price_egress",
    "price_disk",
  ]),
};

// Minimal invoice line stub: netDeployFee reads amount, discount_amounts,
// period.end, and the price id under pricing.price_details.price.
function line(
  priceId: string,
  amount: number,
  periodEnd: number,
  discountCents = 0,
): Stripe.InvoiceLineItem {
  return {
    amount,
    discount_amounts: discountCents ? [{ amount: discountCents, discount: "di_test" }] : [],
    period: { end: periodEnd, start: periodEnd - 3600 },
    pricing: { type: "price_details", price_details: { price: priceId } },
  } as unknown as Stripe.InvoiceLineItem;
}

describe("netDeployFee", () => {
  it("returns the fee of a single plan-fee line (subscribe / renewal)", () => {
    const fee = netDeployFee(config, [
      line("price_fee_business", 5000, 1_700_000_000),
      line("price_cpu", 123, 1_700_000_000),
    ]);
    expect(fee).toEqual({ amountCents: 5000, periodEnd: 1_700_000_000, plan: "business" });
  });

  it("nets a mid-cycle upgrade's proration pair to the top-up", () => {
    // always_invoice upgrade Starter -> Business: unused Starter credited,
    // prorated Business charged. The net is exactly the credits to top up.
    const fee = netDeployFee(config, [
      line("price_fee_starter", -300, 1_700_000_000),
      line("price_fee_business", 2800, 1_700_000_000),
    ]);
    expect(fee?.amountCents).toBe(2500);
  });

  it("nets a downgrade negative, which grants nothing upstream", () => {
    const fee = netDeployFee(config, [
      line("price_fee_business", -2800, 1_700_000_000),
      line("price_fee_starter", 300, 1_700_000_000),
    ]);
    expect(fee?.amountCents).toBeLessThan(0);
  });

  it("ignores metered and unrelated lines", () => {
    expect(
      netDeployFee(config, [
        line("price_cpu", 866, 1_700_000_000),
        line("price_api_plan", 7500, 1_700_000_000),
      ]),
    ).toBeNull();
  });

  it("uses the latest period end across fee lines", () => {
    const fee = netDeployFee(config, [
      line("price_fee_starter", -300, 1_700_000_000),
      line("price_fee_business", 2800, 1_700_500_000),
    ]);
    expect(fee?.periodEnd).toBe(1_700_500_000);
  });

  it("ignores metered lines", () => {
    expect(netDeployFee(config, [line("price_cpu", 866, 1_700_000_000)])).toBeNull();
  });

  it("ignores lines without price details (e.g. credit lines)", () => {
    const creditLine = {
      amount: -500,
      period: { end: 1_700_000_000, start: 1_699_996_400 },
      pricing: null,
    } as unknown as Stripe.InvoiceLineItem;
    expect(netDeployFee(config, [creditLine])).toBeNull();
  });

  it("labels the plan from the charge line regardless of proration order", () => {
    // Negative (departing) line first: plan must still reflect the new plan,
    // since that is what the credit name and metadata are read against.
    const fee = netDeployFee(config, [
      line("price_fee_starter", -300, 1_700_000_000),
      line("price_fee_business", 2800, 1_700_000_000),
    ]);
    expect(fee?.plan).toBe("business");
  });

  it("subtracts discount_amounts so a coupon reduces the credit", () => {
    // $50 fee with a $10 coupon allocated to the line (Stripe distributes
    // invoice-level coupons this way): the grant tracks the $40 actually paid.
    const fee = netDeployFee(config, [line("price_fee_business", 5000, 1_700_000_000, 1000)]);
    expect(fee?.amountCents).toBe(4000);
  });
});

function invoiceStub(overrides: Partial<Stripe.Invoice> = {}): Stripe.Invoice {
  const periodEnd = Math.floor(Date.now() / 1000) + 7 * 24 * 3600;
  return {
    id: "in_test",
    customer: "cus_test",
    currency: "usd",
    lines: {
      has_more: false,
      data: [line("price_fee_business", 5000, periodEnd)],
    },
    ...overrides,
  } as unknown as Stripe.Invoice;
}

describe("grantDeployCreditsForInvoice", () => {
  afterEach(() => {
    vi.mocked(deployBillingConfig).mockReset();
  });

  it("grants credits for a paid renewal fee line", async () => {
    vi.mocked(deployBillingConfig).mockResolvedValue(config);
    const create = vi.fn().mockResolvedValue({ id: "credgrant_test" });
    const list = vi.fn().mockReturnValue((async function* () {})());
    const stripe = {
      billing: { creditGrants: { create, list } },
    } as unknown as Stripe;

    const result = await grantDeployCreditsForInvoice(stripe, invoiceStub());
    expect(result).toEqual({
      granted: true,
      grantId: "credgrant_test",
      amountCents: 5000,
      periodTotalCents: 5000,
    });
    expect(create).toHaveBeenCalledOnce();
  });

  it("skips when billing is not configured", async () => {
    vi.mocked(deployBillingConfig).mockResolvedValue(null);
    const stripe = {
      billing: { creditGrants: { create: vi.fn(), list: vi.fn() } },
    } as unknown as Stripe;
    const result = await grantDeployCreditsForInvoice(stripe, invoiceStub());
    expect(result.granted).toBe(false);
  });

  it("skips duplicate grants for the same invoice", async () => {
    vi.mocked(deployBillingConfig).mockResolvedValue(config);
    const periodEnd = Math.floor(Date.now() / 1000) + 7 * 24 * 3600;
    const expiresAt = periodEnd + 3 * 24 * 3600;
    const list = vi.fn().mockReturnValue(
      (async function* () {
        yield {
          id: "credgrant_existing",
          expires_at: expiresAt,
          amount: { monetary: { value: 5000 } },
          metadata: { stripe_invoice_id: "in_test" },
        };
      })(),
    );
    const stripe = {
      billing: { creditGrants: { create: vi.fn(), list } },
    } as unknown as Stripe;

    const result = await grantDeployCreditsForInvoice(stripe, invoiceStub());
    expect(result).toMatchObject({
      granted: false,
      reason: expect.stringContaining("already granted"),
      periodTotalCents: 5000,
    });
  });
});
