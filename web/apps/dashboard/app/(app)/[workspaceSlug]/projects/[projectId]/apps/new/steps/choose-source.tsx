"use client";
import { Github } from "@unkey/icons";
import { Button, toast, useStepWizard } from "@unkey/ui";
import { IconCodeBranchOutline18 } from "nucleo-ui-outline-18";
import { useState } from "react";
import { trpc } from "@/lib/trpc/client";
import { OnboardingLinks } from "../onboarding-links";
import { DeployImageCard } from "./deploy-image-card";

type ChooseSourceStepProps = {
  projectId: string;
  appId: string;
  onBeforeNavigate?: () => void;
};

export const ChooseSourceStep = ({ projectId, appId, onBeforeNavigate }: ChooseSourceStepProps) => {
  const { next } = useStepWizard();
  const utils = trpc.useUtils();
  const [imageMode, setImageMode] = useState(false);

  // The install URL state is server-signed and bound to this user/workspace.
  // We can't compute it client-side without a server round-trip, so we mint
  // it lazily when the user clicks Import.
  const prepare = trpc.github.prepareInstallation.useMutation();
  const [isPreparing, setIsPreparing] = useState(false);
  const handleClick = async () => {
    setIsPreparing(true);
    try {
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
              <IconCodeBranchOutline18 className="text-gray-12" />
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
            >
              <Github className="size-[18px]! text-gray-12 shrink-0" />
              <span className="text-[13px] text-gray-12 font-medium">Import from GitHub</span>
            </Button>
          </div>
        )}
        <DeployImageCard
          projectId={projectId}
          appId={appId}
          onBeforeNavigate={onBeforeNavigate}
          expanded={imageMode}
          onExpandedChange={setImageMode}
        />
      </div>
      <div className="mb-7" />
      <OnboardingLinks />
    </div>
  );
};
