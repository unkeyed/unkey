import type Stripe from "stripe";
import { describe, expect, it } from "vitest";
import { netDeployFee } from "./deployCredits";
import type { DeployBillingConfig } from "./deployPlans";

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

// Minimal invoice line stub in the legacy (pre-basil) webhook shape, where
// the price rides on line.price. netDeployFee reads amount, period.end, and
// the price id (dual-shape via invoiceCompat).
function legacyLine(priceId: string, amount: number, periodEnd: number): Stripe.InvoiceLineItem {
  return {
    amount,
    period: { end: periodEnd, start: periodEnd - 3600 },
    price: { id: priceId },
  } as unknown as Stripe.InvoiceLineItem;
}

// The same stub in the modern (basil+) shape, where the price id lives under
// pricing.price_details.price.
function modernLine(priceId: string, amount: number, periodEnd: number): Stripe.InvoiceLineItem {
  return {
    amount,
    period: { end: periodEnd, start: periodEnd - 3600 },
    pricing: { type: "price_details", price_details: { price: priceId } },
  } as unknown as Stripe.InvoiceLineItem;
}

describe("netDeployFee", () => {
  it("returns the fee of a single plan-fee line (subscribe / renewal)", () => {
    const fee = netDeployFee(config, [
      legacyLine("price_fee_business", 5000, 1_700_000_000),
      legacyLine("price_cpu", 123, 1_700_000_000),
    ]);
    expect(fee).toEqual({ amountCents: 5000, periodEnd: 1_700_000_000, plan: "business" });
  });

  it("nets a mid-cycle upgrade's proration pair to the top-up", () => {
    // always_invoice upgrade Starter -> Business: unused Starter credited,
    // prorated Business charged. The net is exactly the credits to top up.
    const fee = netDeployFee(config, [
      legacyLine("price_fee_starter", -300, 1_700_000_000),
      legacyLine("price_fee_business", 2800, 1_700_000_000),
    ]);
    expect(fee?.amountCents).toBe(2500);
  });

  it("nets a downgrade negative, which grants nothing upstream", () => {
    const fee = netDeployFee(config, [
      legacyLine("price_fee_business", -2800, 1_700_000_000),
      legacyLine("price_fee_starter", 300, 1_700_000_000),
    ]);
    expect(fee?.amountCents).toBeLessThan(0);
  });

  it("ignores metered and unrelated lines", () => {
    expect(
      netDeployFee(config, [
        legacyLine("price_cpu", 866, 1_700_000_000),
        legacyLine("price_api_plan", 7500, 1_700_000_000),
      ]),
    ).toBeNull();
  });

  it("uses the latest period end across fee lines", () => {
    const fee = netDeployFee(config, [
      legacyLine("price_fee_starter", -300, 1_700_000_000),
      legacyLine("price_fee_business", 2800, 1_700_500_000),
    ]);
    expect(fee?.periodEnd).toBe(1_700_500_000);
  });

  it("reads modern basil-shaped lines too", () => {
    const fee = netDeployFee(config, [modernLine("price_fee_pro", 2500, 1_700_000_000)]);
    expect(fee).toEqual({ amountCents: 2500, periodEnd: 1_700_000_000, plan: "pro" });
  });
});
