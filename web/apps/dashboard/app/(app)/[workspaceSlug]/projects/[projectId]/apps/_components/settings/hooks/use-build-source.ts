"use client";

import { trpc } from "@/lib/trpc/client";
import { match } from "@unkey/match";
import { useApp } from "./use-app";

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

  return {
    projectId,
    appId,
    app,
    hasRepository,
    defaultBranch: data?.repoConnection?.defaultBranch ?? null,
    repositoryFullName: data?.repoConnection?.repositoryFullName ?? null,
  };
}
