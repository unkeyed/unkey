"use client";
import { collection } from "@/lib/collections";
import { eq, useLiveQuery } from "@tanstack/react-db";
import { usePathname } from "next/navigation";
import { use, useState } from "react";
import { ProjectDataProvider } from "./(overview)/data-provider";
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
  const [tableDistanceToTop, setTableDistanceToTop] = useState(0);
  const [isDetailsOpen, setIsDetailsOpen] = useState(false);

  const pathname = usePathname();
  const isOnDeploymentDetail =
    pathname?.includes("/deployments/") && pathname.split("/").filter(Boolean).length >= 5; // /workspace/projects/projectId/deployments/deploymentId/*

  const projects = useLiveQuery((q) =>
    q.from({ project: collection.projects }).where(({ project }) => eq(project.id, projectId)),
  );

  const liveDeploymentId = projects.data.at(0)?.liveDeploymentId;

  return (
    <ProjectDataProvider projectId={projectId}>
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
    </ProjectDataProvider>
  );
};
