import { CUSTOM_DOMAINS_UNLIMITED } from "@/lib/limits";
import type {
  LimitMeter,
  V2WorkspaceGetLimitsApi,
  V2WorkspaceGetLimitsCompute,
  V2WorkspaceGetLimitsLogs,
  V2WorkspaceGetLimitsResponseData,
  V2WorkspaceGetLimitsVcpuMeter,
} from "@unkey/api/models/components";

export type LimitStatus = "ok" | "at-limit" | "over";

export type GroupKey = "api" | "logs" | "compute";

export type RowUsage = { value: number; max: number; label: string };

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

const MIB_PER_GIB = 1024;

function count(value: number): string {
  return new Intl.NumberFormat("en-US").format(value);
}

function vCpus(value: number): string {
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
  if (!usage) {
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

function usageOf(
  meter: LimitMeter | V2WorkspaceGetLimitsVcpuMeter,
  format: (value: number) => string,
): RowUsage {
  return { value: meter.used, max: meter.limit, label: format(meter.used) };
}

function metered(row: Omit<LimitRow, "status">): LimitRow {
  return { ...row, status: statusOf(row.usage) };
}

function ceiling(row: Omit<LimitRow, "status" | "usage">): LimitRow {
  return { ...row, status: "ok" };
}

function apiGroup(api: V2WorkspaceGetLimitsApi): LimitGroup {
  return {
    key: "api",
    title: "API management",
    description: "Operation and request limits for the Unkey API.",
    rows: [
      metered({
        name: "Monthly API operations",
        description: "Billable key verifications and rate limit operations each month.",
        limit: count(api.billableOperations.limit),
        usage: usageOf(api.billableOperations, count),
      }),
      ceiling({
        name: "API requests per minute",
        limit:
          api.requestsPerMinute === undefined
            ? "Unlimited"
            : `${count(api.requestsPerMinute)} / min`,
      }),
    ],
  };
}

function logsGroup(logs: V2WorkspaceGetLimitsLogs): LimitGroup {
  return {
    key: "logs",
    title: "Logs",
    description: "Retention periods for operational and audit data.",
    rows: [
      ceiling({
        name: "Log retention",
        description: "How long request and runtime logs remain available.",
        limit: days(logs.retentionDays),
      }),
      ceiling({
        name: "Audit log retention",
        limit: days(logs.auditRetentionDays),
      }),
      metered({
        name: "Log drains",
        limit: count(logs.logDrains.limit),
        usage: usageOf(logs.logDrains, count),
      }),
    ],
  };
}

function customDomainsRow(customDomains: LimitMeter): LimitRow {
  const row = {
    name: "Custom domains",
    description: "Domains you can attach across all apps in this workspace.",
    breachKey: "domains",
  } as const;

  // A meter of 0 against 0 tells the reader nothing. The plan simply does not
  // include the feature, which is how the docs say it too.
  if (customDomains.limit === 0) {
    return ceiling({ ...row, limit: "Not included" });
  }

  if (customDomains.limit >= CUSTOM_DOMAINS_UNLIMITED) {
    return ceiling({ ...row, limit: "Unlimited" });
  }

  return metered({
    ...row,
    limit: count(customDomains.limit),
    usage: usageOf(customDomains, count),
  });
}

function computeGroup(compute: V2WorkspaceGetLimitsCompute): LimitGroup {
  return {
    key: "compute",
    title: "Compute",
    description: "Workspace and per-instance resource ceilings.",
    rows: [
      metered({
        name: "Workspace CPU",
        description: "Total CPU across all your apps.",
        limit: vCpus(compute.vCpus.limit),
        usage: usageOf(compute.vCpus, vCpus),
      }),
      ceiling({
        name: "CPU per instance",
        limit: vCpus(compute.vCpusPerInstance),
      }),
      metered({
        name: "Workspace memory",
        description: "Total memory across all your apps.",
        limit: mib(compute.memoryMib.limit),
        usage: usageOf(compute.memoryMib, mib),
      }),
      ceiling({
        name: "Memory per instance",
        limit: mib(compute.memoryMibPerInstance),
      }),
      metered({
        name: "Workspace ephemeral disk",
        description: "Total disk across all your apps.",
        limit: mib(compute.storageMib.limit),
        usage: usageOf(compute.storageMib, mib),
      }),
      ceiling({
        name: "Ephemeral disk per instance",
        limit: mib(compute.storageMibPerInstance),
      }),
      ceiling({
        name: "Concurrent builds",
        limit: count(compute.concurrentBuilds),
      }),
      ceiling({
        name: "Replicas per region",
        description: "Instances autoscaling can run for one app in a region.",
        limit: count(compute.replicasPerRegion),
      }),
      customDomainsRow(compute.customDomains),
    ],
  };
}

export function buildLimitGroups(limits: V2WorkspaceGetLimitsResponseData): LimitGroup[] {
  const groups = [apiGroup(limits.api), logsGroup(limits.logs)];
  if (limits.compute) {
    groups.push(computeGroup(limits.compute));
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
