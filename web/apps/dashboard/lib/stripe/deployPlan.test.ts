import type Stripe from "stripe";
import { describe, expect, it, vi } from "vitest";
import {
  computeQuotaUpdateForPlan,
  computeQuotasForPlan,
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

describe("computeQuotasForPlan", () => {
  it("returns the advertised plan quotas", () => {
    expect(computeQuotasForPlan("starter")).toEqual({
      logsRetentionDays: 3,
      auditLogsRetentionDays: 7,
      team: false,
      maxCpuMillicoresPerInstance: 2_000,
      maxMemoryMibPerInstance: 2_048,
      maxStorageMibPerInstance: 10_240,
      maxConcurrentBuilds: 1,
    });
    expect(computeQuotasForPlan("pro")).toEqual({
      logsRetentionDays: 7,
      auditLogsRetentionDays: 14,
      team: true,
      maxCpuMillicoresPerInstance: 8_000,
      maxMemoryMibPerInstance: 8_192,
      maxStorageMibPerInstance: 10_240,
      maxConcurrentBuilds: 1,
    });
    expect(computeQuotasForPlan("business")).toEqual({
      logsRetentionDays: 14,
      auditLogsRetentionDays: 30,
      team: true,
      maxCpuMillicoresPerInstance: 16_000,
      maxMemoryMibPerInstance: 32_768,
      maxStorageMibPerInstance: 10_240,
      maxConcurrentBuilds: 1,
    });
  });

  it("returns the default Compute limits without a plan", () => {
    expect(computeQuotasForPlan(null)).toEqual({
      logsRetentionDays: 7,
      auditLogsRetentionDays: 30,
      team: false,
      maxCpuMillicoresPerInstance: 2_000,
      maxMemoryMibPerInstance: 4_096,
      maxStorageMibPerInstance: 10_240,
      maxConcurrentBuilds: 1,
    });
  });

  it("preserves API-owned team and retention quotas when an API plan is paid", () => {
    expect(computeQuotaUpdateForPlan("business", true)).toEqual({
      maxCpuMillicoresPerInstance: 16_000,
      maxMemoryMibPerInstance: 32_768,
      maxStorageMibPerInstance: 10_240,
      maxConcurrentBuilds: 1,
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
