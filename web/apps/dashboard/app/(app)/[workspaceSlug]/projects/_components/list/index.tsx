import { useNearViewport } from "@/hooks/use-near-viewport";
import { useVisibleProjects } from "@/hooks/use-visible-projects";
import { useWorkspaceNavigation } from "@/hooks/use-workspace-navigation";
import { type Project, collection } from "@/lib/collections";
import { projectDisplayName } from "@/lib/collections/deploy/projects";
import { useCollectionPolling } from "@/lib/collections/use-collection-polling";
import { IconDotsOutline18, IconTriangleWarningOutline18 } from "@unkey/icons";
import { AlertBanner, AlertBannerActions, AlertBannerDescription, Button } from "@unkey/ui";
import { useState } from "react";
import { warmProjectPage } from "../../[projectId]/_components/apps-list/queries";
import { DeployPlanGateDialog } from "../deploy-plan-gate-dialog";
import { useDeployGate } from "../hooks/use-deploy-gate";
import { ProjectActions } from "./project-actions";
import { ProjectCard } from "./project-card";
import { ProjectCardSkeleton } from "./project-card-skeleton";
import { useProjectCard } from "./use-project-card";

const MAX_SKELETON_COUNT = 3;

const IDLE_POLL_MS = 60_000;

export const ProjectsList = () => {
  const { gated } = useDeployGate();
  const [isPlanOpen, setIsPlanOpen] = useState(false);
  const workspace = useWorkspaceNavigation();
  const projects = useVisibleProjects();

  useCollectionPolling(
    () => Promise.all([collection.projects.utils.refetch(), collection.apps.utils.refetch()]),
    { intervalMs: IDLE_POLL_MS, enabled: true },
  );

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
        <AlertBanner variant="warning" className="mb-4">
          <IconTriangleWarningOutline18 className="size-3.5" aria-hidden="true" />
          <AlertBannerDescription className="truncate">
            No active Compute plan. Existing projects stay visible, but creating and deploying are
            paused.
          </AlertBannerDescription>
          <AlertBannerActions>
            <Button
              variant="outline"
              size="md"
              className="bg-background"
              onClick={() => setIsPlanOpen(true)}
            >
              Choose a plan
            </Button>
          </AlertBannerActions>
        </AlertBanner>
      ) : null}
      <DeployPlanGateDialog isOpen={isPlanOpen} onOpenChange={setIsPlanOpen} from="banner" />
      <div className="grid gap-4 grid-cols-1 md:grid-cols-2 xl:grid-cols-3">
        {projects.data.map((project) => (
          <ProjectListCard
            project={project}
            name={projectDisplayName(project, workspace.name)}
            key={project.id}
          />
        ))}
      </div>
    </>
  );
};

function ProjectListCard({ project, name }: { project: Project; name: string }) {
  const { ref, isNear } = useNearViewport<HTMLDivElement>();
  const { apps, isLoading } = useProjectCard(project.id, { nearViewport: isNear });
  return (
    <div
      ref={ref}
      className="h-full"
      onPointerEnter={() => warmProjectPage(project.id)}
      onFocusCapture={() => warmProjectPage(project.id)}
    >
      <ProjectCard
        projectId={project.id}
        name={name}
        apps={apps}
        isLoading={isLoading}
        actions={
          <ProjectActions projectId={project.id}>
            <Button variant="ghost" size="icon" className="shrink-0" title="Project actions">
              <IconDotsOutline18 />
            </Button>
          </ProjectActions>
        }
      />
    </div>
  );
}
