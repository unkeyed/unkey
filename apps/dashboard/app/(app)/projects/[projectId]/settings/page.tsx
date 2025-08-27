"use client";

import { ProjectLayout } from "../project-layout";

export default function ProjectSettings({
  params: { projectId },
}: {
  params: { projectId: string };
}) {
  return (
    <ProjectLayout projectId={projectId}>
      <div className="bg-success-10">ProjectSettings</div>
    </ProjectLayout>
  );
}
