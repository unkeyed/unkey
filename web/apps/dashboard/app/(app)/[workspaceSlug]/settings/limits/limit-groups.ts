import type { Limits } from "@unkey/db";
import { CUSTOM_DOMAINS_UNLIMITED } from "@/lib/limits";

export type LimitStatus = "ok" | "at-limit" | "over";

export type GroupKey = "api" | "logs" | "compute";

export type RowUsage =
  | { state: "loading" }
  | { state: "error" }
  | { state: "ready"; value: number; max: number; label: string };

export type LimitRow = {
  name: string;
  /** Set when the row needs its own banner copy instead of its group's. */
  breachKey?: BreachKey;
  description?: string;
  limit: string;
  usage?: RowUsage;
  status: LimitStatus;
};

export type LimitGroup = {
  key: GroupKey;
  title: string;
  description: string;
  rows: LimitRow[];
};

export type Measured<T> = { state: "loading" } | { state: "error" } | { state: "ready"; value: T };

export type Allocation = {
  totalCpuMillicores: number;
  totalMemoryMib: number;
  totalStorageMib: number;
};

const MIB_PER_GIB = 1024;
const MILLICORES_PER_CORE = 1000;

function count(value: number): string {
  return new Intl.NumberFormat("en-US").format(value);
}

function cores(millicores: number): string {
  const value = millicores / MILLICORES_PER_CORE;
  return `${Number.isInteger(value) ? value : value.toFixed(2)} vCPU`;
}

function mib(value: number): string {
  if (value < MIB_PER_GIB) {
    return `${count(value)} MiB`;
  }
  const gib = value / MIB_PER_GIB;
  return `${Number.isInteger(gib) ? gib : gib.toFixed(1)} GiB`;
}

function days(value: number): string {
  return `${value} day${value === 1 ? "" : "s"}`;
}

function statusOf(usage: RowUsage | undefined): LimitStatus {
  if (usage?.state !== "ready") {
    return "ok";
  }
  if (usage.max === 0) {
    return usage.value > 0 ? "over" : "ok";
  }
  if (usage.value > usage.max) {
    return "over";
  }
  if (usage.value === usage.max) {
    return "at-limit";
  }
  return "ok";
}

function usageOf<T>(
  measured: Measured<T>,
  read: (value: T) => number,
  max: number,
  format: (value: number) => string,
): RowUsage {
  if (measured.state !== "ready") {
    return measured;
  }
  const value = read(measured.value);
  return { state: "ready", value, max, label: format(value) };
}

function metered(row: Omit<LimitRow, "status">): LimitRow {
  return { ...row, status: statusOf(row.usage) };
}

function ceiling(row: Omit<LimitRow, "status" | "usage">): LimitRow {
  return { ...row, status: "ok" };
}

function apiGroup(limits: Limits, apiOperations: Measured<number>): LimitGroup {
  return {
    key: "api",
    title: "API management",
    description: "Operation and request limits for the Unkey API.",
    rows: [
      metered({
        name: "Monthly API operations",
        description: "Billable key verifications and rate limit operations each month.",
        limit: count(limits.apiBillableOperationsCountMaxPerMonth),
        usage: usageOf(
          apiOperations,
          (value) => value,
          limits.apiBillableOperationsCountMaxPerMonth,
          count,
        ),
      }),
      ceiling({
        name: "API requests per minute",
        limit:
          limits.apiRequestsCountMaxPerMinute === null
            ? "Unlimited"
            : `${count(limits.apiRequestsCountMaxPerMinute)} / min`,
      }),
    ],
  };
}

function logsGroup(limits: Limits): LimitGroup {
  return {
    key: "logs",
    title: "Logs",
    description: "Retention periods for operational and audit data.",
    rows: [
      ceiling({
        name: "Log retention",
        description: "How long request and runtime logs remain available.",
        limit: days(limits.logsRetentionDaysMax),
      }),
      ceiling({
        name: "Audit log retention",
        limit: days(limits.logsAuditRetentionDaysMax),
      }),
    ],
  };
}

function customDomainsRow(limits: Limits, domains: Measured<number>): LimitRow {
  const row = {
    name: "Custom domains",
    description: "Domains you can attach across all apps in this workspace.",
    breachKey: "domains",
  } as const;

  // A meter of 0 against 0 tells the reader nothing. The plan simply does not
  // include the feature, which is how the docs say it too.
  if (limits.customDomainsMax === 0) {
    return ceiling({ ...row, limit: "Not included" });
  }

  if (limits.customDomainsMax >= CUSTOM_DOMAINS_UNLIMITED) {
    return ceiling({ ...row, limit: "Unlimited" });
  }

  return metered({
    ...row,
    limit: count(limits.customDomainsMax),
    usage: usageOf(domains, (value) => value, limits.customDomainsMax, count),
  });
}

function computeGroup(
  limits: Limits,
  allocation: Measured<Allocation>,
  customDomains: Measured<number>,
): LimitGroup {
  return {
    key: "compute",
    title: "Compute",
    description: "Workspace and per-instance resource ceilings.",
    rows: [
      metered({
        name: "Workspace CPU",
        description: "Total CPU across all your apps.",
        limit: cores(limits.cpuCoresMax * MILLICORES_PER_CORE),
        usage: usageOf(
          allocation,
          (value) => value.totalCpuMillicores,
          limits.cpuCoresMax * MILLICORES_PER_CORE,
          cores,
        ),
      }),
      ceiling({
        name: "CPU per instance",
        limit: cores(limits.cpuCoresMaxPerInstance * MILLICORES_PER_CORE),
      }),
      metered({
        name: "Workspace memory",
        description: "Total memory across all your apps.",
        limit: mib(limits.memoryMibMax),
        usage: usageOf(allocation, (value) => value.totalMemoryMib, limits.memoryMibMax, mib),
      }),
      ceiling({
        name: "Memory per instance",
        limit: mib(limits.memoryMibMaxPerInstance),
      }),
      metered({
        name: "Workspace ephemeral disk",
        description: "Total disk across all your apps.",
        limit: mib(limits.storageMibMax),
        usage: usageOf(allocation, (value) => value.totalStorageMib, limits.storageMibMax, mib),
      }),
      ceiling({
        name: "Ephemeral disk per instance",
        limit: mib(limits.storageMibMaxPerInstance),
      }),
      ceiling({
        name: "Concurrent builds",
        limit: count(limits.buildsConcurrentMax),
      }),
      ceiling({
        name: "Replicas per region",
        description: "Instances autoscaling can run for one app in a region.",
        limit: count(limits.autoscalingReplicasMax),
      }),
      customDomainsRow(limits, customDomains),
    ],
  };
}

export function buildLimitGroups({
  limits,
  hasComputePlan,
  apiOperations,
  allocation,
  customDomains,
}: {
  limits: Limits;
  hasComputePlan: boolean;
  apiOperations: Measured<number>;
  allocation: Measured<Allocation>;
  customDomains: Measured<number>;
}): LimitGroup[] {
  const groups = [apiGroup(limits, apiOperations), logsGroup(limits)];
  if (hasComputePlan) {
    groups.push(computeGroup(limits, allocation, customDomains));
  }
  return groups;
}

/**
 * The key that BreachBanner uses to select its message. A row can report under
 * its own key when the message of its group does not fit, as custom domains do.
 */
export type BreachKey = GroupKey | "domains";

export function breachedKeys(groups: LimitGroup[]): BreachKey[] {
  const keys = groups.flatMap((group) =>
    group.rows.filter((row) => row.status !== "ok").map((row) => row.breachKey ?? group.key),
  );
  return [...new Set(keys)];
}
