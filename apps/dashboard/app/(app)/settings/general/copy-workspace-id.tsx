"use client";

import { toast } from "@/components/ui/toaster";
import { Clone } from "@unkey/icons";
import { Input, SettingCard } from "@unkey/ui";

export const CopyWorkspaceId = ({ workspaceId }: { workspaceId: string }) => {
  return (
    <SettingCard
      title={
        <div className="flex items-center justify-start gap-2.5">
          <span className="text-sm font-medium text-accent-12">Workspace ID</span>
        </div>
      }
      description={
        <div className="font-normal text-[13px] max-w-[380px]">
          An identifier for the workspace.
        </div>
      }
      border="bottom"
      contentWidth="w-full lg:w-[320px] h-16"
    >
      <div className="flex flex-col w-full h-full gap-x-2 pt-2">
        <Input
          className="w-full lg:w-[315px] focus:ring-0 focus:ring-offset-0"
          readOnly
          defaultValue={workspaceId}
          placeholder="Workspace ID"
          rightIcon={
            <button
              type="button"
              onClick={() => {
                navigator.clipboard.writeText(workspaceId);
                toast.success("Copied to clipboard", {
                  description: workspaceId,
                });
              }}
            >
              <Clone size="md-regular" />
            </button>
          }
        />
      </div>
    </SettingCard>
  );
};
