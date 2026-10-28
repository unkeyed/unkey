import type { ProjectApp } from "@/lib/collections/deploy/project-cards";
import { plural } from "@/lib/fmt";

export type ProjectSummary = {
  appsByRecency: ProjectApp[];
  appsLabel: string;
  deployedAt: number | null;
};

export function projectSummaryOf(apps: ProjectApp[]): ProjectSummary {
  const appsByRecency = apps.toSorted(
    (a, b) => (b.headlineDeployment?.deployedAt ?? 0) - (a.headlineDeployment?.deployedAt ?? 0),
  );
  return {
    appsByRecency,
    appsLabel: apps.length === 0 ? "No apps yet" : plural(apps.length, "app"),
    deployedAt: appsByRecency[0]?.headlineDeployment?.deployedAt ?? null,
  };
}
