import type { Limits } from "@unkey/db";
import { describe, expect, it } from "vitest";
import { type LimitsPlan, limitsByPlan } from "@/lib/limits";
import { breachedKeys, buildLimitGroups, type LimitGroup, type Measured } from "./limit-groups";

const ROW = "Custom domains";

function limitsFor(plan: LimitsPlan, overrides: Partial<Limits> = {}): Limits {
  return { ...limitsByPlan[plan], pk: 1, workspaceId: "ws_KEBAP", ...overrides } as Limits;
}

function groupsFor(plan: LimitsPlan, attached: number, overrides?: Partial<Limits>): LimitGroup[] {
  const ready: Measured<number> = { state: "ready", value: attached };
  return buildLimitGroups({
    limits: limitsFor(plan, overrides),
    hasComputePlan: true,
    apiOperations: ready,
    allocation: {
      state: "ready",
      value: { totalCpuMillicores: 0, totalMemoryMib: 0, totalStorageMib: 0 },
    },
    customDomains: ready,
  });
}

function domainsRow(groups: LimitGroup[]) {
  return groups.flatMap((group) => group.rows).find((row) => row.name === ROW);
}

describe("custom domains row", () => {
  it("lives in the compute group, so it is hidden without a compute plan", () => {
    const compute = groupsFor("starter", 0).find((group) => group.key === "compute");
    expect(compute?.rows.map((row) => row.name)).toContain(ROW);

    const withoutPlan = buildLimitGroups({
      limits: limitsFor("starter"),
      hasComputePlan: false,
      apiOperations: { state: "ready", value: 0 },
      allocation: { state: "loading" },
      customDomains: { state: "ready", value: 0 },
    });
    expect(domainsRow(withoutPlan)).toBeUndefined();
  });

  it("reads 'Not included' with no meter when the plan allows none", () => {
    const row = domainsRow(groupsFor("free", 0));
    expect(row?.limit).toBe("Not included");
    expect(row?.usage).toBeUndefined();
    expect(row?.status).toBe("ok");
  });

  it("reads 'Unlimited' with no meter on the uncapped plans", () => {
    for (const plan of ["pro", "business"] as const) {
      const row = domainsRow(groupsFor(plan, 3));
      expect(row?.limit).toBe("Unlimited");
      expect(row?.usage).toBeUndefined();
    }
  });

  it("meters the attached count against a real cap", () => {
    const row = domainsRow(groupsFor("starter", 0));
    expect(row?.limit).toBe("1");
    expect(row?.usage).toEqual({ state: "ready", value: 0, max: 1, label: "0" });
    expect(row?.status).toBe("ok");
  });

  it("is at-limit once the count reaches the cap", () => {
    expect(domainsRow(groupsFor("starter", 1))?.status).toBe("at-limit");
    expect(domainsRow(groupsFor("starter", 2))?.status).toBe("over");
  });
});

describe("breachedKeys", () => {
  it("reports a full domain cap as 'domains', not as its compute group", () => {
    expect(breachedKeys(groupsFor("starter", 1))).toEqual(["domains"]);
  });

  it("reports nothing while every row is under its limit", () => {
    expect(breachedKeys(groupsFor("starter", 0))).toEqual([]);
  });

  it("reports the group key for rows that carry no breachKey", () => {
    // storageMibMax 0 with disk allocated is a compute breach, not a domain one.
    const groups = buildLimitGroups({
      limits: limitsFor("starter", { storageMibMax: 0 }),
      hasComputePlan: true,
      apiOperations: { state: "ready", value: 0 },
      allocation: {
        state: "ready",
        value: { totalCpuMillicores: 0, totalMemoryMib: 0, totalStorageMib: 512 },
      },
      customDomains: { state: "ready", value: 0 },
    });
    expect(breachedKeys(groups)).toEqual(["compute"]);
  });
});
