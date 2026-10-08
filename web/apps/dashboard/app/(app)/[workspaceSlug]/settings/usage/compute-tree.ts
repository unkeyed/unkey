import {
  type DeployMeterCostsCents,
  MICRO_CENTS_PER_CENT,
  priceActiveKeysMicroCents,
  priceComputeMeterMicroCents,
  priceDeployMetersCents,
} from "@/lib/billing/deployPricing";
import type { V2WorkspaceGetUsageBreakdowns } from "@unkey/api/models/components";

const SECONDS_PER_HOUR = 3600;

const UNATTRIBUTED = "Unattributed";

export type UsageQuantities = {
  cpuHours: number;
  memoryGiBHours: number;
  egressGiB: number;
  diskGiBHours: number;
};

export type UsageCostsCents = Omit<DeployMeterCostsCents, "activeKeys">;

type Priced = UsageQuantities & { microCents: number };

type ResourceLabel = { name: string; deleted: boolean };

export type UsageEnvironment = Priced &
  ResourceLabel & {
    environmentId: string;
  };

export type UsageApp = Priced &
  ResourceLabel & {
    appId: string;
    environments: UsageEnvironment[];
  };

export type UsageGateway = {
  activeKeys: number;
  microCents: number;
};

/** `microCents` is compute plus gateway, so it equals the rows shown beneath it. */
export type UsageProject = Priced &
  ResourceLabel & {
    projectId: string;
    apps: UsageApp[];
    gateway: UsageGateway;
  };

export type ComputeTree = {
  projects: UsageProject[];
  microCents: number;
};

/** Priced usage rows. An empty id means the usage has no project or app id. */
export type UsageBreakdown = {
  usage: Array<{
    projectId: string;
    projectName: string | null;
    appId: string;
    appName: string | null;
    environmentId: string;
    environmentSlug: string | null;
    cpuSeconds: number;
    memoryGiBHours: number;
    diskGiBHours: number;
    egressGiB: number;
    grossMicroCents: number;
  }>;
  gateway: Array<{
    projectId: string;
    projectName: string | null;
    appId: string;
    activeKeys: number;
    grossMicroCents: number;
  }>;
};

function zero(): Priced {
  return { cpuHours: 0, memoryGiBHours: 0, egressGiB: 0, diskGiBHours: 0, microCents: 0 };
}

function add(total: Priced, part: Priced): Priced {
  return {
    cpuHours: total.cpuHours + part.cpuHours,
    memoryGiBHours: total.memoryGiBHours + part.memoryGiBHours,
    egressGiB: total.egressGiB + part.egressGiB,
    diskGiBHours: total.diskGiBHours + part.diskGiBHours,
    microCents: total.microCents + part.microCents,
  };
}

function rollUp(parts: Priced[]): Priced {
  return parts.reduce(add, zero());
}

function label(
  id: string,
  name: string | null,
  resource: "project" | "app" | "environment",
): ResourceLabel {
  if (id === "") {
    return { name: UNATTRIBUTED, deleted: false };
  }
  const deleted = name === null || name === "";
  return { name: deleted ? `Deleted ${resource}` : name, deleted };
}

function byCostDescending(a: Priced, b: Priced): number {
  return b.microCents - a.microCents;
}

/**
 * Prices each workspace.getUsage row with the Deploy meter rates. A missing name
 * becomes null, and a missing app or project becomes "".
 */
export function breakdownFromUsage({
  byEnvironment,
  byApp,
}: V2WorkspaceGetUsageBreakdowns): UsageBreakdown {
  return {
    usage: byEnvironment.map((row) => {
      const meters = {
        cpuSeconds: row.compute.cpuSeconds,
        memoryGiBHours: row.compute.memoryGiBHours,
        diskGiBHours: row.compute.storageGiBHours,
        egressGiB: row.compute.egressGiB,
      };
      return {
        projectId: row.project.id,
        projectName: row.project.name ?? null,
        appId: row.app?.id ?? "",
        appName: row.app?.name ?? null,
        environmentId: row.environment.id,
        environmentSlug: row.environment.slug ?? null,
        ...meters,
        grossMicroCents: priceComputeMeterMicroCents(meters),
      };
    }),
    gateway: byApp.map((row) => ({
      projectId: row.project?.id ?? "",
      projectName: row.project?.name ?? null,
      appId: row.app?.id ?? "",
      activeKeys: row.gateway.activeKeys,
      grossMicroCents: priceActiveKeysMicroCents(row.gateway.activeKeys),
    })),
  };
}

export function buildComputeTree({ usage, gateway }: UsageBreakdown): ComputeTree {
  const gatewayByProject = Map.groupBy(gateway, (row) => row.projectId);
  const usageByProject = Map.groupBy(usage, (row) => row.projectId);
  const projectIds = new Set([...gatewayByProject.keys(), ...usageByProject.keys()]);

  const tree = [...projectIds].map((projectId): UsageProject => {
    const usageRows = usageByProject.get(projectId) ?? [];
    const gatewayRows = gatewayByProject.get(projectId) ?? [];

    const appNodes = [...Map.groupBy(usageRows, (row) => row.appId)]
      .map(([appId, rows]): UsageApp => {
        const environments = rows
          .map(
            (row): UsageEnvironment => ({
              environmentId: row.environmentId,
              ...label(row.environmentId, row.environmentSlug, "environment"),
              cpuHours: row.cpuSeconds / SECONDS_PER_HOUR,
              memoryGiBHours: row.memoryGiBHours,
              egressGiB: row.egressGiB,
              diskGiBHours: row.diskGiBHours,
              microCents: row.grossMicroCents,
            }),
          )
          .sort(byCostDescending);
        return {
          appId,
          ...label(appId, rows[0]?.appName ?? null, "app"),
          environments,
          ...rollUp(environments),
        };
      })
      .sort(byCostDescending);

    const projectGateway = gatewayRows.reduce<UsageGateway>(
      (total, row) => ({
        activeKeys: total.activeKeys + row.activeKeys,
        microCents: total.microCents + row.grossMicroCents,
      }),
      { activeKeys: 0, microCents: 0 },
    );
    const compute = rollUp(appNodes);

    return {
      projectId,
      ...label(
        projectId,
        usageRows[0]?.projectName ?? gatewayRows[0]?.projectName ?? null,
        "project",
      ),
      apps: appNodes,
      gateway: projectGateway,
      ...compute,
      microCents: compute.microCents + projectGateway.microCents,
    };
  });
  tree.sort(byCostDescending);

  return { projects: tree, microCents: rollUp(tree).microCents };
}

export function microCentsToDisplayCents(microCents: number): number {
  return Math.round(microCents / MICRO_CENTS_PER_CENT);
}

export function priceUsageQuantitiesCents(usage: UsageQuantities): UsageCostsCents {
  const costs = priceDeployMetersCents({
    cpuSeconds: usage.cpuHours * SECONDS_PER_HOUR,
    memoryGiBHours: usage.memoryGiBHours,
    egressGiB: usage.egressGiB,
    diskGiBHours: usage.diskGiBHours,
    activeKeys: 0,
  });

  return {
    cpu: costs.cpu,
    memory: costs.memory,
    egress: costs.egress,
    disk: costs.disk,
  };
}
