"use client";
import {
  CopyButton,
  SettingsRow,
  SettingsRowContent,
  SettingsRowDescription,
  SettingsRowHeader,
  SettingsRowTitle,
} from "@unkey/ui";

export const CopyWorkspaceId = ({ workspaceId }: { workspaceId: string }) => {
  return (
    <SettingsRow>
      <SettingsRowHeader>
        <SettingsRowTitle>Workspace ID</SettingsRowTitle>
        <SettingsRowDescription>An identifier for the workspace.</SettingsRowDescription>
      </SettingsRowHeader>
      <SettingsRowContent>
        <div className="flex max-w-(--setting-w) items-center justify-between rounded-lg border bg-raised px-2 py-2">
          <div className="text-sm leading-5 text-gray-11">{workspaceId}</div>
          <CopyButton
            value={workspaceId}
            variant="ghost"
            toastMessage={workspaceId}
            className="shrink-0"
          />
        </div>
      </SettingsRowContent>
    </SettingsRow>
  );
};
