"use client";

import { ProjectNavigation } from "./navigations/project-navigation";
import { ProjectSubNavigation } from "./navigations/project-sub-navigation";

export default function ProjectDetails({
  params: { projectId },
}: {
  params: { projectId: string };
}) {
  return (
    <div>
      <ProjectNavigation projectId={projectId} />
      <ProjectSubNavigation />
      <div className="flex flex-col">Overview</div>
    </div>
  );
}
