"use client";

import { useProjectData } from "@/app/(app)/[workspaceSlug]/projects/[projectId]/apps/[appId]/(overview)/data-provider";
import {
  PageBody,
  PageContainer,
  PageHeader,
  PageHeaderContent,
  PageHeaderTitle,
  SettingsDangerZone,
} from "@unkey/ui";
import { DeleteProject } from "./components/delete-project";
import { UpdateProjectSettings } from "./components/update-project-settings";

export default function ProjectSettingsPage() {
  const { project } = useProjectData();

  return (
    <PageContainer>
      <PageHeader>
        <PageHeaderContent>
          <PageHeaderTitle>Project Settings</PageHeaderTitle>
        </PageHeaderContent>
      </PageHeader>
      <PageBody>
        <UpdateProjectSettings />
        {project?.isDefault ? null : (
          <SettingsDangerZone>
            <DeleteProject />
          </SettingsDangerZone>
        )}
      </PageBody>
    </PageContainer>
  );
}
