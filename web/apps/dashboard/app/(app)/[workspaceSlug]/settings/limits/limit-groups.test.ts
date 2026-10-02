import { type LimitsPlan, limitsByPlan } from "@/lib/limits";
import type {
  V2WorkspaceGetLimitsCompute,
  V2WorkspaceGetLimitsResponseData,
} from "@unkey/api/models/components";
import { describe, expect, it } from "vitest";
import { type LimitGroup, breachedKeys, buildLimitGroups } from "./limit-groups";

const ROW = "Custom domains";

function computeFor(plan: LimitsPlan, attachedDomains: number): V2WorkspaceGetLimitsCompute {
  const limits = limitsByPlan[plan];
  return {
    vCpus: { limit: limits.cpuCoresMax, used: 0 },
    vCpusPerInstance: limits.cpuCoresMaxPerInstance,
    memoryMib: { limit: limits.memoryMibMax, used: 0 },
    memoryMibPerInstance: limits.memoryMibMaxPerInstance,
    storageMib: { limit: limits.storageMibMax, used: 0 },
    storageMibPerInstance: limits.storageMibMaxPerInstance,
    concurrentBuilds: limits.buildsConcurrentMax,
    replicasPerRegion: limits.autoscalingReplicasMax,
    customDomains: { limit: limits.customDomainsMax, used: attachedDomains },
  };
}

function responseFor(plan: LimitsPlan, attachedDomains: number): V2WorkspaceGetLimitsResponseData {
  const limits = limitsByPlan[plan];
  return {
    api: {
      billableOperations: { limit: limits.apiBillableOperationsCountMaxPerMonth, used: 0 },
    },
    logs: {
      retentionDays: limits.logsRetentionDaysMax,
      auditRetentionDays: limits.logsAuditRetentionDaysMax,
      logDrains: { limit: limits.logdrainsMax, used: 0 },
    },
    compute: computeFor(plan, attachedDomains),
  };
}

function groupsFor(plan: LimitsPlan, attachedDomains: number): LimitGroup[] {
  return buildLimitGroups(responseFor(plan, attachedDomains));
}

function domainsRow(groups: LimitGroup[]) {
  return groups.flatMap((group) => group.rows).find((row) => row.name === ROW);
}

describe("log drains row", () => {
  it("meters the current count against the workspace allowance", () => {
    const response = responseFor("free", 0);
    const groups = buildLimitGroups({
      ...response,
      logs: { ...response.logs, logDrains: { limit: 3, used: 1 } },
    });
    expect(groups.find((group) => group.key === "logs")?.rows).toContainEqual({
      name: "Log drains",
      limit: "3",
      usage: { value: 1, max: 3, label: "1" },
      status: "ok",
    });
  });
});

describe("workspace CPU row", () => {
  it("shows fractional reserved vCPUs against the workspace limit", () => {
    const groups = buildLimitGroups({
      ...responseFor("starter", 0),
      compute: { ...computeFor("starter", 0), vCpus: { limit: 30, used: 2.5 } },
    });
    const row = groups.flatMap((group) => group.rows).find((r) => r.name === "Workspace CPU");
    expect(row?.limit).toBe("30 vCPU");
    expect(row?.usage).toEqual({ value: 2.5, max: 30, label: "2.50 vCPU" });
  });
});

describe("custom domains row", () => {
  it("lives in the compute group, so it is hidden without a compute plan", () => {
    const compute = groupsFor("starter", 0).find((group) => group.key === "compute");
    expect(compute?.rows.map((row) => row.name)).toContain(ROW);

    const withoutPlan = buildLimitGroups({ ...responseFor("starter", 0), compute: undefined });
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
    expect(row?.usage).toEqual({ value: 0, max: 1, label: "0" });
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
    const groups = buildLimitGroups({
      ...responseFor("starter", 0),
      compute: { ...computeFor("starter", 0), storageMib: { limit: 0, used: 512 } },
    });
    expect(breachedKeys(groups)).toEqual(["compute"]);
  });
});
