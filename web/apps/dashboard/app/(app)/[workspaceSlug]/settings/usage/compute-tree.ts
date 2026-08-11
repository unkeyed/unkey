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
  const projects = new Map<string, Map<string, UsageEnvironment[]>>();
  const projectNames = new Map<string, string>();
  const appNames = new Map<string, string>();
  const gateways = new Map<string, UsageGateway>();

  for (const row of gateway) {
    projectNames.set(row.projectId, label(row.projectId, row.projectName));
    if (!projects.has(row.projectId)) {
      projects.set(row.projectId, new Map<string, UsageEnvironment[]>());
    }
    const current = gateways.get(row.projectId) ?? { activeKeys: 0, microCents: 0 };
    gateways.set(row.projectId, {
      activeKeys: current.activeKeys + row.activeKeys,
      microCents: current.microCents + row.grossMicroCents,
    });
  }

  for (const row of usage) {
    projectNames.set(row.projectId, label(row.projectId, row.projectName));
    appNames.set(row.appId, label(row.appId, row.appName));

    const apps = projects.get(row.projectId) ?? new Map<string, UsageEnvironment[]>();
    const environments = apps.get(row.appId) ?? [];
    environments.push({
      environmentId: row.environmentId,
      name: label(row.environmentId, row.environmentSlug),
      cpuHours: row.cpuSeconds / SECONDS_PER_HOUR,
      memoryGiBHours: row.memoryGiBHours,
      egressGiB: row.egressGiB,
      diskGiBHours: row.diskGiBHours,
      microCents: row.grossMicroCents,
    });
    apps.set(row.appId, environments);
    projects.set(row.projectId, apps);
  }

  const tree = [...projects.entries()].map(([projectId, apps]): UsageProject => {
    const appNodes = [...apps.entries()].map(([appId, environments]): UsageApp => {
      environments.sort(byCostDescending);
      return {
        appId,
        name: appNames.get(appId) ?? UNATTRIBUTED,
        environments,
        ...rollUp(environments),
      };
    });
    appNodes.sort(byCostDescending);

    const projectGateway = gateways.get(projectId) ?? { activeKeys: 0, microCents: 0 };
    const compute = rollUp(appNodes);

    return {
      projectId,
      name: projectNames.get(projectId) ?? UNATTRIBUTED,
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
