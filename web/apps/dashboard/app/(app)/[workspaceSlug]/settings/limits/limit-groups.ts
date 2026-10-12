import type {
  V2WorkspaceGetLimitsLimit,
  V2WorkspaceGetLimitsResponseData,
} from "@unkey/api/models/components";

export type LimitStatus = "ok" | "at-limit" | "over";

export type GroupKey = "api" | "logs" | "compute";

export type RowUsage = { value: number; max: number; label: string };

export type RowValue =
  | { state: "loading"; metered: boolean }
  | { state: "ready"; limit: string; usage?: RowUsage };

export type LimitRow = {
  name: string;
  /** Set when the row needs its own banner copy instead of its group's. */
  breachKey?: BreachKey;
  description?: string;
  value: RowValue;
  status: LimitStatus;
};

export type LimitGroup = {
  key: GroupKey;
  title: string;
  description: string;
  rows: LimitRow[];
};

type RowText = Pick<LimitRow, "name" | "breachKey" | "description">;

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

function statusOf(usage: RowUsage): LimitStatus {
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

function metered(
  text: RowText,
  entry: V2WorkspaceGetLimitsLimit | undefined,
  format: (value: number) => string,
): LimitRow {
  if (entry === undefined) {
    return { ...text, value: { state: "loading", metered: true }, status: "ok" };
  }
  if (entry.limit === null) {
    return ceiling(text, "Unlimited");
  }
  const value = entry.current ?? 0;
  const usage = { value, max: entry.limit, label: format(value) };
  return {
    ...text,
    value: { state: "ready", limit: format(entry.limit), usage },
    status: statusOf(usage),
  };
}

function ceiling(text: RowText, limit: string | undefined): LimitRow {
  return {
    ...text,
    value: limit === undefined ? { state: "loading", metered: false } : { state: "ready", limit },
    status: "ok",
  };
}

function capped(
  entry: V2WorkspaceGetLimitsLimit | undefined,
  format: (value: number) => string,
): string | undefined {
  if (entry === undefined) {
    return undefined;
  }
  return entry.limit === null ? "Unlimited" : format(entry.limit);
}

function apiGroup(limits: V2WorkspaceGetLimitsResponseData | undefined): LimitGroup {
  return {
    key: "api",
    title: "API management",
    description: "Operation and request limits for the Unkey API.",
    rows: [
      metered(
        {
          name: "Monthly API operations",
          description: "Billable key verifications and rate limit operations each month.",
        },
        limits?.apiBillableOperationsCountMaxPerMonth,
        count,
      ),
      ceiling(
        { name: "API requests per minute" },
        capped(limits?.apiRequestsCountMaxPerMinute, (value) => `${count(value)} / min`),
      ),
    ],
  };
}

function logGroup(limits: V2WorkspaceGetLimitsResponseData | undefined): LimitGroup {
  return {
    key: "logs",
    title: "Logs",
    description: "Retention periods for operational and audit data.",
    rows: [
      ceiling(
        {
          name: "Log retention",
          description: "How long request and runtime logs remain available.",
        },
        capped(limits?.logsRetentionDaysMax, days),
      ),
      ceiling({ name: "Audit log retention" }, capped(limits?.logsAuditRetentionDaysMax, days)),
      metered({ name: "Log drains" }, limits?.logdrainsMax, count),
    ],
  };
}

function customDomainsRow(customDomains: V2WorkspaceGetLimitsLimit | undefined): LimitRow {
  const text = {
    name: "Custom domains",
    description: "Domains you can attach across all apps in this workspace.",
    breachKey: "domains",
  } as const;

  // While loading, the plan is unknown, so the row loads as a meter like a capped plan
  if (customDomains === undefined) {
    return metered(text, undefined, count);
  }

  if (customDomains.limit === null) {
    return ceiling(text, "Unlimited");
  }

  // A meter of 0 against 0 tells the reader nothing. The plan simply does not
  // include the feature, which is how the docs say it too.
  if (customDomains.limit === 0) {
    return ceiling(text, "Not included");
  }

  return metered(text, customDomains, count);
}

function computeGroup(limits: V2WorkspaceGetLimitsResponseData | undefined): LimitGroup {
  return {
    key: "compute",
    title: "Compute",
    description: "Workspace and per-instance resource ceilings.",
    rows: [
      metered(
        { name: "Workspace CPU", description: "Total CPU across all your apps." },
        limits?.cpuCoresMax,
        vCpus,
      ),
      ceiling({ name: "CPU per instance" }, capped(limits?.cpuCoresMaxPerInstance, vCpus)),
      metered(
        { name: "Workspace memory", description: "Total memory across all your apps." },
        limits?.memoryMibMax,
        mib,
      ),
      ceiling({ name: "Memory per instance" }, capped(limits?.memoryMibMaxPerInstance, mib)),
      metered(
        { name: "Workspace ephemeral disk", description: "Total disk across all your apps." },
        limits?.storageMibMax,
        mib,
      ),
      ceiling(
        { name: "Ephemeral disk per instance" },
        capped(limits?.storageMibMaxPerInstance, mib),
      ),
      ceiling({ name: "Concurrent builds" }, capped(limits?.buildsConcurrentMax, count)),
      ceiling(
        {
          name: "Replicas per region",
          description: "Instances autoscaling can run for one app in a region.",
        },
        capped(limits?.autoscalingReplicasMax, count),
      ),
      customDomainsRow(limits?.customDomainsMax),
    ],
  };
}

/**
 * Builds the page from loaded limits, or the same page in its loading state
 * when limits is undefined. hasComputePlan decides the Compute group only while
 * loading, because loaded limits carry the Compute limits exactly when the plan
 * has it
 */
export function buildLimitGroups(
  limits: V2WorkspaceGetLimitsResponseData | undefined,
  hasComputePlan: boolean,
): LimitGroup[] {
  const groups = [apiGroup(limits), logGroup(limits)];
  const showCompute = limits === undefined ? hasComputePlan : limits.cpuCoresMax !== undefined;
  if (showCompute) {
    groups.push(computeGroup(limits));
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
