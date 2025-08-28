"use client";
import { DoubleChevronLeft } from "@unkey/icons";
import { Button, InfoTooltip } from "@unkey/ui";
import { useCallback, useRef, useState } from "react";
import { ProjectDetailsExpandable } from "./details/project-details-expandables";
import { ProjectLayoutContext } from "./layout-provider";
import { ProjectNavigation } from "./navigations/project-navigation";
import { ProjectSubNavigation } from "./navigations/project-sub-navigation";

export default function ProjectLayoutWrapper({
  children,
  params: { projectId },
}: {
  children: React.ReactNode;
  params: { projectId: string };
}) {
  return <ProjectLayout projectId={projectId}>{children}</ProjectLayout>;
}

type ProjectLayoutProps = {
  projectId: string;
  children: React.ReactNode;
};

const ProjectLayout = ({ projectId, children }: ProjectLayoutProps) => {
  const tableDistanceToTopRef = useRef(0);
  const [isDetailsOpen, setIsDetailsOpen] = useState(true);

  const handleDistanceToTop = useCallback((distanceToTop: number) => {
    tableDistanceToTopRef.current = distanceToTop;
  }, []);

  const contextValue = {
    isDetailsOpen,
    setIsDetailsOpen,
  };

  return (
    <ProjectLayoutContext.Provider value={contextValue}>
      <div className="h-screen flex flex-col overflow-hidden">
        <ProjectNavigation projectId={projectId} />
        <div className="flex items-center flex-shrink-0">
          <ProjectSubNavigation
            onMount={handleDistanceToTop}
            detailsExpandableTrigger={
              <InfoTooltip
                asChild
                content="Show details"
                position={{
                  side: "bottom",
                  align: "end",
                }}
              >
                <Button
                  variant="ghost"
                  className="size-7"
                  onClick={() => setIsDetailsOpen(!isDetailsOpen)}
                >
                  <DoubleChevronLeft size="lg-medium" className="text-gray-13" />
                </Button>
              </InfoTooltip>
            }
          />
        </div>
        <div className="flex flex-1 min-h-0">
          <div className="flex-1 overflow-auto">{children}</div>
          <ProjectDetailsExpandable
            tableDistanceToTop={tableDistanceToTopRef.current}
            isOpen={isDetailsOpen}
            onClose={() => setIsDetailsOpen(false)}
          />
        </div>
      </div>
    </ProjectLayoutContext.Provider>
  );
};
