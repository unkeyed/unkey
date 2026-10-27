import type { ProjectApp } from "@/lib/collections/deploy/project-cards";

export type ProjectSummary = {
  appsByRecency: ProjectApp[];
  appsLabel: string;
  deployedAt: number | null;
};

export function appNoun(count: number): string {
  return count === 1 ? "app" : "apps";
}

export function projectSummaryOf(apps: ProjectApp[]): ProjectSummary {
  const appsByRecency = apps.toSorted(
    (a, b) => (b.headlineDeployment?.deployedAt ?? 0) - (a.headlineDeployment?.deployedAt ?? 0),
  );
  return {
    appsByRecency,
    appsLabel: apps.length === 0 ? "No apps yet" : `${apps.length} ${appNoun(apps.length)}`,
    deployedAt: appsByRecency[0]?.headlineDeployment?.deployedAt ?? null,
  };
}
