"use client";

import { trpc } from "@/lib/trpc/client";
import { getErrorMessage } from "@/lib/unkey-client";
import { Github, IconChevronRightOutline18 } from "@unkey/icons";
import { match } from "@unkey/match";
import { Button, ItemGroup, Skeleton, toast } from "@unkey/ui";
import { useEffect, useState } from "react";
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
  resolvePickView,
  resolveSetupView,
} from "./repository/repository-view";
import { useAppSettings } from "./settings";

function useInstallation(appId: string | null) {
  const { projectId } = useNewAppFlow();
  return trpc.github.getInstallations.useQuery(
    { projectId, appId: appId ?? "" },
    { enabled: appId !== null },
  );
}

function useLinkRepository() {
  const { projectId } = useNewAppFlow();
  const utils = trpc.useUtils();
  const selectRepository = trpc.github.selectRepository.useMutation();
  const link = async (appId: string, repo: Omit<Connection, "branch">, branch: string) => {
    await selectRepository.mutateAsync({
      projectId,
      appId,
      repositoryId: repo.repositoryId,
      repositoryFullName: repo.repositoryFullName,
      installationId: repo.installationId,
      selectedBranch: branch,
    });
    await Promise.all([
      utils.github.getInstallations.invalidate(),
      utils.github.getRepoTree.invalidate(),
    ]);
  };
  return { link, linking: selectRepository.isLoading };
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
  const { link } = useLinkRepository();
  const github = useConnectGithub();
  const [pendingRepoId, setPendingRepoId] = useState<number | null>(null);
  const [linkedAppId, setLinkedAppId] = useState<string | null>(null);
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
    try {
      const app = await ensureApp("git", () => createGitApp(baseName));
      if (!app.ok) {
        setPendingRepoId(null);
        setPickError(app.error);
        return;
      }
      if (existing) {
        await renameApp(existing.id, baseName);
      }
      await link(
        app.appId,
        {
          repositoryId: repo.id,
          repositoryFullName: repo.fullName,
          installationId: repo.installationId,
        },
        repo.defaultBranch,
      );
      setLinkedAppId(app.appId);
    } catch (error) {
      setPendingRepoId(null);
      setPickError(`Could not connect the repository. ${getErrorMessage(error)}`);
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
            className="text-xs text-gray-10 underline decoration-dotted underline-offset-2 hover:text-gray-12"
          >
            Can't find it? Add repositories on GitHub
          </button>
          {current ? (
            <Button
              size="sm"
              variant="ghost"
              disabled={pendingRepoId !== null}
              onClick={() => dispatch({ type: "go", card: "configure-repo" })}
            >
              Keep {current.repositoryFullName}
            </Button>
          ) : null}
        </div>
        {linkedAppId ? <AdvanceWhenSettingsReady appId={linkedAppId} /> : null}
      </div>
    ))
    .exhaustive();
}

// The picked row keeps its spinner until the setup card has its settings, so
// the column never slides onto a skeleton.
function AdvanceWhenSettingsReady({ appId }: { appId: string }) {
  const { projectId, dispatch } = useNewAppFlow();
  const ready = useAppSettings(projectId, appId).status === "ready";
  useEffect(() => {
    if (ready) {
      dispatch({ type: "go", card: "configure-repo" });
    }
  }, [ready, dispatch]);
  return null;
}

export function ConfigureRepoPane({
  appId,
  focus,
}: {
  appId: string;
  focus: SetupFieldFocus | null;
}) {
  const { projectId, dispatch } = useNewAppFlow();
  const { renameApp } = useAppLifecycle(projectId);
  const { link, linking } = useLinkRepository();
  const installationQuery = useInstallation(appId);
  const view = resolveSetupView({
    installation: installationQuery.data,
    installationError: installationQuery.error?.message ?? null,
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
        branchDisabled={linking}
        onBranchChange={(branch) =>
          link(appId, connection, branch).catch((error) =>
            toast.error("Could not change the branch", { description: getErrorMessage(error) }),
          )
        }
        onRename={(name) => renameApp(appId, name)}
        onContinue={() => dispatch({ type: "next" })}
        focusField={focus}
      />
    ))
    .exhaustive();
}
