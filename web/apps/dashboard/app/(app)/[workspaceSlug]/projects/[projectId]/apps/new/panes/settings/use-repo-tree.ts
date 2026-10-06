"use client";

import type { RepoTreeEntry } from "@/app/(app)/[workspaceSlug]/projects/_components/repo-tree";
import { trpc } from "@/lib/trpc/client";

export type RepoRef = { installationId: number; repositoryFullName: string; branch: string };

export type RepoTree = { entries: RepoTreeEntry[]; loading: boolean };

export const REPO_TREE_STALE_MS = 5 * 60 * 1000;

export function repoTreeInput(projectId: string, repo: RepoRef) {
  return {
    projectId,
    installationId: repo.installationId,
    repositoryFullName: repo.repositoryFullName,
    branch: repo.branch,
  };
}

export function useRepoTree(projectId: string, repo: RepoRef): RepoTree {
  const { data, isLoading } = trpc.github.getRepositoryTree.useQuery(
    repoTreeInput(projectId, repo),
    { staleTime: REPO_TREE_STALE_MS, refetchOnWindowFocus: false },
  );
  return { entries: data?.tree ?? [], loading: isLoading };
}
