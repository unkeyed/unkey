"use client";

import { useWorkspaceNavigation } from "@/hooks/use-workspace-navigation";
import {
  PageContainer,
  PageHeader,
  PageHeaderContent,
  PageHeaderTitle,
  SettingsGroup,
  SettingsGroupContent,
  SettingsGroupTitle,
  SettingsGroups,
} from "@unkey/ui";
import { CopyWorkspaceId } from "./copy-workspace-id";
import { GithubConnection } from "./github-connection";
import { UpdateWorkspaceName } from "./update-workspace-name";

export default function SettingsPage() {
  const workspace = useWorkspaceNavigation();

  return (
    <PageContainer>
      <PageHeader className="max-w-[920px]">
        <PageHeaderContent>
          <PageHeaderTitle>General</PageHeaderTitle>
        </PageHeaderContent>
      </PageHeader>
      <SettingsGroups>
        <SettingsGroup>
          <SettingsGroupTitle>Workspace</SettingsGroupTitle>
          <SettingsGroupContent>
            <UpdateWorkspaceName />
            <CopyWorkspaceId workspaceId={workspace.id} />
          </SettingsGroupContent>
        </SettingsGroup>
        <SettingsGroup>
          <SettingsGroupTitle>Integrations</SettingsGroupTitle>
          <SettingsGroupContent>
            <GithubConnection />
          </SettingsGroupContent>
        </SettingsGroup>
      </SettingsGroups>
    </PageContainer>
  );
}
