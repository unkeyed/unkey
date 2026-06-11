"use client";

import { useWorkspaceNavigation } from "@/hooks/use-workspace-navigation";
import { PageBody, PageContainer, PageHeader, PageHeaderContent, PageHeaderTitle } from "@unkey/ui";
import { CopyWorkspaceId } from "./copy-workspace-id";
import { UpdateWorkspaceName } from "./update-workspace-name";

export default function SettingsPage() {
  const workspace = useWorkspaceNavigation();

  return (
    <PageContainer>
      <PageHeader>
        <PageHeaderContent>
          <PageHeaderTitle>General</PageHeaderTitle>
        </PageHeaderContent>
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
