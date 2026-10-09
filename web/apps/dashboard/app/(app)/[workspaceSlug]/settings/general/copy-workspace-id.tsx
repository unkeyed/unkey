"use client";
import { CopyInput, SettingsRow } from "@unkey/ui";

export const CopyWorkspaceId = ({ workspaceId }: { workspaceId: string }) => {
  return (
    <SettingsRow title="Workspace ID" description="An identifier for the workspace.">
      <CopyInput value={workspaceId} aria-label="Workspace ID" />
    </SettingsRow>
  );
};
