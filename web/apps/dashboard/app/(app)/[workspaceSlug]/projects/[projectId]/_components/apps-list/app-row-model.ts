import { type AppSource, appSource } from "@/components/deploy/app-source";
import type { App } from "@/lib/collections/deploy/apps";
import { githubUrl } from "@/lib/github-url";
import type { Route } from "next";

export type AppRowData = {
  app: App;
  source: AppSource;
  deployment: App["headlineDeployment"];
  href: Route;
  commitUrl: string | undefined;
  branchUrl: string | undefined;
};

export function toAppRow(app: App, href: Route): AppRowData {
  const source = appSource(app);
  const deployment = app.headlineDeployment;
  const isGit = source === "git" && deployment !== null;
  return {
    app,
    source,
    deployment,
    href,
    commitUrl: isGit
      ? githubUrl.deployment({
          repoFullName: app.repositoryFullName,
          forkRepoFullName: deployment.forkRepositoryFullName,
          prNumber: deployment.prNumber,
          sha: deployment.commitSha,
        })
      : undefined,
    branchUrl: isGit
      ? githubUrl.branch(
          deployment.forkRepositoryFullName || app.repositoryFullName,
          deployment.branch,
        )
      : undefined,
  };
}

export function filterApps<T extends { app: App }>(rows: T[], search: string): T[] {
  const query = search.trim().toLowerCase();
  if (!query) {
    return rows;
  }
  return rows.filter(({ app }) =>
    [app.name, app.domain, app.repositoryFullName, app.imageReference].some((field) =>
      field?.toLowerCase().includes(query),
    ),
  );
}
