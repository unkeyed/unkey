"use client";

import { LogDrainWizard } from "@/components/log-drains/log-drain-wizard";
import { useParams } from "next/navigation";
import { ProjectContentWrapper } from "../../../components/project-content-wrapper";

export default function NewLogDrainPage() {
  const params = useParams();
  const projectId = typeof params?.projectId === "string" ? params.projectId : null;
  const workspaceSlug = typeof params?.workspaceSlug === "string" ? params.workspaceSlug : null;

  if (!projectId || !workspaceSlug) {
    return null;
  }

  return (
    <ProjectContentWrapper centered maxWidth="960px" className="mt-8">
      <LogDrainWizard scope="project" projectId={projectId} workspaceSlug={workspaceSlug} />
    </ProjectContentWrapper>
  );
}
