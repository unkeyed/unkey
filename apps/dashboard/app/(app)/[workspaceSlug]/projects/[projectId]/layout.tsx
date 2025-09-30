"use client";
import { collection, collectionManager } from "@/lib/collections";
import { eq, useLiveQuery } from "@tanstack/react-db";
import { DoubleChevronLeft } from "@unkey/icons";
import { Button, InfoTooltip } from "@unkey/ui";
import { useEffect, useState } from "react";
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
  const [tableDistanceToTop, setTableDistanceToTop] = useState(0);
  const [isDetailsOpen, setIsDetailsOpen] = useState(false);

  const collections = collectionManager.getProjectCollections(projectId);

  const projects = useLiveQuery((q) =>
    q.from({ project: collection.projects }).where(({ project }) => eq(project.id, projectId)),
  );

  const liveDeploymentId = projects.data.at(0)?.liveDeploymentId;

  // biome-ignore lint/correctness/useExhaustiveDependencies: We just wanna refetch domains as soon as liveDeploymentId changes.
  useEffect(() => {
    //@ts-expect-error Without this we can't refetch domains on-demand. It's either this or we do `refetchInternal` on domains collection level.
    // Second approach causing too any re-renders. This is fine because data is partitioned and centralized in this context.
    // Until they introduce a way to invalidate collections properly we stick to this.
    collections.domains.utils.refetch();
  }, [liveDeploymentId]);

  const getTooltipContent = () => {
    if (!liveDeploymentId) {
      return "No deployments available. Deploy your project to view details.";
    }
    return isDetailsOpen ? "Hide deployment details" : "Show deployment details";
  };

  return (
    <ProjectLayoutContext.Provider
      value={{
        isDetailsOpen,
        setIsDetailsOpen,
        projectId,
        collections,
      }}
    >
      <div className="h-screen flex flex-col overflow-hidden">
        <ProjectNavigation projectId={projectId} />
        <div className="flex items-center flex-shrink-0">
          <ProjectSubNavigation
            onMount={setTableDistanceToTop}
            detailsExpandableTrigger={
              <InfoTooltip
                asChild
                content={getTooltipContent()}
                position={{
                  side: "bottom",
                  align: "end",
                }}
              >
                <Button
                  variant="ghost"
                  className="size-7"
                  disabled={!liveDeploymentId}
                  onClick={() => setIsDetailsOpen(!isDetailsOpen)}
                >
                  <DoubleChevronLeft
                    iconsize="lg-medium"
                    className="text-gray-13"
                  />
                </Button>
              </InfoTooltip>
            }
          />
        </div>
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
