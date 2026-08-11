"use client";
import { queryClient, trpcClient } from "@/lib/collections/client";
import type { CreateAppRequestSchema } from "@/lib/collections/deploy/apps";
import { buildDefaultSettingsMutations } from "@/lib/collections/deploy/environment-settings";
import { trpc } from "@/lib/trpc/client";
import { CodeBranch, Github } from "@unkey/icons";
import { Button, toast, useStepWizard } from "@unkey/ui";
import { useState } from "react";
import { OnboardingLinks } from "../onboarding-links";
import type { AppDetails } from "./create-app";
import { DeployImageCard } from "./deploy-image-card";

type ChooseSourceStepProps = {
  projectId: string;
  appDetails: AppDetails;
  onAppCreated: (id: string) => void;
  onBeforeNavigate?: () => void;
};

type CreatedApp = {
  id: string;
  sourceKind: CreateAppRequestSchema["source"]["kind"];
};

export const ChooseSourceStep = ({
  projectId,
  appDetails,
  onAppCreated,
  onBeforeNavigate,
}: ChooseSourceStepProps) => {
  const { next } = useStepWizard();
  const utils = trpc.useUtils();
  const [imageMode, setImageMode] = useState(false);
  const [createdApp, setCreatedApp] = useState<CreatedApp | null>(null);
  const [selectedSource, setSelectedSource] = useState<CreatedApp["sourceKind"] | null>(null);

  const { data: availableRegions } = trpc.deploy.environmentSettings.getAvailableRegions.useQuery();
  const createApp = trpc.deploy.app.create.useMutation();

  const ensureApp = async (source: CreateAppRequestSchema["source"]): Promise<string> => {
    if (createdApp) {
      if (createdApp.sourceKind !== source.kind) {
        throw new Error("The app source has already been selected");
      }
      return createdApp.id;
    }

    setSelectedSource(source.kind);
    try {
      const app = await createApp.mutateAsync({
        projectId,
        ...appDetails,
        source,
      });
      const nextCreatedApp = { id: app.id, sourceKind: source.kind };
      setCreatedApp(nextCreatedApp);
      onAppCreated(app.id);

      await Promise.all([
        queryClient.invalidateQueries({ queryKey: ["apps", projectId] }),
        queryClient.invalidateQueries({ queryKey: ["projects"] }),
      ]);

      if (source.kind === "git") {
        try {
          const regions =
            availableRegions ??
            (await utils.deploy.environmentSettings.getAvailableRegions.fetch());
          const environments = await trpcClient.deploy.environment.list.query({ projectId });
          const mutations = environments
            .filter((environment) => environment.appId === app.id)
            .flatMap((environment) => buildDefaultSettingsMutations(environment.id, regions));
          await Promise.all(mutations);
        } catch (error) {
          toast.error("Failed to initialize settings", {
            description: error instanceof Error ? error.message : "An unexpected error occurred",
          });
        }
      }

      return app.id;
    } catch (error) {
      setSelectedSource(null);
      throw error;
    }
  };

  // The install URL state is server-signed and bound to this user/workspace.
  // We can't compute it client-side without a server round-trip, so we mint
  // it lazily when the user clicks Import.
  const prepare = trpc.github.prepareInstallation.useMutation();
  const [isPreparing, setIsPreparing] = useState(false);
  const handleClick = async () => {
    setIsPreparing(true);
    try {
      const appId = await ensureApp({ kind: "git" });
      const github = await utils.github.getInstallations.fetch({ projectId, appId });
      if ((github?.installations?.length ?? 0) > 0) {
        setIsPreparing(false);
        next();
        return;
      }
      const { state } = await prepare.mutateAsync({ projectId, appId });
      onBeforeNavigate?.();
      window.location.href = `https://github.com/apps/${process.env.NEXT_PUBLIC_GITHUB_APP_NAME}/installations/new?state=${encodeURIComponent(state)}`;
    } catch (err) {
      setIsPreparing(false);
      toast.error(err instanceof Error ? err.message : "Failed to connect GitHub");
    }
  };

  return (
    <div className="flex flex-col items-center">
      <div className="flex flex-col gap-3 w-[600px]">
        {imageMode ? null : (
          <div className="border border-grayA-5 rounded-lg flex justify-start items-center gap-4 py-[18px] px-4">
            <div className="size-8 rounded-[10px] grid place-items-center ring-1 ring-grayA-4 shadow-sm shadow-grayA-8/20 dark:shadow-none shrink-0">
              <CodeBranch className="size-[18px] text-gray-12" iconSize="md-medium" />
            </div>
            <div className="flex flex-col gap-3">
              <span className="font-medium text-gray-12 text-[13px] leading-[9px]">
                Connect a repo
              </span>
              <span className="text-gray-10 text-[13px] leading-[9px]">
                Add a repo from your GitHub account
              </span>
            </div>
            <Button
              variant="outline"
              className="ml-auto rounded-lg border-grayA-4 hover:bg-grayA-2 shadow-sm hover:shadow-md transition-all"
              onClick={handleClick}
              loading={isPreparing}
              disabled={selectedSource === "docker" || createApp.isLoading}
            >
              <Github className="size-[18px]! text-gray-12 shrink-0" />
              <span className="text-[13px] text-gray-12 font-medium">Import from GitHub</span>
            </Button>
          </div>
        )}
        <DeployImageCard
          projectId={projectId}
          onCreateApp={(imageReference) => ensureApp({ kind: "docker", imageReference })}
          onBeforeNavigate={onBeforeNavigate}
          expanded={imageMode}
          onExpandedChange={setImageMode}
          disabled={selectedSource === "git" || createApp.isLoading}
        />
      </div>
      <div className="mb-7" />
      <OnboardingLinks />
    </div>
  );
};
