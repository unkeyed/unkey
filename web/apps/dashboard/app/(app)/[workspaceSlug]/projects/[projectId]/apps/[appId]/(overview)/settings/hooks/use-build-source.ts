"use client";

import { collection } from "@/lib/collections";
import { ENVIRONMENT_KINDS, type Environment } from "@/lib/collections/deploy/environments";
import { trpc } from "@/lib/trpc/client";
import { and, eq, useLiveQuery } from "@tanstack/react-db";
import { match } from "@unkey/match";
import { useAppId, useProjectData } from "../../data-provider";

export function useEnvironmentsProductionFirst(): Environment[] {
  const { environments } = useProjectData();
  return [...environments].sort(
    (a, b) => ENVIRONMENT_KINDS.indexOf(a.kind) - ENVIRONMENT_KINDS.indexOf(b.kind),
  );
}

export function useApp() {
  const { projectId } = useProjectData();
  const appId = useAppId();
  const appQuery = useLiveQuery(
    (q) =>
      q
        .from({ app: collection.apps })
        .where(({ app }) => and(eq(app.projectId, projectId), eq(app.id, appId))),
    [projectId, appId],
  );
  return { projectId, appId, app: appQuery.data?.[0], isLoading: appQuery.isLoading };
}

export function useBuildSource() {
  const { projectId, appId, app } = useApp();
  const shouldLoadGitHub = app
    ? match(app.sourceType)
        .with("git", "unknown", () => true)
        .with("oci", () => false)
        .exhaustive()
    : false;
  const { data } = trpc.github.getInstallations.useQuery(
    { projectId, appId },
    { enabled: shouldLoadGitHub },
  );

  const hasRepository = app
    ? match(app.sourceType)
        .with("oci", () => false)
        .with("git", () => !data || Boolean(data.repoConnection?.repositoryFullName))
        .with("unknown", () => Boolean(data?.repoConnection?.repositoryFullName))
        .exhaustive()
    : false;

  const defaultBranch = data?.repoConnection?.defaultBranch ?? null;

  return { projectId, appId, app, hasRepository, defaultBranch };
}
