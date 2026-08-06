import type Stripe from "stripe";
import { describe, expect, it, vi } from "vitest";
import {
  computeLimitUpdateForPlan,
  computeLimitsForPlan,
  deployPlanGrantsTeam,
  detectDeployPlan,
  parseDeployPlan,
} from "./deployPlan";

// Minimal subscription stub. detectDeployPlan reads items[].price.metadata.plan.
function subWithItems(...items: Array<{ id?: string; plan?: string }>): Stripe.Subscription {
  return {
    id: "sub_test",
    items: {
      data: items.map(({ id, plan }) => ({
        price: {
          id: id ?? "price_x",
          metadata: {
            ...(plan === undefined ? {} : { plan }),
          },
        },
      })),
    },
  } as unknown as Stripe.Subscription;
}

describe("detectDeployPlan", () => {
  it("maps the plan-fee metadata to its plan", () => {
    expect(detectDeployPlan(subWithItems({ plan: "starter" }))).toBe("starter");
    expect(detectDeployPlan(subWithItems({ plan: "pro" }))).toBe("pro");
    expect(detectDeployPlan(subWithItems({ plan: "business" }))).toBe("business");
  });

  it("finds the tagged plan-fee item among other items", () => {
    const sub = subWithItems(
      { id: "price_api_plan" },
      { id: "price_metered_cpu" },
      { id: "price_plan_fee", plan: "pro" },
    );
    expect(detectDeployPlan(sub)).toBe("pro");
  });

  it("trims surrounding whitespace in the metadata value", () => {
    expect(detectDeployPlan(subWithItems({ plan: " pro " }))).toBe("pro");
  });

  it("returns null when no item carries plan metadata", () => {
    expect(detectDeployPlan(subWithItems({ id: "price_api_plan" }))).toBeNull();
  });

  it("returns null for a subscription with no items", () => {
    expect(detectDeployPlan(subWithItems())).toBeNull();
  });

  it("fails closed on an unrecognized plan value and logs a warning", () => {
    const warn = vi.spyOn(console, "warn").mockImplementation(() => {});
    expect(detectDeployPlan(subWithItems({ plan: "enterprise" }))).toBeNull();
    expect(warn).toHaveBeenCalledOnce();
    warn.mockRestore();
  });
});

describe("computeLimitsForPlan", () => {
  it("returns the advertised plan limits", () => {
    expect(computeLimitsForPlan("starter")).toEqual({
      logsRetentionDaysMax: 3,
      logsAuditRetentionDaysMax: 7,
      teamEnabled: false,
      cpuCoresMaxPerInstance: 2,
      memoryMibMaxPerInstance: 2_048,
      storageMibMaxPerInstance: 10_240,
      buildsConcurrentMax: 1,
    });
    expect(computeLimitsForPlan("pro")).toEqual({
      logsRetentionDaysMax: 7,
      logsAuditRetentionDaysMax: 14,
      teamEnabled: true,
      cpuCoresMaxPerInstance: 8,
      memoryMibMaxPerInstance: 8_192,
      storageMibMaxPerInstance: 10_240,
      buildsConcurrentMax: 1,
    });
    expect(computeLimitsForPlan("business")).toEqual({
      logsRetentionDaysMax: 14,
      logsAuditRetentionDaysMax: 30,
      teamEnabled: true,
      cpuCoresMaxPerInstance: 16,
      memoryMibMaxPerInstance: 32_768,
      storageMibMaxPerInstance: 10_240,
      buildsConcurrentMax: 1,
    });
  });

  it("returns the default Compute limits without a plan", () => {
    expect(computeLimitsForPlan(null)).toEqual({
      logsRetentionDaysMax: 7,
      logsAuditRetentionDaysMax: 30,
      teamEnabled: false,
      cpuCoresMaxPerInstance: 2,
      memoryMibMaxPerInstance: 4_096,
      storageMibMaxPerInstance: 10_240,
      buildsConcurrentMax: 1,
    });
  });

  it("preserves API-owned team and retention limits when an API plan is paid", () => {
    expect(computeLimitUpdateForPlan("business", true)).toEqual({
      cpuCoresMaxPerInstance: 16,
      memoryMibMaxPerInstance: 32_768,
      storageMibMaxPerInstance: 10_240,
      buildsConcurrentMax: 1,
    });
  });
});

describe("Deploy plan entitlement helpers", () => {
  it("grants team access only to Pro and Business", () => {
    expect(deployPlanGrantsTeam("starter")).toBe(false);
    expect(deployPlanGrantsTeam("pro")).toBe(true);
    expect(deployPlanGrantsTeam("business")).toBe(true);
    expect(deployPlanGrantsTeam(null)).toBe(false);
  });

  it("fails closed when parsing persisted plan values", () => {
    expect(parseDeployPlan("starter")).toBe("starter");
    expect(parseDeployPlan("enterprise")).toBeNull();
    expect(parseDeployPlan(null)).toBeNull();
  });
});
