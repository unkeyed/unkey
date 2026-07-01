import { ProximityPrefetch } from "@/components/proximity-prefetch";
import { collection } from "@/lib/collections";
import { useLiveQuery } from "@tanstack/react-db";
import { Dots, TriangleWarning2 } from "@unkey/icons";
import { Button, Empty } from "@unkey/ui";
import { useState } from "react";
import { DeployPlanGateDialog } from "../deploy-plan-gate-dialog";
import { useDeployGate } from "../hooks/use-deploy-gate";
import { ProjectActions } from "./project-actions";
import { ProjectCard } from "./project-card";
import { ProjectCardSkeleton } from "./project-card-skeleton";

// One row at the 3-column desktop width so loading doesn't tower over the
// real list before it resolves.
const MAX_SKELETON_COUNT = 3;

export const ProjectsList = () => {
  const { gated } = useDeployGate();
  const [isPlanOpen, setIsPlanOpen] = useState(false);
  const projects = useLiveQuery((q) => q.from({ project: collection.projects }));

  if (projects.isLoading) {
    return (
      <div className="grid gap-4 grid-cols-1 md:grid-cols-2 xl:grid-cols-3">
        {Array.from({ length: MAX_SKELETON_COUNT }).map((_, i) => (
          // biome-ignore lint/suspicious/noArrayIndexKey: skeleton items don't need stable keys
          <ProjectCardSkeleton key={i} />
        ))}
      </div>
    );
  }

  if (projects.data.length === 0) {
    // The gated + no-projects case is handled one level up by the page-level
    // EmptyProjects screen, so this only renders for the non-gated empty case.
    return (
      <div className="w-full flex justify-center items-center h-full">
        <Empty className="w-[400px] flex items-start">
          <Empty.Icon className="w-auto" />
          <Empty.Title>No Projects Found</Empty.Title>
          <Empty.Description className="text-left">
            This workspace has no projects yet.
          </Empty.Description>
        </Empty>
      </div>
    );
  }

  return (
    <>
      {gated ? (
        // Quiet notice, same language as the billing page banners: one slim
        // strip, not a warning slab. The create button's tooltip carries the
        // same message, so this only needs to orient, not shout.
        <div className="mb-4 flex items-center justify-between gap-4 rounded-lg border border-warningA-6 bg-warningA-2 px-4 py-3">
          <div className="flex min-w-0 items-center gap-3">
            <TriangleWarning2 iconSize="md-regular" className="shrink-0 text-warning-11" />
            <p className="truncate text-[13px] text-gray-11">
              No active Compute plan. Existing projects stay visible, but creating and deploying are
              paused.
            </p>
          </div>
          <Button variant="outline" size="md" onClick={() => setIsPlanOpen(true)}>
            Choose a plan
          </Button>
        </div>
      ) : null}
      <DeployPlanGateDialog isOpen={isPlanOpen} onOpenChange={setIsPlanOpen} from="banner" />
      <div className="grid gap-4 grid-cols-1 md:grid-cols-2 xl:grid-cols-3">
        {projects.data.map((project) => (
          <ProximityPrefetch distance={300} debounceDelay={150} key={project.id}>
            <ProjectCard
              projectId={project.id}
              name={project.name}
              appCount={project.appCount}
              apps={project.apps}
              actions={
                <ProjectActions projectId={project.id}>
                  <Button
                    variant="ghost"
                    size="icon"
                    className="mb-auto shrink-0"
                    title="Project actions"
                  >
                    <Dots iconSize="sm-regular" />
                  </Button>
                </ProjectActions>
              }
            />
          </ProximityPrefetch>
        ))}
      </div>
    </>
  );
};
