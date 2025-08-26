"use client";

import { ProjectNavigation } from "../navigations/project-navigation";
import { ProjectSubNavigation } from "../navigations/project-sub-navigation";

export default function ProjectSettings({
  params: { projectId },
}: {
  params: { projectId: string };
}) {
  return (
    <div>
      <ProjectNavigation projectId={projectId} />
      <ProjectSubNavigation />
      <div className="flex flex-col">Dummy Settings</div>
    </div>
  );
}
