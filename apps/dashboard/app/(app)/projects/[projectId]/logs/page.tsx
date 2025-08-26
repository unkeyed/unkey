"use client";

import { ProjectNavigation } from "../navigations/project-navigation";
import { ProjectSubNavigation } from "../navigations/project-sub-navigation";

export default function ProjectLogs({
  params: { projectId },
}: {
  params: { projectId: string };
}) {
  return (
    <div>
      <ProjectNavigation projectId={projectId} />
      <ProjectSubNavigation onMount={() => {}} />
      <div className="flex flex-col">Go away!</div>
    </div>
  );
}
