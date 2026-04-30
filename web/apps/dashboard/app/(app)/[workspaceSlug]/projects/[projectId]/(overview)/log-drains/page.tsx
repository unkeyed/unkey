"use client";

import { LogDrainsList } from "@/components/log-drains/log-drains-list";
import { useParams } from "next/navigation";
import { ProjectContentWrapper } from "../../components/project-content-wrapper";

export default function LogDrainsPage() {
  const params = useParams();
  const projectId = typeof params?.projectId === "string" ? params.projectId : null;
  const workspaceSlug = typeof params?.workspaceSlug === "string" ? params.workspaceSlug : null;

  if (!projectId || !workspaceSlug) {
    return null;
  }

  return (
    <ProjectContentWrapper centered maxWidth="960px" className="mt-8">
      <LogDrainsList scope="project" projectId={projectId} workspaceSlug={workspaceSlug} />
    </ProjectContentWrapper>
  );
}
