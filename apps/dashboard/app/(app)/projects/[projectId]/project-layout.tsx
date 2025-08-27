import { useState, useCallback } from "react";
import { ProjectDetailsExpandable } from "./details/project-details-expandables";
import { ProjectNavigation } from "./navigations/project-navigation";
import { ProjectSubNavigation } from "./navigations/project-sub-navigation";
import { Button, InfoTooltip } from "@unkey/ui";
import { DoubleChevronLeft } from "@unkey/icons";

type ProjectLayoutProps = {
  projectId: string;
  children: React.ReactNode;
};

export const ProjectLayout = ({ projectId, children }: ProjectLayoutProps) => {
  const [tableDistanceToTop, setTableDistanceToTop] = useState(0);
  const [isDetailsOpen, setIsDetailsOpen] = useState(true);

  const handleDistanceToTop = useCallback((distanceToTop: number) => {
    setTableDistanceToTop(distanceToTop);
  }, []);

  return (
    <div>
      <ProjectNavigation projectId={projectId} />
      <div className="flex items-center">
        <ProjectSubNavigation
          onMount={handleDistanceToTop}
          detailsExpandableTrigger={
            <InfoTooltip
              content="Show details"
              position={{
                side: "bottom",
                align: "end",
              }}
            >
              <Button
                variant="ghost"
                className="size-6"
                onClick={() => setIsDetailsOpen(true)}
              >
                <DoubleChevronLeft size="lg-medium" className="text-gray-12" />
              </Button>
            </InfoTooltip>
          }
        />
      </div>
      <div className="flex">
        <div className="flex-1">{children}</div>
        <ProjectDetailsExpandable
          tableDistanceToTop={tableDistanceToTop}
          isOpen={isDetailsOpen}
          onClose={() => setIsDetailsOpen(false)}
        />
      </div>
    </div>
  );
};
