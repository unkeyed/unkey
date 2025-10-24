import type Stripe from "stripe";

export const mapProduct = (p: Stripe.Product) => {
  if (!p.default_price) {
    throw new Error(`Product ${p.id} is missing default_price`);
  }

  const price = typeof p.default_price === "string" ? null : (p.default_price as Stripe.Price);

  if (!price) {
    throw new Error(`Product ${p.id} default_price must be expanded`);
  }

  if (price.unit_amount === null || price.unit_amount === undefined) {
    throw new Error(`Product ${p.id} price is missing unit_amount`);
  }

  const quotaValue = Number.parseInt(p.metadata.quota_requests_per_month, 10);

  return {
    id: p.id,
    name: p.name,
    priceId: price.id,
    dollar: price.unit_amount / 100,
    quotas: {
      requestsPerMonth: quotaValue,
    },
  };
};
