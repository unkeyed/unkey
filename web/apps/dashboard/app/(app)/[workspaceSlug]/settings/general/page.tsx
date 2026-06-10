"use client";

import { CopyableIDButton } from "@/components/navigation/copyable-id-button";
import { useWorkspaceNavigation } from "@/hooks/use-workspace-navigation";
import {
  PageBody,
  PageContainer,
  PageHeader,
  PageHeaderActions,
  PageHeaderContent,
  PageHeaderTitle,
} from "@unkey/ui";
import { CopyWorkspaceId } from "./copy-workspace-id";
import { UpdateWorkspaceName } from "./update-workspace-name";

/**
 * TODO: WorkOS doesn't have workspace images
 */

export default function SettingsPage() {
  const workspace = useWorkspaceNavigation();

  return (
    <PageContainer>
      <PageHeader>
        <PageHeaderContent>
          <PageHeaderTitle>General</PageHeaderTitle>
        </PageHeaderContent>
        <PageHeaderActions>
          <CopyableIDButton value={workspace.id} />
        </PageHeaderActions>
      </PageHeader>
      <PageBody>
        <div className="w-full flex flex-col pt-4">
          <UpdateWorkspaceName />
          {/* <UpdateWorkspaceImage /> */}
          <CopyWorkspaceId workspaceId={workspace.id} />
        </div>
      </PageBody>
    </PageContainer>
  );
}
