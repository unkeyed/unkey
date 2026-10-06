"use client";

import { trpc } from "@/lib/trpc/client";
import { getErrorMessage } from "@/lib/unkey-client";
import { Github, IconChevronRightOutline18 } from "@unkey/icons";
import { match } from "@unkey/match";
import { Button, ItemGroup, Skeleton } from "@unkey/ui";
import { useState } from "react";
import { appNameFromRepo } from "../app-name";
import { useNewAppFlow } from "../flow";
import { useAppLifecycle } from "../use-app-lifecycle";
import { useConnectGithub } from "../use-connect-github";
import type { SetupFieldFocus } from "../wizard-model";
import { PaneRow } from "./pane-row";
import { ConnectedSetup } from "./repository/connected-setup";
import { RepoListPlaceholder, RepoNameList } from "./repository/repo-name-list";
import {
  type Connection,
  type RepoItem,
  repoShortName,
  resolvePickView,
  resolveSetupView,
} from "./repository/repository-view";
import { REPO_TREE_STALE_MS, repoTreeInput } from "./settings/use-repo-tree";

function useInstallation(appId: string | null) {
  const { projectId } = useNewAppFlow();
  return trpc.github.getInstallations.useQuery(
    { projectId, appId: appId ?? "" },
    { enabled: appId !== null },
  );
}

// The link and later branch changes queue under one key, so a branch change
// never lands before the link it changes.
function useLinkRepository() {
  const { projectId, beginSetup } = useNewAppFlow();
  const utils = trpc.useUtils();
  const selectRepository = trpc.github.selectRepository.useMutation();
  return (appId: string, connection: Connection) => {
    const work = beginSetup(appId, connection);
    work.run("link", async () => {
      await selectRepository.mutateAsync({
        projectId,
        appId,
        repositoryId: connection.repositoryId,
        repositoryFullName: connection.repositoryFullName,
        installationId: connection.installationId,
        selectedBranch: connection.branch,
      });
      await Promise.all([
        utils.github.getInstallations.invalidate(),
        utils.github.getRepoTree.invalidate(),
      ]);
    });
    return work;
  };
}

function LoadingRows() {
  return (
    <div className="flex flex-col gap-2">
      <Skeleton className="h-8 w-full rounded-lg" />
      <Skeleton className="h-12 w-full rounded-lg" />
      <Skeleton className="h-12 w-full rounded-lg" />
    </div>
  );
}

function RetryBox({ message, onRetry }: { message: string; onRetry: () => void }) {
  return (
    <div className="flex flex-col items-start gap-3 rounded-lg border border-dashed p-4">
      <p className="text-sm text-gray-11">{message}</p>
      <Button size="sm" variant="outline" onClick={onRetry}>
        Retry
      </Button>
    </div>
  );
}

