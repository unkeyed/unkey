"use client";

import { LogDrainDetail } from "@/components/log-drains/log-drain-detail";
import { useParams } from "next/navigation";
import { ProjectContentWrapper } from "../../../components/project-content-wrapper";

export default function LogDrainDetailPage() {
  const params = useParams();
  const projectId = typeof params?.projectId === "string" ? params.projectId : null;
  const workspaceSlug = typeof params?.workspaceSlug === "string" ? params.workspaceSlug : null;
  const drainId = typeof params?.drainId === "string" ? params.drainId : null;

  if (!projectId || !workspaceSlug || !drainId) {
    return null;
  }

  return (
    <ProjectContentWrapper centered maxWidth="960px" className="mt-8">
      <LogDrainDetail
        scope="project"
        projectId={projectId}
        workspaceSlug={workspaceSlug}
        drainId={drainId}
      />
    </ProjectContentWrapper>
  );
}
