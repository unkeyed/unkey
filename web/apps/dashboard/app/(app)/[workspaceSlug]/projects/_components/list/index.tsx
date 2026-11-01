import { ListViews } from "@/components/list-views";
import { ResourceSearchInput, useListSearch } from "@/components/resource-search-input";
import { useVisibleProjects } from "@/hooks/use-visible-projects";
import { useWorkspaceNavigation } from "@/hooks/use-workspace-navigation";
import { projectDisplayName } from "@/lib/collections/deploy/projects";
import { useCollectionPolling } from "@/lib/collections/use-collection-polling";
import { refetchAllApps, refetchProjects } from "@/lib/deploy-data";
import { GRID_PAGE_SIZE } from "@/lib/list-limits";
import { useShowMore } from "@/lib/loadable/list";
import { IconTriangleWarningOutline18 } from "@unkey/icons";
import { AlertBanner, AlertBannerActions, AlertBannerDescription, Button } from "@unkey/ui";
import { useState } from "react";
import { DeployPlanGateDialog } from "../deploy-plan-gate-dialog";
import { useDeployGate } from "../hooks/use-deploy-gate";
import { EmptyProjects } from "./empty-projects";
import { ProjectCardSkeleton } from "./project-card-skeleton";
import { PROJECT_TABLE, ProjectGridItem, ProjectRowItem } from "./project-list-item";

const IDLE_POLL_MS = 60_000;

export function ProjectsList() {
  const { gated } = useDeployGate();
  const [isPlanOpen, setIsPlanOpen] = useState(false);
  const workspace = useWorkspaceNavigation();
  const projects = useVisibleProjects();
  const search = useListSearch();
  const needle = search.toLowerCase();
  const list = useShowMore({
    all: projects.data.map((project) => ({
      project,
      name: projectDisplayName(project, workspace.name),
    })),
    isLoading: projects.isLoading,
    matches: (rows) => rows.filter(({ name }) => name.toLowerCase().includes(needle)),
    search,
    pageSize: GRID_PAGE_SIZE,
  });

  useCollectionPolling(() => Promise.all([refetchProjects(), refetchAllApps()]), {
    intervalMs: IDLE_POLL_MS,
    enabled: true,
  });

  return (
    <ListViews
      list={list}
      viewStorageKey="projects-view"
      noun="projects"
      failedTitle="Could not load projects"
      empty={<EmptyProjects />}
      banner={
        <>
          {gated ? (
            <AlertBanner variant="warning">
              <IconTriangleWarningOutline18 className="size-3.5" aria-hidden="true" />
              <AlertBannerDescription className="truncate">
                No active Compute plan. Existing projects stay visible, but creating and deploying
                are paused.
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
        </>
      }
      controls={<ResourceSearchInput label="Search projects" placeholder="Search projects" />}
      table={PROJECT_TABLE}
      rowKey={({ project }) => project.id}
      renderCard={({ project, name }) => <ProjectGridItem project={project} name={name} />}
      renderRow={({ project, name }) => <ProjectRowItem project={project} name={name} />}
      cardSkeleton={<ProjectCardSkeleton />}
    />
  );
}
