"use client";

import { useProjectData } from "@/app/(app)/[workspaceSlug]/projects/[projectId]/apps/[appId]/(overview)/data-provider";
import { useBuildSource } from "@/app/(app)/[workspaceSlug]/projects/[projectId]/apps/_components/settings/hooks/use-build-source";
import { useDeployActionGate } from "@/app/(app)/[workspaceSlug]/projects/_components/hooks/use-deploy-action-gate";
import { useWorkspaceNavigation } from "@/hooks/use-workspace-navigation";
import { collection } from "@/lib/collections";
import { refetchProjectApps } from "@/lib/collections/deploy/apps";
import {
  defaultSettingsSettled,
  useDefaultSettingsSettled,
} from "@/lib/collections/deploy/environment-settings";
import { routes } from "@/lib/navigation/routes";
import { slugify } from "@/lib/slugify";
import { trpc } from "@/lib/trpc/client";
import { getUnkeyClient } from "@/lib/unkey-client";
import { useMutation } from "@tanstack/react-query";
import { FormInput, toast } from "@unkey/ui";
import { useRouter } from "next/navigation";
import { useState } from "react";
import { APP_NAME_TAKEN, appNameError } from "../app-name";
import { firstDeployEnvironment } from "../deploy-target";
import { NEW_APP_MUTATION_KEY, useNewAppFlow } from "../flow";
import { AppNameTakenError, newAppErrorMessage, useAppLifecycle } from "../use-app-lifecycle";
import type { SourceKind } from "../wizard-model";
import { BranchField } from "./branch-field";
import { ConfigBlocks, ConfigBlocksSkeleton } from "./config-blocks";
import { ConfigureSkeleton } from "./configure-skeleton";
import { PaneSubmit } from "./pane-actions";

type DeployInput = { environment: string; name: string; branch: string | null };

export function ConfigurePane({ source }: { source: SourceKind }) {
  const { app, installations } = useBuildSource();
  if (!app || installations.isInitialLoading) {
    return <ConfigureSkeleton source={source} />;
  }
  return <ConfigureCard source={source} savedName={app.name} />;
}

function ConfigureCard({ source, savedName }: { source: SourceKind; savedName: string }) {
  const { projectId } = useNewAppFlow();
  const { appId } = useBuildSource();
  const { environments } = useProjectData();
  const defaultsSettled = useDefaultSettingsSettled(appId);
  const { gated, openPaywall, planGate } = useDeployActionGate();
  const { takenSlugs } = useAppLifecycle(projectId);
  const [name, setName] = useState(savedName);
  const [branch, setBranch] = useState<string | null>(null);
  const [checkName, setCheckName] = useState(false);
  const [takenName, setTakenName] = useState<string | null>(null);
  const deploy = useDeploy({
    onNameTaken: (taken) => {
      setTakenName(taken);
      setCheckName(true);
    },
  });
  const target = firstDeployEnvironment(source, environments);
  const deploying = deploy.isLoading || deploy.isSuccess;
  const slug = slugify(name);
  const nameError =
    (takenName === slug ? APP_NAME_TAKEN : null) ?? appNameError(name, takenSlugs(appId));

  return (
    <div className="flex flex-col gap-6">
      <div className="grid grid-cols-1 gap-4 @lg:grid-cols-2">
        <FormInput
          data-1p-ignore
          label="Name"
          value={name}
          error={checkName ? (nameError ?? undefined) : undefined}
          onChange={(e) => setName(e.target.value)}
          onBlur={() => setName((current) => slugify(current) || current)}
        />
        <BranchField value={branch} onChange={setBranch} />
      </div>
      {defaultsSettled ? <ConfigBlocks source={source} /> : <ConfigBlocksSkeleton />}
      <PaneSubmit
        loading={deploying}
        disabled={target === null || deploying}
        notice={
          target === null ? (
            <span className="text-xs text-gray-11">
              This app has no production environment to deploy to
            </span>
          ) : null
        }
        onClick={() => {
          if (nameError !== null) {
            setCheckName(true);
            return;
          }
          if (gated) {
            openPaywall();
          } else if (target !== null) {
            deploy.mutate({ environment: target, name: slug, branch });
          }
        }}
      >
        Deploy
      </PaneSubmit>
      {planGate}
    </div>
  );
}

function useDeploy({ onNameTaken }: { onNameTaken: (slug: string) => void }) {
  const { projectId, leave } = useNewAppFlow();
  const { appId, connection } = useBuildSource();
  const { setAppName } = useAppLifecycle(projectId);
  const router = useRouter();
  const workspace = useWorkspaceNavigation();
  const selectRepository = trpc.github.selectRepository.useMutation();
  const deploy = useMutation({
    mutationKey: NEW_APP_MUTATION_KEY,
    mutationFn: async ({ environment, name, branch }: DeployInput) => {
      await defaultSettingsSettled(appId);
      const switchBranch = connection && branch !== null && branch !== connection.branch;
      await Promise.all([
        setAppName(appId, name),
        switchBranch
          ? selectRepository.mutateAsync({
              projectId,
              appId,
              repositoryId: connection.repositoryId,
              repositoryFullName: connection.repositoryFullName,
              installationId: connection.installationId,
              selectedBranch: branch,
            })
          : undefined,
      ]);
      const response = await getUnkeyClient().deployments.createDeploymentV3({
        project: projectId,
        app: appId,
        environment,
      });
      collection.deployments.utils.refetch().catch(() => undefined);
      await refetchProjectApps(projectId).catch(() => undefined);
      return response.data.deploymentId;
    },
    onError: (error, { name }) => {
      if (error instanceof AppNameTakenError) {
        onNameTaken(name);
        return;
      }
      toast.error("Could not start the deployment", { description: newAppErrorMessage(error) });
    },
  });
  return {
    isLoading: deploy.isLoading,
    isSuccess: deploy.isSuccess,
    mutate: (input: DeployInput) =>
      deploy.mutate(input, {
        onSuccess: (deploymentId) => {
          leave();
          router.replace(
            routes.projects.apps.deployment({
              workspaceSlug: workspace.slug,
              projectId,
              appId,
              deploymentId,
            }),
          );
        },
      }),
  };
}
