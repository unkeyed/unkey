"use client";
import { type PropsWithChildren, useState } from "react";
import { ProjectDataProvider, useProjectData } from "./(overview)/data-provider";
import { ProjectDetailsExpandable } from "./(overview)/details/project-details-expandables";
import { ProjectLayoutContext } from "./(overview)/layout-provider";
import { ProjectNavigation } from "./(overview)/navigations/project-navigation";
import { PendingRedeployBanner } from "./components/pending-redeploy-banner";

export default function ProjectLayoutWrapper({ children }: PropsWithChildren) {
  return <ProjectLayout>{children}</ProjectLayout>;
}

const ProjectLayout = ({ children }: PropsWithChildren) => {
  return (
    <ProjectDataProvider>
      <ProjectLayoutInner>{children}</ProjectLayoutInner>
    </ProjectDataProvider>
  );
};

const ProjectLayoutInner = ({ children }: PropsWithChildren) => {
  const [tableDistanceToTop, setTableDistanceToTop] = useState(0);
  const [isDetailsOpen, setIsDetailsOpen] = useState(false);

  const { project } = useProjectData();
  const currentDeploymentId = project?.currentDeploymentId;

  return (
    <ProjectLayoutContext.Provider
      value={{
        isDetailsOpen,
        setIsDetailsOpen,
        tableDistanceToTop,
      }}
    >
      <div className="h-full flex flex-col overflow-hidden">
        <ProjectNavigation
          onClick={() => setIsDetailsOpen(!isDetailsOpen)}
          isDetailsOpen={isDetailsOpen}
          currentDeploymentId={currentDeploymentId}
          onMount={setTableDistanceToTop}
        />
        <div className="flex flex-1 min-h-0">
          <div className="flex-1 overflow-auto">{children}</div>
          <ProjectDetailsExpandable
            tableDistanceToTop={tableDistanceToTop}
            isOpen={isDetailsOpen && Boolean(currentDeploymentId)}
            onClose={() => setIsDetailsOpen(false)}
          />
        </div>
        <PendingRedeployBanner />
      </div>
    </ProjectLayoutContext.Provider>
  );
};
