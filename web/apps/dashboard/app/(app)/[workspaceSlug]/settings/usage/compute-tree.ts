import { MICRO_CENTS_PER_CENT } from "@/lib/billing/deployPricing";
import type { DeployUsageBreakdown } from "@/lib/trpc/routers/billing/query-deploy-usage-breakdown";

const SECONDS_PER_HOUR = 3600;

const UNATTRIBUTED = "Unattributed";

export type UsageQuantities = {
  cpuHours: number;
  memoryGiBHours: number;
  egressGiB: number;
  diskGiBHours: number;
};

type Priced = UsageQuantities & { microCents: number };

export type UsageEnvironment = Priced & {
  environmentId: string;
  name: string;
};

export type UsageApp = Priced & {
  appId: string;
  name: string;
  environments: UsageEnvironment[];
};

export type UsageGateway = {
  activeKeys: number;
  microCents: number;
};

/** `microCents` is compute plus gateway, so it equals the rows shown beneath it. */
export type UsageProject = Priced & {
  projectId: string;
  name: string;
  apps: UsageApp[];
  gateway: UsageGateway;
};

export type ComputeTree = {
  projects: UsageProject[];
  microCents: number;
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

function label(id: string, name: string | null): string {
  if (id === "") {
    return UNATTRIBUTED;
  }
  return name === null || name === "" ? id : name;
}

function byCostDescending(a: Priced, b: Priced): number {
  return b.microCents - a.microCents;
}

export function buildComputeTree({ usage, gateway }: DeployUsageBreakdown): ComputeTree {
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
              name: label(row.environmentId, row.environmentSlug),
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
          name: label(appId, rows[0]?.appName ?? null),
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
      name: label(projectId, usageRows[0]?.projectName ?? gatewayRows[0]?.projectName ?? null),
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
