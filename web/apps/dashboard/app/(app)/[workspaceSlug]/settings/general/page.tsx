"use client";

import { PageBody, PageContainer, PageHeader, PageHeaderContent, PageHeaderTitle } from "@unkey/ui";
import { useWorkspaceNavigation } from "@/hooks/use-workspace-navigation";
import { CopyWorkspaceId } from "./copy-workspace-id";
import { GithubConnection } from "./github-connection";
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
        <div className="w-full flex flex-col">
          <UpdateWorkspaceName />
          {/* <UpdateWorkspaceImage /> */}
          <CopyWorkspaceId workspaceId={workspace.id} />
        </div>
        <div className="w-full flex flex-col mt-6">
          <GithubConnection />
        </div>
      </PageBody>
    </PageContainer>
  );
}
