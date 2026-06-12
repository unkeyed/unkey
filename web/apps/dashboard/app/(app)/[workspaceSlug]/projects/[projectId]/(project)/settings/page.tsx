"use client";

import {
  PageBody,
  PageContainer,
  PageHeader,
  PageHeaderContent,
  PageHeaderDescription,
  PageHeaderTitle,
  SettingsDangerZone,
} from "@unkey/ui";
import { DeleteProject } from "./components/delete-project";
import { UpdateProjectSettings } from "./components/update-project-settings";

export default function ProjectSettingsPage() {
  return (
    <PageContainer>
      <PageHeader>
        <PageHeaderContent>
          <PageHeaderTitle>Settings</PageHeaderTitle>
          <PageHeaderDescription>Manage your project name.</PageHeaderDescription>
        </PageHeaderContent>
      </PageHeader>
      <PageBody>
        <UpdateProjectSettings />
        <SettingsDangerZone>
          <DeleteProject />
        </SettingsDangerZone>
      </PageBody>
    </PageContainer>
  );
}
