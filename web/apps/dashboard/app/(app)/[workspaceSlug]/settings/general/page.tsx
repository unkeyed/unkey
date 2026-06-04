"use client";

import { CopyableIDButton } from "@/components/navigation/copyable-id-button";
import { PageChrome } from "@/components/page-header/page-chrome";
import { useWorkspaceNavigation } from "@/hooks/use-workspace-navigation";
import { PageHeader, PageHeaderActions, PageHeaderContent, PageHeaderTitle } from "@unkey/ui";
import { WorkspaceNavbar } from "../workspace-navbar";
import { CopyWorkspaceId } from "./copy-workspace-id";
import { UpdateWorkspaceName } from "./update-workspace-name";

/**
 * TODO: WorkOS doesn't have workspace images
 */

export default function SettingsPage() {
  const workspace = useWorkspaceNavigation();

  return (
    <PageChrome
      legacyHeader={<WorkspaceNavbar activePage={{ href: "general", text: "General" }} />}
      header={
        <PageHeader>
          <PageHeaderContent>
            <PageHeaderTitle>General</PageHeaderTitle>
          </PageHeaderContent>
          <PageHeaderActions>
            <CopyableIDButton value={workspace.id} />
          </PageHeaderActions>
        </PageHeader>
      }
    >
      <div className="w-full flex flex-col pt-4">
        <UpdateWorkspaceName />
        {/* <UpdateWorkspaceImage /> */}
        <CopyWorkspaceId workspaceId={workspace.id} />
      </div>
    </PageChrome>
  );
}
