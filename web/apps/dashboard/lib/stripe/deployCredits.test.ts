import type Stripe from "stripe";
import { describe, expect, it } from "vitest";
import type { DeployBillingConfig } from "./deployBilling";
import { netDeployFee } from "./deployCredits";

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
