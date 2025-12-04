"use client";
import { KeySecretSection } from "@/app/(app)/[workspaceSlug]/apis/[apiId]/_components/create-key/components/key-secret-section";
import { useWorkspaceNavigation } from "@/hooks/use-workspace-navigation";
import { trpc } from "@/lib/trpc/client";
import { TriangleWarning } from "@unkey/icons";
import { ConfirmPopover } from "@unkey/ui";
import { useRouter, useSearchParams } from "next/navigation";
import { useRef } from "react";
import { API_ID_PARAM, KEY_PARAM } from "../constants";

type OnboardingSuccessStepProps = {
  isConfirmOpen: boolean;
  setIsConfirmOpen: (open: boolean) => void;
};

export const OnboardingSuccessStep = ({
  isConfirmOpen,
  setIsConfirmOpen,
}: OnboardingSuccessStepProps) => {
  const router = useRouter();
  const anchorRef = useRef<HTMLDivElement>(null);
  const searchParams = useSearchParams();
  const workspace = useWorkspaceNavigation();
  const apiId = searchParams?.get(API_ID_PARAM);
  const key = searchParams?.get(KEY_PARAM);
  const utils = trpc.useUtils();

  if (!apiId || !key) {
    return (
      <div className="rounded-xl bg-grayA-3 dark:bg-black border border-grayA-3 flex items-center gap-4 px-[22px] py-6">
        <div className="bg-gray-1 size-8 rounded-full flex items-center justify-center flex-shrink-0">
          <TriangleWarning className="text-error-9" iconSize="xl-medium" />
        </div>
        <div className="text-gray-12 text-[13px] leading-6">
          <span className="font-medium">Error:</span> Missing API or key information. Please go back
          and create your API key again to continue with the setup process.
        </div>
      </div>
    );
  }

  return (
    <div>
      <span className="text-gray-11 text-[13px] leading-6" ref={anchorRef}>
        Run this command to verify your new API key against the API ID. This ensures your key is
        ready for authenticated requests.
      </span>
      <KeySecretSection
        keyValue={key}
        apiId={apiId}
        className="mt-6"
        secretKeyClassName="bg-gray-2"
        codeClassName="bg-gray-2"
      />
      <ConfirmPopover
        isOpen={isConfirmOpen}
        onOpenChange={setIsConfirmOpen}
        onConfirm={() => {
          setIsConfirmOpen(false);
          utils.workspace.getCurrent.invalidate();
          router.push(`/${workspace.slug}/apis`);
        }}
        triggerRef={anchorRef}
        title="You won't see this secret key again!"
        description="Make sure to copy your secret key before closing. It cannot be retrieved later."
        confirmButtonText="Close anyway"
        cancelButtonText="Dismiss"
        variant="warning"
        popoverProps={{
          side: "right",
          align: "end",
          sideOffset: 5,
          alignOffset: 30,
          onOpenAutoFocus: (e) => e.preventDefault(),
        }}
      />
    </div>
  );
};
