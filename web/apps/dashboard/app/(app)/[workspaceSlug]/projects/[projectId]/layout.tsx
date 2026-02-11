"use client";
import { usePathname } from "next/navigation";
import { use, useState } from "react";
import { ProjectDataProvider, useProjectData } from "./(overview)/data-provider";
import { ProjectDetailsExpandable } from "./(overview)/details/project-details-expandables";
import { ProjectLayoutContext } from "./(overview)/layout-provider";
import { ProjectNavigation } from "./(overview)/navigations/project-navigation";

export default function ProjectLayoutWrapper(props: {
  children: React.ReactNode;
  params: Promise<{ projectId: string }>;
}) {
  const params = use(props.params);

  const { projectId } = params;

  const { children } = props;

  return <ProjectLayout projectId={projectId}>{children}</ProjectLayout>;
}

type ProjectLayoutProps = {
  projectId: string;
  children: React.ReactNode;
};

const ProjectLayout = ({ projectId, children }: ProjectLayoutProps) => {
  return (
    <ProjectDataProvider projectId={projectId}>
      <ProjectLayoutInner projectId={projectId}>{children}</ProjectLayoutInner>
    </ProjectDataProvider>
  );
};

const ProjectLayoutInner = ({ projectId, children }: ProjectLayoutProps) => {
  const [tableDistanceToTop, setTableDistanceToTop] = useState(0);
  const [isDetailsOpen, setIsDetailsOpen] = useState(false);

  const pathname = usePathname();
  const isOnDeploymentDetail =
    pathname?.includes("/deployments/") && pathname.split("/").filter(Boolean).length >= 5; // /workspace/projects/projectId/deployments/deploymentId/*

  const { project } = useProjectData();
  const liveDeploymentId = project?.liveDeploymentId;

  return (
    <ProjectLayoutContext.Provider
      value={{
        isDetailsOpen,
        setIsDetailsOpen,
        projectId,
        liveDeploymentId,
      }}
    >
      <div className="h-screen flex flex-col overflow-hidden">
        {!isOnDeploymentDetail && (
          <ProjectNavigation
            projectId={projectId}
            onClick={() => setIsDetailsOpen(!isDetailsOpen)}
            isDetailsOpen={isDetailsOpen}
            liveDeploymentId={liveDeploymentId}
            onMount={setTableDistanceToTop}
          />
        )}
        <div className="flex flex-1 min-h-0">
          <div className="flex-1 overflow-auto">{children}</div>
          <ProjectDetailsExpandable
            projectId={projectId}
            tableDistanceToTop={tableDistanceToTop}
            isOpen={isDetailsOpen && Boolean(liveDeploymentId)}
            onClose={() => setIsDetailsOpen(false)}
          />
        </div>
      </div>
    </ProjectLayoutContext.Provider>
  );
};
