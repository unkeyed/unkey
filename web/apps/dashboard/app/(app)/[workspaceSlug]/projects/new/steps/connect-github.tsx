"use client";
import { trpc } from "@/lib/trpc/client";
import { Github, Layers3 } from "@unkey/icons";
import { Button, toast } from "@unkey/ui";
import { useState } from "react";
import { OnboardingLinks } from "../onboarding-links";

type ConnectGithubStepProps = {
  projectId: string;
  onBeforeNavigate?: () => void;
};

export const ConnectGithubStep = ({ projectId, onBeforeNavigate }: ConnectGithubStepProps) => {
  // The install URL state is server-signed and bound to this user/workspace.
  // We can't compute it client-side without a server round-trip, so we mint
  // it lazily when the user clicks Import.
  const prepare = trpc.github.prepareInstallation.useMutation();
  const [isPreparing, setIsPreparing] = useState(false);
  const handleClick = async (e: React.MouseEvent<HTMLAnchorElement>) => {
    e.preventDefault();
    if (isPreparing) {
      return;
    }
    setIsPreparing(true);
    try {
      const { state } = await prepare.mutateAsync({ projectId });
      onBeforeNavigate?.();
      window.location.href = `https://github.com/apps/${process.env.NEXT_PUBLIC_GITHUB_APP_NAME}/installations/new?state=${encodeURIComponent(state)}`;
    } catch (err) {
      setIsPreparing(false);
      toast.error(err instanceof Error ? err.message : "Failed to start GitHub install");
    }
  };

  return (
    <div className="flex flex-col items-center">
      <div className="border border-grayA-5 rounded-[14px] flex justify-start items-center gap-4 py-[18px] px-4 min-w-[600px]">
        <div className="size-8 rounded-[10px] bg-gray-12 grid place-items-center">
          <Layers3 className="size-[18px] text-gray-1" iconSize="md-medium" />
        </div>
        <div className="flex flex-col gap-3">
          <span className="font-medium text-gray-12 text-[13px] leading-[9px]">Import project</span>
          <span className="text-gray-10 text-[13px] leading-[9px]">
            Add a repo from your GitHub account
          </span>
        </div>
        <Button
          variant="outline"
          className="ml-auto rounded-lg border-grayA-4 hover:bg-grayA-2 shadow-sm hover:shadow-md transition-all"
          onClick={handleClick}
        >
          <Github className="size-[18px]! text-gray-12 shrink-0" />
          <span className="text-[13px] text-gray-12 font-medium">Import from GitHub</span>
        </Button>
      </div>
      <div className="mb-7" />
      <OnboardingLinks />
    </div>
  );
};
