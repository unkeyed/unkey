"use client";
import { trpc } from "@/lib/trpc/client";
import { DoubleChevronLeft } from "@unkey/icons";
import { Button, InfoTooltip } from "@unkey/ui";
import { useCallback, useEffect, useState } from "react";
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

const FAKE_DEPLOYMENT_ID = "im-a-fake-deployment-id";
const ProjectLayout = ({ projectId, children }: ProjectLayoutProps) => {
  const trpcUtil = trpc.useUtils();
  const [tableDistanceToTop, setTableDistanceToTop] = useState(0);
  const [isDetailsOpen, setIsDetailsOpen] = useState(false);

  useEffect(() => {
    trpcUtil.deploy.project.envs.getEnvs.prefetch({
      projectId,
    });
  }, [trpcUtil, projectId]);

  // This will be called on mount to determine the offset to top, then it will prefetch project details and mount project details drawer.
  const handleDistanceToTop = useCallback(
    async (distanceToTop: number) => {
      setTableDistanceToTop(distanceToTop);

      if (distanceToTop !== 0) {
        try {
          // Only proceed if prefetch succeeds
          await trpcUtil.deploy.project.activeDeployment.details.prefetch({
            deploymentId: FAKE_DEPLOYMENT_ID,
          });

          setTimeout(() => {
            setIsDetailsOpen(true);
          }, 200);
        } catch (error) {
          console.error("Failed to prefetch project details:", error);
          // Don't open the drawer if prefetch fails
        }
      }
    },
    [trpcUtil],
  );

  return (
    <ProjectLayoutContext.Provider
      value={{
        isDetailsOpen,
        setIsDetailsOpen,
        activeDeploymentId: FAKE_DEPLOYMENT_ID,
        projectId,
      }}
    >
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
            tableDistanceToTop={tableDistanceToTop}
            isOpen={isDetailsOpen}
            onClose={() => setIsDetailsOpen(false)}
            activeDeploymentId={FAKE_DEPLOYMENT_ID}
          />
        </div>
      </div>
    </ProjectLayoutContext.Provider>
  );
};
