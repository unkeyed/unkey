"use client";
import {
  CopyInput,
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
        <CopyInput value={workspaceId} aria-label="Workspace ID" className="max-w-(--setting-w)" />
      </SettingsRowContent>
    </SettingsRow>
  );
};
