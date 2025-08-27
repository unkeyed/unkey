"use client";
import { ProjectLayout } from "./project-layout";

export default function ProjectDetails({
  params: { projectId },
}: {
  params: { projectId: string };
}) {
  return (
    <ProjectLayout projectId={projectId}>
      {/*616 stands for 256 for sidebar and 360 for the project details*/}
      <div className="w-[calc(100vw-616px)] border border-error-9 flex justify-center">
        <div className="mt-4">asd</div>
      </div>
    </ProjectLayout>
  );
}
