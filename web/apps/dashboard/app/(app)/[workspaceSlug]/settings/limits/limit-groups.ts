import type { Limits } from "@unkey/db";

export type LimitStatus = "ok" | "at-limit" | "over";

export type GroupKey = "api" | "logs" | "compute";

/**
 * A figure measured against a ceiling. Loading and error are part of the model
 * because the two endpoints that supply them can fail independently of the
 * limits row, and a row must never invent a number it does not have.
 */
export type RowUsage =
  | { state: "loading" }
  | { state: "error" }
  | { state: "ready"; value: number; max: number; label: string };

export type LimitRow = {
  name: string;
  description: string;
  limit: string;
  /** Present only where a workspace-wide total exists to measure. */
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
  if (usage?.state !== "ready" || usage.max === 0) {
    return "ok";
  }
  if (usage.value > usage.max) {
    return "over";
  }
  return usage.value >= usage.max ? "at-limit" : "ok";
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

/**
 * The workspace's limits row, grouped by the product each ceiling belongs to.
 * Every ceiling is the already-resolved value from that row, so the page never
 * has to work out which plan granted it.
 */
export function buildLimitGroups({
  limits,
  hasComputePlan,
  apiOperations,
  allocation,
}: {
  limits: Limits;
  hasComputePlan: boolean;
  apiOperations: Measured<number>;
  allocation: Measured<Allocation>;
}): LimitGroup[] {
  const api: LimitGroup = {
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
        description: "Requests accepted across the workspace each minute.",
        limit:
          limits.apiRequestsCountMaxPerMinute === null
            ? "Unlimited"
            : `${count(limits.apiRequestsCountMaxPerMinute)} / min`,
      }),
    ],
  };

  const logs: LimitGroup = {
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
        description: "How long workspace audit logs remain available.",
        limit: days(limits.logsAuditRetentionDaysMax),
      }),
    ],
  };

  if (!hasComputePlan) {
    return [api, logs];
  }

  const compute: LimitGroup = {
    key: "compute",
    title: "Compute",
    description: "Workspace and per-instance resource ceilings.",
    rows: [
      metered({
        name: "Workspace CPU",
        description: "CPU across every running instance, counted at full scale-out.",
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
        description: "Maximum CPU one instance can request.",
        limit: cores(limits.cpuCoresMaxPerInstance * MILLICORES_PER_CORE),
      }),
      metered({
        name: "Workspace memory",
        description: "Memory across every running instance, counted at full scale-out.",
        limit: mib(limits.memoryMibMax),
        usage: usageOf(allocation, (value) => value.totalMemoryMib, limits.memoryMibMax, mib),
      }),
      ceiling({
        name: "Memory per instance",
        description: "Maximum memory one instance can request.",
        limit: mib(limits.memoryMibMaxPerInstance),
      }),
      metered({
        name: "Workspace ephemeral disk",
        description: "Ephemeral disk across every running instance, counted at full scale-out.",
        limit: mib(limits.storageMibMax),
        usage: usageOf(allocation, (value) => value.totalStorageMib, limits.storageMibMax, mib),
      }),
      ceiling({
        name: "Ephemeral disk per instance",
        description: "Maximum ephemeral disk one instance can request.",
        limit: mib(limits.storageMibMaxPerInstance),
      }),
      ceiling({
        name: "Concurrent builds",
        description: "Builds that can run at the same time.",
        limit: count(limits.buildsConcurrentMax),
      }),
      ceiling({
        name: "Replicas per region",
        description: "Instances autoscaling can run for one app in a region.",
        limit: count(limits.autoscalingReplicasMax),
      }),
    ],
  };

  return [api, logs, compute];
}

export function breachedGroups(groups: LimitGroup[]): GroupKey[] {
  return groups
    .filter((group) => group.rows.some((row) => row.status !== "ok"))
    .map((group) => group.key);
}
