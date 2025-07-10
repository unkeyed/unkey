"use client";

import { CopyButton, SettingCard, toast } from "@unkey/ui";

export const CopyWorkspaceId = ({ workspaceId }: { workspaceId: string }) => {
  return (
    <SettingCard
      title={"Workspace ID"}
      description={"An identifier for the workspace."}
      border="bottom"
      contentWidth="w-full lg:w-[420px] justify-end items-end"
    >
      <div className="flex flex-row justify-end items-center">
        <div className="flex flex-row justify-end min-w-[395px] items-center pl-4 pr-3 w-full h-9 border border-gray-5 hover:border-gray-8 bg-gray-2 dark:bg-black rounded-lg ">
          <pre className="flex-1 text-xs text-left overflow-x-auto">
            <code>{workspaceId}</code>
          </pre>
          <CopyButton
            value={workspaceId}
            variant="ghost"
            size="sm"
            onClick={() => {
              toast.success("Copied to clipboard", {
                description: workspaceId,
              });
            }}
          />
        </div>
      </div>
    </SettingCard>
  );
};
