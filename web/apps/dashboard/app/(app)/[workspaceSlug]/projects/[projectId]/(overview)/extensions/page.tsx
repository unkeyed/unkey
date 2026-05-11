"use client";

import { useInstallations } from "@/lib/extensions/installations";
import { useParams } from "next/navigation";
import { ProjectContentWrapper } from "../../components/project-content-wrapper";
import { ExtensionsHeader } from "./components/extensions-header";
import { ExtensionsClient } from "./extensions-client";

export default function ExtensionsPage() {
  const params = useParams<{ workspaceSlug: string; projectId: string }>();
  const basePath = `/${params.workspaceSlug}/projects/${params.projectId}/extensions`;
  const { installations } = useInstallations(params.projectId);

  return (
    <ProjectContentWrapper centered maxWidth="1200px" className="mt-8">
      <ExtensionsHeader
        basePath={basePath}
        active="marketplace"
        installedCount={installations.length}
      />
      <ExtensionsClient basePath={basePath} projectId={params.projectId} />
    </ProjectContentWrapper>
  );
}
