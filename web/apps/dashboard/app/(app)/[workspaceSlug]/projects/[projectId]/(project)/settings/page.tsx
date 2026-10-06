"use client";

import { useProject } from "@/hooks/use-project";
import {
  PageContainer,
  PageHeader,
  PageHeaderContent,
  PageHeaderTitle,
  SettingsDangerZone,
  SettingsGroups,
} from "@unkey/ui";
import { DeleteProject } from "./components/delete-project";
import { UpdateProjectSettings } from "./components/update-project-settings";

export default function ProjectSettingsPage() {
  const { project } = useProject();

  return (
    <PageContainer>
      <PageHeader className="max-w-[920px]">
        <PageHeaderContent>
          <PageHeaderTitle>Project Settings</PageHeaderTitle>
        </PageHeaderContent>
      </PageHeader>
      <SettingsGroups>
        {project ? (
          <>
            <UpdateProjectSettings project={project} />
            {project.isDefault ? null : (
              <SettingsDangerZone>
                <DeleteProject project={project} />
              </SettingsDangerZone>
            )}
          </>
        ) : null}
      </SettingsGroups>
    </PageContainer>
  );
}
