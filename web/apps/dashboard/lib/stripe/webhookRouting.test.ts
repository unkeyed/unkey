import { describe, expect, it } from "vitest";
import { keepsTeamAfterDelete, stripeWebhookResponse } from "./webhookRouting";

describe("stripeWebhookResponse", () => {
  it("makes the event result visible in Stripe delivery logs", async () => {
    const response = stripeWebhookResponse(
      { id: "evt_123", type: "customer.subscription.created" },
      "api_signup_alert_processed",
      { workspaceLinked: false, slackDelivery: "sent" },
    );

    expect(response.status).toBe(200);
    expect(response.headers.get("X-Unkey-Stripe-Event-Id")).toBe("evt_123");
    expect(response.headers.get("X-Unkey-Stripe-Event-Type")).toBe("customer.subscription.created");
    expect(response.headers.get("X-Unkey-Webhook-Result")).toBe("api_signup_alert_processed");
    await expect(response.json()).resolves.toEqual({
      eventId: "evt_123",
      eventType: "customer.subscription.created",
      result: "api_signup_alert_processed",
      details: { workspaceLinked: false, slackDelivery: "sent" },
    });
  });
});

describe("keepsTeamAfterDelete", () => {
  it("API delete keeps team while a team-enabled Deploy plan is active", () => {
    expect(keepsTeamAfterDelete("api", { tier: "Free", plan: "pro" })).toBe(true);
    expect(keepsTeamAfterDelete("api", { tier: "Free", plan: "business" })).toBe(true);
  });

  it("API delete drops team on Starter or when no Deploy plan remains", () => {
    expect(keepsTeamAfterDelete("api", { tier: "Pro", plan: "starter" })).toBe(false);
    expect(keepsTeamAfterDelete("api", { tier: "Pro", plan: null })).toBe(false);
  });

  it("Deploy delete keeps team while the API tier is paid", () => {
    expect(keepsTeamAfterDelete("compute", { tier: "Pro", plan: "pro" })).toBe(true);
  });

  it("Deploy delete drops team when the API tier is Free", () => {
    expect(keepsTeamAfterDelete("compute", { tier: "Free", plan: "pro" })).toBe(false);
  });

  it("Deploy delete treats a null tier as Free", () => {
    expect(keepsTeamAfterDelete("compute", { tier: null, plan: "pro" })).toBe(false);
  });
});
