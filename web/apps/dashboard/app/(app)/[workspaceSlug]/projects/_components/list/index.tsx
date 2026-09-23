import { ProximityPrefetch } from "@/components/proximity-prefetch";
import { useVisibleProjects } from "@/hooks/use-visible-projects";
import { useWorkspaceNavigation } from "@/hooks/use-workspace-navigation";
import { collection } from "@/lib/collections";
import { isDeploymentInFlight } from "@/lib/collections/deploy/deployment-status";
import { projectDisplayName } from "@/lib/collections/deploy/projects";
import { useCollectionPolling } from "@/lib/collections/use-collection-polling";
import { IconDotsOutline18, IconTriangleWarningOutline18 } from "@unkey/icons";
import { Button } from "@unkey/ui";
import { useState } from "react";
import { DeployPlanGateDialog } from "../deploy-plan-gate-dialog";
import { useDeployGate } from "../hooks/use-deploy-gate";
import { ProjectActions } from "./project-actions";
import { ProjectCard } from "./project-card";
import { ProjectCardSkeleton } from "./project-card-skeleton";

const MAX_SKELETON_COUNT = 3;

const IDLE_POLL_MS = 60_000;
const BUILDING_POLL_MS = 5_000;

export const ProjectsList = () => {
  const { gated } = useDeployGate();
  const [isPlanOpen, setIsPlanOpen] = useState(false);
  const workspace = useWorkspaceNavigation();
  const projects = useVisibleProjects();

  const hasInFlightDeployment = projects.data.some((project) =>
    project.apps.some(
      (app) => app.headlineDeployment && isDeploymentInFlight(app.headlineDeployment.status),
    ),
  );
  useCollectionPolling(() => collection.projects.utils.refetch(), {
    intervalMs: hasInFlightDeployment ? BUILDING_POLL_MS : IDLE_POLL_MS,
    enabled: true,
  });

  if (projects.isLoading) {
    return (
      <div aria-busy="true" className="grid gap-4 grid-cols-1 md:grid-cols-2 xl:grid-cols-3">
        {Array.from({ length: MAX_SKELETON_COUNT }).map((_, i) => (
          // biome-ignore lint/suspicious/noArrayIndexKey: skeleton items don't need stable keys
          <ProjectCardSkeleton key={i} />
        ))}
      </div>
    );
  }

  return (
    <>
      {gated ? (
        <div className="mb-4 flex items-center justify-between gap-4 rounded-lg border border-warningA-6 bg-warningA-2 px-4 py-3">
          <div className="flex min-w-0 items-center gap-3">
            <IconTriangleWarningOutline18 className="size-3.5 shrink-0 text-warning-11" />
            <p className="truncate text-[13px] text-gray-11">
              No active Compute plan. Existing projects stay visible, but creating and deploying are
              paused.
            </p>
          </div>
          <Button
            variant="outline"
            size="md"
            className="bg-background"
            onClick={() => setIsPlanOpen(true)}
          >
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
              name={projectDisplayName(project, workspace.name)}
              apps={project.apps}
              actions={
                <ProjectActions projectId={project.id}>
                  <Button variant="ghost" size="icon" className="shrink-0" title="Project actions">
                    <IconDotsOutline18 />
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
