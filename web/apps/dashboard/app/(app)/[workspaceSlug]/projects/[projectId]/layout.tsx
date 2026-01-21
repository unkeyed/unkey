"use client";
import { collection, collectionManager } from "@/lib/collections";
import { eq, useLiveQuery } from "@tanstack/react-db";
import { useEffect, useState } from "react";
import { ProjectLayoutContext } from "./(overview)/layout-provider";
import { ProjectNavigation } from "./(overview)/navigations/project-navigation";
import { ProjectDetailsExpandable } from "./(overview)/details/project-details-expandables";

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
  const lastestDeploymentId = projects.data.at(0)?.latestDeploymentId;

  // We just wanna refetch domains as soon as lastestCommitTimestamp changes.
  // We could use the liveDeploymentId for that but when user make `env=preview` this doesn't refetch properly.
  // biome-ignore lint/correctness/useExhaustiveDependencies: Read above.
  useEffect(() => {
    //@ts-expect-error Without this we can't refetch domains on-demand. It's either this or we do `refetchInternal` on domains collection level.
    // Second approach causing too any re-renders. This is fine because data is partitioned and centralized in this context.
    // Until they introduce a way to invalidate collections properly we stick to this.
    collections.domains.utils.refetch();
  }, [lastestDeploymentId]);

  return (
    <ProjectLayoutContext.Provider
      value={{
        isDetailsOpen,
        setIsDetailsOpen,
        projectId,
        collections,
        liveDeploymentId,
      }}
    >
      <div className="h-screen flex flex-col overflow-hidden">
        <ProjectNavigation
          projectId={projectId}
          onClick={() => setIsDetailsOpen(!isDetailsOpen)}
          isDetailsOpen={isDetailsOpen}
          liveDeploymentId={liveDeploymentId}
          onMount={setTableDistanceToTop}
        />
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
