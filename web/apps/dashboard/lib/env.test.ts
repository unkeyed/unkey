import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

beforeEach(() => {
  vi.resetModules();
  vi.stubEnv("STRIPE_SECRET_KEY", "sk_test_placeholder");
  vi.stubEnv("STRIPE_PRODUCT_IDS_PRO", "prod_pro");
  vi.stubEnv("STRIPE_PRODUCT_IDS_ENTERPRISE", "prod_enterprise");
});

afterEach(() => {
  vi.unstubAllEnvs();
});

describe("stripeEnv", () => {
  it.each([undefined, "", "   ", "whsec_", "not-a-webhook-secret", "whsec_ secret"])(
    "disables Stripe when the webhook secret is invalid: %j",
    async (secret) => {
      vi.stubEnv("STRIPE_WEBHOOK_SECRET", secret);
      const { stripeEnv } = await import("./env");

      expect(stripeEnv()).toBeNull();
    },
  );

  it("accepts a configured webhook secret and trims surrounding whitespace", async () => {
    const secret = `whsec_${"a".repeat(32)}`;
    vi.stubEnv("STRIPE_WEBHOOK_SECRET", ` ${secret}\n`);
    const { stripeEnv } = await import("./env");

    expect(stripeEnv()?.STRIPE_WEBHOOK_SECRET).toBe(secret);
  });
});
