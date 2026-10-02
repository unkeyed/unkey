"use client";

import { trpc } from "@/lib/trpc/client";
import { Github, IconArrowUpRightOutline12, IconCheckOutline12 } from "@unkey/icons";
import { match } from "@unkey/match";
import {
  Button,
  SettingsRow,
  SettingsRowContent,
  SettingsRowDescription,
  SettingsRowHeader,
  SettingsRowTitle,
  Skeleton,
  toast,
} from "@unkey/ui";
import { cn } from "cn";

type GithubConnectionState =
  | { status: "loading" }
  | { status: "connected" }
  | { status: "disconnected" };

type InstallStatus = Exclude<GithubConnectionState["status"], "loading">;

const INSTALL_CARD: Record<
  InstallStatus,
  { summary: string; indicator: React.ReactNode; border: string }
> = {
  connected: {
    summary: "Installed",
    indicator: <IconCheckOutline12 className="size-2.5 shrink-0 text-success-11" />,
    border: "border-solid",
  },
  disconnected: {
    summary: "Not installed",
    indicator: <span className="size-1.5 shrink-0 rounded-full bg-gray-8" />,
    border: "border-dashed",
  },
};

export function GithubConnection() {
  const { data, isLoading } = trpc.github.hasInstallations.useQuery();
  const configuration = trpc.github.configuration.useQuery();
  const prepareInstall = trpc.github.prepareWorkspaceInstall.useMutation();

  const openInstall = async () => {
    try {
      const { url } = await prepareInstall.mutateAsync();
      window.location.href = url;
    } catch (err) {
      toast.error(err instanceof Error ? err.message : "Failed to start GitHub install");
    }
  };

  const installStatus: InstallStatus = data?.hasInstallation ? "connected" : "disconnected";
  const state: GithubConnectionState = isLoading
    ? { status: "loading" }
    : { status: installStatus };

  return (
    <SettingsRow>
      <SettingsRowHeader>
        <SettingsRowTitle>GitHub</SettingsRowTitle>
        <SettingsRowDescription>
          {configuration.data?.available === false
            ? "GitHub is not configured on this instance. You can deploy a container image instead."
            : "Deploy apps from your GitHub repositories."}
        </SettingsRowDescription>
      </SettingsRowHeader>
      <SettingsRowContent>
        {match(state)
          .with({ status: "loading" }, () => (
            <Skeleton className="h-[58px] max-w-(--setting-w) rounded-lg" />
          ))
          .with({ status: "connected" }, ({ status }) => (
            <GithubAppCard status={status}>
              <Button
                variant="outline"
                loading={prepareInstall.isLoading}
                disabled={!configuration.data?.available}
                onClick={openInstall}
              >
                {configuration.data?.available === false ? "GitHub unavailable" : "Manage"}
                <IconArrowUpRightOutline12 className="size-3! text-gray-11" />
              </Button>
            </GithubAppCard>
          ))
          .with({ status: "disconnected" }, ({ status }) => (
            <GithubAppCard status={status}>
              <Button
                variant="primary"
                loading={prepareInstall.isLoading}
                disabled={!configuration.data?.available}
                onClick={openInstall}
              >
                {configuration.data?.available === false ? "GitHub unavailable" : "Install"}
              </Button>
            </GithubAppCard>
          ))
          .exhaustive()}
      </SettingsRowContent>
    </SettingsRow>
  );
}

function GithubAppCard({ status, children }: { status: InstallStatus; children: React.ReactNode }) {
  const card = INSTALL_CARD[status];
  return (
    <div
      className={cn(
        "flex max-w-(--setting-w) items-center gap-3 rounded-lg border bg-raised p-3",
        card.border,
      )}
    >
      <div className="flex size-8 shrink-0 items-center justify-center rounded-md bg-gray-12 text-gray-1">
        <Github className="size-4" />
      </div>
      <div className="flex min-w-0 flex-1 flex-col gap-1">
        <span className="text-sm font-medium leading-4 text-gray-12">Unkey GitHub App</span>
        <span className="flex items-center gap-1.5 text-xs leading-4 text-gray-11">
          {card.indicator}
          {card.summary}
        </span>
      </div>
      {children}
    </div>
  );
}
