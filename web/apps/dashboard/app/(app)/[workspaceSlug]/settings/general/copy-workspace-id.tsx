"use client";
import { CopyButton, SettingCard } from "@unkey/ui";

export const CopyWorkspaceId = ({ workspaceId }: { workspaceId: string }) => {
  return (
    <SettingCard
      title={"Workspace ID"}
      description={"An identifier for the workspace."}
      border="bottom"
      contentWidth="w-full lg:w-[420px] justify-end items-end"
    >
      <div className="flex flex-row justify-end items-center">
        <div
          className={
            "flex flex-row justify-between min-w-[395px] pl-2 pr-2 py-2 bg-gray-2 dark:bg-black border rounded-lg border-grayA-5"
          }
        >
          <div className="text-sm leading-5 text-gray-11">{workspaceId}</div>
          <CopyButton value={workspaceId} variant="ghost" toastMessage={workspaceId} />
        </div>
      </div>
    </SettingCard>
  );
};
