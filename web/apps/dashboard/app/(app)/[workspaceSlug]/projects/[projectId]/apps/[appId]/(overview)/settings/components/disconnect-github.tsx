"use client";

import { collection } from "@/lib/collections";
import { trpc } from "@/lib/trpc/client";
import { match } from "@unkey/match";
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
  SettingsZoneRow,
  toast,
} from "@unkey/ui";
import { useState } from "react";
import { useAppId, useProjectData } from "../../data-provider";
import { useApp } from "../hooks/use-build-source";

export function DisconnectGitHub() {
  const { projectId } = useProjectData();
  const appId = useAppId();
  const utils = trpc.useUtils();
  const [isConfirmOpen, setIsConfirmOpen] = useState(false);
  const [pendingRepo, setPendingRepo] = useState<string | null>(null);
  const { app } = useApp();
  const shouldLoadGitHub = app
    ? match(app.sourceType)
        .with("git", () => true)
        .with("oci", () => false)
        .with("unknown", () => Boolean(app.repositoryFullName))
        .exhaustive()
    : false;

  const { data } = trpc.github.getInstallations.useQuery(
    { projectId, appId },
    { enabled: shouldLoadGitHub },
  );

  const repositoryFullName = data?.repoConnection?.repositoryFullName;

  const disconnectRepoMutation = trpc.github.disconnectRepo.useMutation({
    onSuccess: async () => {
      toast.success("Repository disconnected");
      await utils.github.getInstallations.invalidate();
      await utils.github.getRepoTree.invalidate();
      await collection.apps.utils.refetch();
    },
    onError: (error) => {
      toast.error(error.message);
    },
  });

  return (
    <>
      {shouldLoadGitHub && repositoryFullName ? (
        <SettingsZoneRow
          title="Disconnect repository"
          description="Pushes to this repository stop triggering deployments."
          action={{
            label: "Disconnect repository",
            onClick: () => {
              setPendingRepo(repositoryFullName);
              setIsConfirmOpen(true);
            },
            loading: disconnectRepoMutation.isLoading,
            disabled: disconnectRepoMutation.isLoading,
          }}
        />
      ) : null}

      <AlertDialog open={isConfirmOpen} onOpenChange={setIsConfirmOpen}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>Disconnect {pendingRepo}?</AlertDialogTitle>
            <AlertDialogDescription>
              Unkey will stop building pushes to this repository. Running deployments stay live.
              Builds awaiting approval stay blocked until you connect a repository again.
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>Cancel</AlertDialogCancel>
            <AlertDialogAction
              color="danger"
              onClick={() => disconnectRepoMutation.mutate({ appId })}
            >
              Disconnect repository
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </>
  );
}
