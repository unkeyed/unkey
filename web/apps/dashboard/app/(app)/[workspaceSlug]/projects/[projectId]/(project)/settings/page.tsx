"use client";

import { useProject } from "@/lib/deploy-data";
import {
  PageContainer,
  PageHeader,
  PageHeaderContent,
  PageHeaderTitle,
  SettingsDangerZone,
  SettingsGroups,
} from "@unkey/ui";
import { useParams } from "next/navigation";
import { DeleteProject } from "./components/delete-project";
import { UpdateProjectSettings } from "./components/update-project-settings";

export default function ProjectSettingsPage() {
  const { projectId } = useParams<{ projectId?: string }>();
  const { data: project } = useProject(projectId);

  return (
    <PageContainer>
      <PageHeader className="max-w-[920px]">
        <PageHeaderContent>
          <PageHeaderTitle>Project settings</PageHeaderTitle>
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
