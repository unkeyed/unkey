"use client";

import { ProjectLayout } from "../project-layout";

export default function ProjectLogs({
  params: { projectId },
}: {
  params: { projectId: string };
}) {
  return (
    <ProjectLayout projectId={projectId}>
      <div className="bg-success-10">Overview</div>
    </ProjectLayout>
  );
}