export function PickRepoPane({ appId }: { appId: string | null }) {
  const { projectId, state, returningFromGithub, dispatch, ensureApp } = useNewAppFlow();
  const { createGitApp, renameApp } = useAppLifecycle(projectId);
  const link = useLinkRepository();
  const utils = trpc.useUtils();
  const github = useConnectGithub();
  const [pendingRepoId, setPendingRepoId] = useState<number | null>(null);
  const [pickError, setPickError] = useState<string | null>(null);

  const { data: context } = trpc.deploy.project.creationContext.useQuery();
  const installed = context?.hasGithubInstallation;
  const reposQuery = trpc.github.listRepositories.useQuery(
    { projectId },
    { enabled: installed === true, refetchOnWindowFocus: false },
  );
  const installationQuery = useInstallation(appId);

  const pick = async (repo: RepoItem) => {
    setPendingRepoId(repo.id);
    setPickError(null);
    const baseName = appNameFromRepo(repo.fullName);
    const existing = state.app;
    const connection: Connection = {
      repositoryId: repo.id,
      repositoryFullName: repo.fullName,
      installationId: repo.installationId,
      branch: repo.defaultBranch,
    };
    void utils.github.getRepositoryTree.prefetch(repoTreeInput(projectId, connection), {
      staleTime: REPO_TREE_STALE_MS,
    });
    try {
      const app = await ensureApp("git", () => createGitApp(baseName));
      if (!app.ok) {
        setPendingRepoId(null);
        setPickError(app.error);
        return;
      }
      const work = link(app.appId, connection);
      work.run("settings", app.applyDefaults);
      if (existing) {
        work.run("rename", () => renameApp(existing.id, baseName));
      }
      dispatch({ type: "go", card: "configure-repo" });
    } catch (error) {
      setPendingRepoId(null);
      setPickError(`Could not create the app. ${getErrorMessage(error)}`);
    }
  };

  const view = resolvePickView({
    installed,
    installation: installationQuery.data,
    installationError: installationQuery.error?.message ?? null,
    repos: reposQuery.data?.repositories,
    reposError: reposQuery.error?.message ?? null,
  });
  const busy = github.connecting || pendingRepoId !== null;
  const error = pickError ?? github.error;

  return match(view)
    .with({ kind: "loading" }, () =>
      returningFromGithub ? (
        <RepoListPlaceholder message="Finalizing GitHub connection…" />
      ) : (
        <LoadingRows />
      ),
    )
    .with({ kind: "not-installed" }, () => (
      <ItemGroup variant="outline" className="overflow-hidden">
        <PaneRow
          icon={<Github />}
          label="Connect GitHub"
          hint="Install the Unkey GitHub app to import a repository"
          disabled={github.connecting}
          trailing={<IconChevronRightOutline18 />}
          onClick={github.connect}
        />
      </ItemGroup>
    ))
    .with({ kind: "error" }, ({ message }) => (
      <RetryBox
        message={message}
        onRetry={() => Promise.all([installationQuery.refetch(), reposQuery.refetch()])}
      />
    ))
    .with({ kind: "pick" }, ({ repos, current }) => (
      <div className="flex min-h-0 flex-col gap-3">
        <RepoNameList repos={repos} pendingRepoId={pendingRepoId} onPick={pick} />
        {error ? <p className="text-sm text-error-11">{error}</p> : null}
        <div className="flex items-center justify-between gap-3">
          <button
            type="button"
            onClick={github.connect}
            disabled={busy}
            className="shrink-0 whitespace-nowrap text-xs text-gray-10 underline decoration-dotted underline-offset-2 hover:text-gray-12"
          >
            Can't find it? Add repositories on GitHub
          </button>
          {current ? (
            <Button
              size="sm"
              variant="ghost"
              className="min-w-0"
              disabled={pendingRepoId !== null}
              onClick={() => dispatch({ type: "go", card: "configure-repo" })}
            >
              <span className="truncate">
                Continue with {repoShortName(current.repositoryFullName)}
              </span>
            </Button>
          ) : null}
        </div>
      </div>
    ))
    .exhaustive();
}

export function ConfigureRepoPane({
  appId,
  focus,
}: {
  appId: string;
  focus: SetupFieldFocus | null;
}) {
  const { projectId, setup, dispatch } = useNewAppFlow();
  const { renameApp } = useAppLifecycle(projectId);
  const link = useLinkRepository();
  const installationQuery = useInstallation(appId);
  const view = resolveSetupView({
    installation: installationQuery.data,
    installationError: installationQuery.error?.message ?? null,
    picked: setup?.appId === appId ? setup.connection : null,
  });

  return match(view)
    .with({ kind: "loading" }, () => <LoadingRows />)
    .with({ kind: "error" }, ({ message }) => (
      <RetryBox message={message} onRetry={() => installationQuery.refetch()} />
    ))
    .with({ kind: "disconnected" }, () => (
      <div className="flex flex-col items-start gap-3 rounded-lg border border-dashed p-4">
        <p className="text-sm text-gray-11">This app has no repository yet.</p>
        <Button
          size="sm"
          variant="outline"
          onClick={() => dispatch({ type: "go", card: "pick-repo" })}
        >
          Choose a repository
        </Button>
      </div>
    ))
    .with({ kind: "connected" }, ({ connection }) => (
      <ConnectedSetup
        key={connection.repositoryFullName}
        projectId={projectId}
        appId={appId}
        connection={connection}
        onBranchChange={(branch) => link(appId, { ...connection, branch })}
        onRename={(name) => renameApp(appId, name)}
        onContinue={() => dispatch({ type: "next" })}
        focusField={focus}
      />
    ))
    .exhaustive();
}
