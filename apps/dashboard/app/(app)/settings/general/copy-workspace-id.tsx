"use client";

import { toast } from "@/components/ui/toaster";
import { Clone } from "@unkey/icons";
import { Input, SettingCard } from "@unkey/ui";

export const CopyWorkspaceId = ({ workspaceId }: { workspaceId: string }) => {
  return (
    <SettingCard
      title={"Workspace ID"}
      description={"An identifier for the workspace."}
      border="bottom"
      contentWidth="w-full lg:w-[420px] justify-end items-end"
    >
      <div className="flex flex-row justify-end items-center">
        <Input
          className="min-w-[25.1rem] self-end"
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
