"use client";

import { LoadError } from "@/components/load-error";
import { collection } from "@/lib/collections";
import { refetchProjectApps } from "@/lib/collections/deploy/apps";
import { trpc } from "@/lib/trpc/client";
import { useMutation } from "@tanstack/react-query";
import { Github, IconChevronRightOutline18 } from "@unkey/icons";
import { match } from "@unkey/match";
import { Button, ItemGroup } from "@unkey/ui";
import { appNameFromRepo, hasGeneratedName, isPlaceholderApp } from "../app-name";
import { NEW_APP_MUTATION_KEY, useNewAppFlow } from "../flow";
import { newAppErrorMessage, useAppLifecycle } from "../use-app-lifecycle";
import { CONNECT_GITHUB, useConnectGithub } from "../use-connect-github";
import { PaneSubmit } from "./pane-actions";
import { PaneRow } from "./pane-row";
import { RepoNameList } from "./repository/repo-name-list";
import { type RepoItem, resolvePickView } from "./repository/repository-view";

const reposLoadErrorTitle = "Could not load your repositories";

const REPOS_STALE_MS = 30_000;

function usePickRepository(appId: string | null) {
  const { projectId, dispatch, ensureApp } = useNewAppFlow();
  const { createGitApp, renameApp } = useAppLifecycle(projectId);
  const utils = trpc.useUtils();
  const selectRepository = trpc.github.selectRepository.useMutation();

  const relink = async (id: string, repo: RepoItem, baseName: string) => {
    const linked = collection.apps.get(id);
    await selectRepository.mutateAsync({
      projectId,
      appId: id,
      repositoryId: repo.id,
      repositoryFullName: repo.fullName,
      installationId: repo.installationId,
      selectedBranch: repo.defaultBranch,
    });
    await Promise.all([
      linked && (isPlaceholderApp(linked) || hasGeneratedName(linked))
        ? renameApp(id, baseName)
        : refetchProjectApps(projectId),
      utils.github.getInstallations.invalidate({ projectId, appId: id }),
    ]);
    return id;
  };

  return useMutation({
    mutationKey: NEW_APP_MUTATION_KEY,
    mutationFn: async (repo: RepoItem) => {
      const baseName = appNameFromRepo(repo.fullName);
      const id =
        appId === null
          ? await ensureApp("git", () =>
              createGitApp(baseName, {
                repository: repo.fullName,
                defaultBranch: repo.defaultBranch,
              }),
            )
          : await relink(appId, repo, baseName);
      void utils.github.getRepoTree.invalidate({ projectId, appId: id });
    },
    onSuccess: () => dispatch({ type: "go", step: "settings" }),
  });
}

export function PickRepoPane({ appId }: { appId: string | null }) {
  const { projectId, locked, dispatch } = useNewAppFlow();
  const github = useConnectGithub();
  const pick = usePickRepository(appId);
  const utils = trpc.useUtils();

  const contextQuery = trpc.deploy.project.creationContext.useQuery();
  const context = contextQuery.data;
  const reposQuery = trpc.github.listRepositories.useQuery(
    { projectId },
    {
      enabled: context?.hasGithubInstallation === true,
      refetchOnWindowFocus: false,
      staleTime: REPOS_STALE_MS,
    },
  );
  const installationQuery = trpc.github.getInstallations.useQuery(
    { projectId, appId: appId ?? "" },
    { enabled: appId !== null },
  );

  const view = resolvePickView({
    installed: context?.hasGithubInstallation,
    contextError: contextQuery.error?.message ?? null,
    installation: installationQuery.data,
    installationError: installationQuery.error?.message ?? null,
    repos: reposQuery.data?.repositories,
    reposError: reposQuery.error?.message ?? null,
  });
  const pending = pick.isLoading
    ? {
        repoId: pick.variables?.id ?? null,
        label: appId === null ? "Creating your app…" : "Connecting repository…",
      }
    : null;
  const pickError = pick.error
    ? `Could not connect the repository. ${newAppErrorMessage(pick.error)}`
    : null;
  const error = pickError ?? github.error;
  const toConfigure = () => dispatch({ type: "go", step: "settings" });

  const pickList = (repos: RepoItem[] | null, linkedRepoId: number | null) => (
    <div className="flex min-h-0 flex-col gap-3">
      <RepoNameList
        repos={repos}
        pending={pending}
        linkedRepoId={linkedRepoId}
        disabled={locked}
        onPick={(repo) => {
          if (pick.isLoading) {
            return;
          }
          if (repo.id === linkedRepoId) {
            toConfigure();
          } else {
            pick.mutate(repo);
          }
        }}
      />
      {error ? (
        <p role="alert" className="text-sm text-error-11">
          {error}
        </p>
      ) : null}
      <Button
        variant="ghost"
        size="sm"
        className="self-start text-gray-11"
        onClick={github.connect}
        disabled={locked}
      >
        Can't find it? Add repositories on GitHub
      </Button>
      {linkedRepoId !== null ? (
        <PaneSubmit loading={false} disabled={locked} onClick={toConfigure} />
      ) : null}
    </div>
  );

  return match(view)
    .with({ kind: "loading" }, () => pickList(null, null))
    .with({ kind: "not-installed" }, () => (
      <>
        <ItemGroup variant="outline" className="overflow-hidden">
          <PaneRow
            icon={<Github />}
            label={CONNECT_GITHUB.title}
            hint={CONNECT_GITHUB.hint}
            disabled={locked}
            trailing={<IconChevronRightOutline18 />}
            onClick={github.connect}
          />
        </ItemGroup>
        {github.error ? (
          <p role="alert" className="text-sm text-error-11">
            {github.error}
          </p>
        ) : null}
      </>
    ))
    .with({ kind: "error" }, () => (
      <LoadError
        title={reposLoadErrorTitle}
        onRetry={() =>
          Promise.all([
            utils.deploy.project.creationContext.invalidate(),
            utils.github.getInstallations.invalidate({ projectId }),
            utils.github.listRepositories.invalidate({ projectId }),
          ])
        }
      />
    ))
    .with({ kind: "pick" }, ({ repos, linkedRepoId }) => pickList(repos, linkedRepoId))
    .exhaustive();
}
