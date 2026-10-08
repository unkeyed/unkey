import { CUSTOM_DOMAINS_UNLIMITED, type LimitsPlan, limitsByPlan } from "@/lib/limits";
import type { V2WorkspaceGetLimitsResponseData } from "@unkey/api/models/components";
import { describe, expect, it } from "vitest";
import { type LimitGroup, breachedKeys, buildLimitGroups } from "./limit-groups";

const ROW = "Custom domains";

function withoutComputeFor(plan: LimitsPlan): V2WorkspaceGetLimitsResponseData {
  const limits = limitsByPlan[plan];
  return {
    apiBillableOperationsCountMaxPerMonth: {
      limit: limits.apiBillableOperationsCountMaxPerMonth,
      current: 0,
    },
    apiRequestsCountMaxPerMinute: { limit: null },
    logsRetentionDaysMax: { limit: limits.logsRetentionDaysMax },
    logsAuditRetentionDaysMax: { limit: limits.logsAuditRetentionDaysMax },
    logdrainsMax: { limit: limits.logdrainsMax, current: 0 },
  };
}

function responseFor(plan: LimitsPlan, attachedDomains: number): V2WorkspaceGetLimitsResponseData {
  const limits = limitsByPlan[plan];
  return {
    ...withoutComputeFor(plan),
    cpuCoresMax: { limit: limits.cpuCoresMax, current: 0 },
    cpuCoresMaxPerInstance: { limit: limits.cpuCoresMaxPerInstance },
    memoryMibMax: { limit: limits.memoryMibMax, current: 0 },
    memoryMibMaxPerInstance: { limit: limits.memoryMibMaxPerInstance },
    storageMibMax: { limit: limits.storageMibMax, current: 0 },
    storageMibMaxPerInstance: { limit: limits.storageMibMaxPerInstance },
    buildsConcurrentMax: { limit: limits.buildsConcurrentMax },
    autoscalingReplicasMax: { limit: limits.autoscalingReplicasMax },
    // The API sends a null limit for unlimited plans
    customDomainsMax: {
      limit: limits.customDomainsMax >= CUSTOM_DOMAINS_UNLIMITED ? null : limits.customDomainsMax,
      current: attachedDomains,
    },
  };
}

function groupsFor(plan: LimitsPlan, attachedDomains: number): LimitGroup[] {
  return buildLimitGroups(responseFor(plan, attachedDomains), true);
}

function domainsRow(groups: LimitGroup[]) {
  return groups.flatMap((group) => group.rows).find((row) => row.name === ROW);
}

describe("API requests per minute row", () => {
  function requestsRow(requestsPerMinute: number | null) {
    const groups = buildLimitGroups(
      { ...responseFor("free", 0), apiRequestsCountMaxPerMinute: { limit: requestsPerMinute } },
      true,
    );
    return groups
      .flatMap((group) => group.rows)
      .find((row) => row.name === "API requests per minute");
  }

  it("reads 'Unlimited' when the workspace has no per-minute limit", () => {
    expect(requestsRow(null)?.value).toEqual({ state: "ready", limit: "Unlimited" });
  });

  it("shows the per-minute limit", () => {
    expect(requestsRow(1000)?.value).toEqual({ state: "ready", limit: "1,000 / min" });
  });
});

describe("log drains row", () => {
  it("meters the current count against the workspace allowance", () => {
    const groups = buildLimitGroups(
      { ...responseFor("free", 0), logdrainsMax: { limit: 3, current: 1 } },
      true,
    );
    expect(groups.find((group) => group.key === "logs")?.rows).toContainEqual({
      name: "Log drains",
      value: { state: "ready", limit: "3", usage: { value: 1, max: 3, label: "1" } },
      status: "ok",
    });
  });
});

describe("workspace CPU row", () => {
  it("shows fractional reserved vCPUs against the workspace limit", () => {
    const groups = buildLimitGroups(
      { ...responseFor("starter", 0), cpuCoresMax: { limit: 30, current: 2.5 } },
      true,
    );
    const row = groups.flatMap((group) => group.rows).find((r) => r.name === "Workspace CPU");
    expect(row?.value).toEqual({
      state: "ready",
      limit: "30 vCPU",
      usage: { value: 2.5, max: 30, label: "2.50 vCPU" },
    });
  });
});

describe("custom domains row", () => {
  it("lives in the compute group, so it is hidden without a compute plan", () => {
    const compute = groupsFor("starter", 0).find((group) => group.key === "compute");
    expect(compute?.rows.map((row) => row.name)).toContain(ROW);

    const withoutPlan = buildLimitGroups(withoutComputeFor("starter"), true);
    expect(domainsRow(withoutPlan)).toBeUndefined();
  });

  it("reads 'Not included' with no meter when the plan allows none", () => {
    const row = domainsRow(groupsFor("free", 0));
    expect(row?.value).toEqual({ state: "ready", limit: "Not included" });
    expect(row?.status).toBe("ok");
  });

  it("reads 'Unlimited' with no meter on the uncapped plans", () => {
    for (const plan of ["pro", "business"] as const) {
      const row = domainsRow(groupsFor(plan, 3));
      expect(row?.value).toEqual({ state: "ready", limit: "Unlimited" });
    }
  });

  it("meters the attached count against a real cap", () => {
    const row = domainsRow(groupsFor("starter", 0));
    expect(row?.value).toEqual({
      state: "ready",
      limit: "1",
      usage: { value: 0, max: 1, label: "0" },
    });
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
    // A storage limit of 0 with disk reserved is a compute breach, not a domain one.
    const groups = buildLimitGroups(
      { ...responseFor("starter", 0), storageMibMax: { limit: 0, current: 512 } },
      true,
    );
    expect(breachedKeys(groups)).toEqual(["compute"]);
  });
});

describe("loading page", () => {
  // The loading page is the loaded page with values still loading, so nothing
  // moves when the limits arrive
  function shape(groups: LimitGroup[]) {
    return groups.map((group) => ({
      key: group.key,
      title: group.title,
      description: group.description,
      rows: group.rows.map((row) => ({
        name: row.name,
        description: row.description,
        metered: row.value.state === "loading" ? row.value.metered : row.value.usage !== undefined,
      })),
    }));
  }

  it("has the groups, rows, and meters of the loaded page", () => {
    expect(shape(buildLimitGroups(undefined, true))).toEqual(shape(groupsFor("starter", 0)));
  });

  it("shows no Compute group without a compute plan", () => {
    const loading = buildLimitGroups(undefined, false);
    expect(loading.map((group) => group.key)).toEqual(["api", "logs"]);
    expect(shape(loading)).toEqual(shape(buildLimitGroups(withoutComputeFor("starter"), false)));
  });

  it("loads every value and reports no breach", () => {
    const loading = buildLimitGroups(undefined, true);
    expect(
      loading.flatMap((group) => group.rows).every((row) => row.value.state === "loading"),
    ).toBe(true);
    expect(breachedKeys(loading)).toEqual([]);
  });
});
