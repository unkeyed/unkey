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
  return (b.updatedAt ?? -1) - (a.updatedAt ?? -1) || byIdDescending(a, b);
}

export function pickPrimaryApp<T extends CardApp>(apps: ReadonlyArray<T>): T | undefined {
  return apps.toSorted(byLatestUpdate).find((app) => app.currentDeploymentId !== null);
}

export function buildProjectApps(projectId: string, input: CardInput): ProjectApp[] {
  const recentByApp = Map.groupBy(input.recentDeployments, (d) => d.appId);
  const domainsByApp = Map.groupBy(
    input.productionDomains.filter((d) => d.verificationStatus === "verified"),
    (d) => d.appId,
  );

  return input.apps
    .filter(({ app }) => app.projectId === projectId)
    .toSorted((a, b) => byLatestUpdate(a.app, b.app))
    .map(({ app, current }) => {
      const recent = recentByApp.get(app.id) ?? [];
      const deployments = current ? [current, ...recent] : recent;
      const production = deployments.filter((d) => d.environmentId === current?.environmentId);
      const headline = (production.length > 0 ? production : deployments).toSorted(
        (a, b) => b.createdAt - a.createdAt || byIdDescending(a, b),
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

function byIdDescending(a: { id: string }, b: { id: string }): number {
  return b.id < a.id ? -1 : b.id > a.id ? 1 : 0;
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
  apps: ReadonlyArray<{ app: CardApp; current?: CardDeployment }>;
  recentDeployments: ReadonlyArray<CardDeployment>;
  productionDomains: ReadonlyArray<CardDomain>;
};
