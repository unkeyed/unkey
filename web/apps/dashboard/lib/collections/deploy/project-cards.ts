import type { DeploymentStatus } from "./deployment-status";

export type ProjectApp = {
  id: string;
  name: string;
  customDomain: string | null;
  headlineDeployment: {
    id: string;
    status: DeploymentStatus;
    commitMessage: string | null;
    branch: string | null;
    deployedAt: number;
  } | null;
};

export function byLatestUpdate(a: CardApp, b: CardApp): number {
  return (b.updatedAt ?? -1) - (a.updatedAt ?? -1) || (b.id < a.id ? -1 : b.id > a.id ? 1 : 0);
}

export function pickPrimaryApp<T extends CardApp>(apps: ReadonlyArray<T>): T | undefined {
  return [...apps].sort(byLatestUpdate).find((app) => app.currentDeploymentId !== null);
}

export function buildProjectApps(projectId: string, input: CardInput): ProjectApp[] {
  const deploymentById = new Map(input.deployments.map((d) => [d.id, d]));
  const deploymentsByApp = groupBy(input.deployments, (d) => d.appId);
  const domainsByApp = groupBy(
    input.productionDomains.filter((d) => d.verificationStatus === "verified"),
    (d) => d.appId,
  );

  return input.apps
    .filter((app) => app.projectId === projectId)
    .toSorted(byLatestUpdate)
    .map((app) => {
      const productionEnvironmentId = app.currentDeploymentId
        ? deploymentById.get(app.currentDeploymentId)?.environmentId
        : undefined;
      const headline = (deploymentsByApp.get(app.id) ?? []).toSorted(
        (a, b) =>
          Number(b.environmentId === productionEnvironmentId) -
            Number(a.environmentId === productionEnvironmentId) ||
          b.createdAt - a.createdAt ||
          (b.id < a.id ? -1 : b.id > a.id ? 1 : 0),
      )[0];
      const domain = app.currentDeploymentId
        ? (domainsByApp.get(app.id) ?? []).toSorted((a, b) => a.domain.localeCompare(b.domain))[0]
        : undefined;
      return {
        id: app.id,
        name: app.name,
        customDomain: domain?.domain ?? null,
        headlineDeployment: headline
          ? {
              id: headline.id,
              status: headline.status,
              commitMessage: headline.gitCommitMessage ?? null,
              branch: headline.gitBranch || null,
              deployedAt: headline.createdAt,
            }
          : null,
      };
    });
}

type CardApp = {
  id: string;
  projectId: string;
  name: string;
  updatedAt: number | null;
  currentDeploymentId: string | null;
};

type CardDeployment = {
  id: string;
  appId: string;
  environmentId: string;
  status: DeploymentStatus;
  gitCommitMessage: string | null;
  gitBranch: string;
  createdAt: number;
};

type CardDomain = { appId: string; domain: string; verificationStatus: string };

type CardInput = {
  apps: ReadonlyArray<CardApp>;
  deployments: ReadonlyArray<CardDeployment>;
  productionDomains: ReadonlyArray<CardDomain>;
};

function groupBy<T>(items: ReadonlyArray<T>, key: (item: T) => string): Map<string, T[]> {
  const groups = new Map<string, T[]>();
  for (const item of items) {
    const group = groups.get(key(item));
    if (group) {
      group.push(item);
    } else {
      groups.set(key(item), [item]);
    }
  }
  return groups;
}
