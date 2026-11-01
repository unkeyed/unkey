"use client";

import { type App, useApp, useAppScope } from "@/lib/deploy-data";
import { trpc } from "@/lib/trpc/client";

const SOURCES: Record<
  App["sourceType"],
  { loadsGitHub: boolean; hasRepository: (repository: string | null, loaded: boolean) => boolean }
> = {
  git: { loadsGitHub: true, hasRepository: (repository, loaded) => !loaded || repository !== null },
  unknown: { loadsGitHub: true, hasRepository: (repository) => repository !== null },
  oci: { loadsGitHub: false, hasRepository: () => false },
};

export function useBuildSource(
  options: { staleTime?: number; refetchOnWindowFocus?: boolean } = {},
) {
  const { projectId, appId } = useAppScope();
  const { data: app } = useApp();
  const source = app ? SOURCES[app.sourceType] : undefined;
  const installations = trpc.github.getInstallations.useQuery(
    { projectId, appId },
    { ...options, enabled: source?.loadsGitHub ?? false },
  );
  const { data } = installations;
  const repo = data?.repoConnection;
  const repositoryFullName = repo?.repositoryFullName || null;

  return {
    projectId,
    appId,
    app,
    installations,
    hasRepository: source?.hasRepository(repositoryFullName, data !== undefined) ?? false,
    defaultBranch: repo?.defaultBranch ?? null,
    repositoryFullName,
    connection:
      repo && data
        ? {
            repositoryId: repo.repositoryId,
            repositoryFullName: repo.repositoryFullName,
            installationId: repo.installationId,
            branch: data.defaultBranch,
          }
        : null,
  };
}
