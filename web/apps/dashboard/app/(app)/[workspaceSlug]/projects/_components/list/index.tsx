import { ListViews } from "@/components/list-views";
import { ResourceSearchInput, useListSearch } from "@/components/resource-search-input";
import { useListView } from "@/hooks/use-list-view";
import { useVisibleProjects } from "@/hooks/use-visible-projects";
import { useWorkspaceNavigation } from "@/hooks/use-workspace-navigation";
import { collection } from "@/lib/collections";
import { projectDisplayName } from "@/lib/collections/deploy/projects";
import { useCollectionPolling } from "@/lib/collections/use-collection-polling";
import { useShowMore } from "@/lib/loadable/list";
import { IconTriangleWarningOutline18 } from "@unkey/icons";
import { match } from "@unkey/match";
import {
  AlertBanner,
  AlertBannerActions,
  AlertBannerDescription,
  Button,
  ResourceList,
  ResourceListHeader,
  ResourceListViewToggle,
} from "@unkey/ui";
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
  const [view, setView] = useListView("projects-view");
  const search = useListSearch();
  const needle = search.toLowerCase();
  const rows = projects.data
    .map((project) => ({ project, name: projectDisplayName(project, workspace.name) }))
    .filter(({ name }) => name.toLowerCase().includes(needle));
  const list = useShowMore({
    rows,
    total: projects.data.length,
    isLoading: projects.isLoading,
    search,
  });

  useCollectionPolling(
    () => Promise.all([collection.projects.utils.refetch(), collection.apps.utils.refetch()]),
    { intervalMs: IDLE_POLL_MS, enabled: true },
  );

  return match(list)
    .with({ status: "empty" }, () => <EmptyProjects />)
    .otherwise((content) => (
      <ResourceList>
        {gated ? (
          <AlertBanner variant="warning">
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
        <ResourceListHeader>
          <ResourceSearchInput label="Search projects" placeholder="Search projects" />
          <ResourceListViewToggle value={view} onValueChange={setView} />
        </ResourceListHeader>
        <ListViews
          view={view}
          content={content}
          noun="projects"
          table={PROJECT_TABLE}
          rowKey={({ project }) => project.id}
          renderCard={({ project, name }) => <ProjectGridItem project={project} name={name} />}
          renderRow={({ project, name }, table) => (
            <ProjectRowItem project={project} name={name} table={table} />
          )}
          cardSkeleton={<ProjectCardSkeleton />}
        />
      </ResourceList>
    ));
}
