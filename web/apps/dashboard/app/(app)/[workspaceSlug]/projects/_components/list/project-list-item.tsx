"use client";

import { ListTableRow, listTable } from "@/components/list-views";
import { useNearViewport } from "@/hooks/use-near-viewport";
import { useWorkspaceNavigation } from "@/hooks/use-workspace-navigation";
import type { Project } from "@/lib/collections";
import { routes } from "@/lib/navigation/routes";
import { IconCubeOutline18 } from "@unkey/icons";
import { Skeleton } from "@unkey/ui";
import { DeployedAgo } from "../../[projectId]/_components/apps-list/app-source";
import { warmProjectPage } from "../../[projectId]/_components/apps-list/queries";
import {
  type WarmHandlers,
  useWarmAppPage,
} from "../../[projectId]/apps/[appId]/(overview)/hooks/use-warm-app-page";
import { ProjectActionsButton } from "./project-actions";
import { ProjectCard } from "./project-card";
import { projectSummaryOf } from "./project-summary";
import { useProjectCard } from "./use-project-card";

export const PROJECT_TABLE = listTable({
  minWidth: 560,
  columns: [
    { id: "project", header: "Project", width: "minmax(0,1.6fr)", skeleton: "title" },
    { id: "apps", header: "Apps", width: "minmax(0,0.8fr)", skeleton: "text" },
    { id: "deployed", header: "Last deployed", width: "minmax(0,1fr)", skeleton: "text" },
    { id: "actions", header: null, width: "32px", skeleton: "none" },
  ],
});

type ProjectItemProps = { project: Project; name: string };

type ProjectRowItemProps = ProjectItemProps & { table: typeof PROJECT_TABLE };

function useProjectListItem(project: Project, nearViewport: boolean) {
  const { apps, isLoading } = useProjectCard(project.id, { nearViewport });
  const { warmApp, cancelWarm } = useWarmAppPage();
  const warmProjectAndOnlyApp = () => {
    warmProjectPage(project.id);
    const onlyApp = apps.length === 1 ? apps[0] : undefined;
    if (onlyApp) {
      warmApp(project.id, onlyApp.id);
    }
  };
  const warm: WarmHandlers = {
    onPointerEnter: warmProjectAndOnlyApp,
    onPointerLeave: cancelWarm,
    onFocusCapture: warmProjectAndOnlyApp,
  };

  return { apps, isLoading, warm };
}

export function ProjectGridItem({ project, name }: ProjectItemProps) {
  const { ref, isNear } = useNearViewport<HTMLDivElement>();
  const { apps, isLoading, warm } = useProjectListItem(project, isNear);

  return (
    <ProjectCard
      ref={ref}
      warm={warm}
      projectId={project.id}
      name={name}
      apps={apps}
      isLoading={isLoading}
    />
  );
}

export function ProjectRowItem({ project, name, table }: ProjectRowItemProps) {
  const { ref, isNear } = useNearViewport<HTMLLIElement>();
  const { apps, isLoading, warm } = useProjectListItem(project, isNear);
  const workspace = useWorkspaceNavigation();
  const { appsLabel, deployedAt } = projectSummaryOf(apps);

  return (
    <ListTableRow
      ref={ref}
      table={table}
      href={routes.projects.detail({ workspaceSlug: workspace.slug, projectId: project.id })}
      label={`View ${name} project`}
      {...warm}
      cells={{
        project: (
          <span className="flex min-w-0 items-center gap-2.5">
            <span className="flex size-6 shrink-0 items-center justify-center rounded-md border bg-raised">
              <IconCubeOutline18 className="size-3 text-gray-12" />
            </span>
            <span className="truncate text-sm font-medium">{name}</span>
          </span>
        ),
        apps: isLoading ? (
          <Skeleton className="h-3 w-12" />
        ) : (
          <span className="text-gray-11">{appsLabel}</span>
        ),
        deployed: isLoading ? <Skeleton className="h-3 w-20" /> : <LastDeployed at={deployedAt} />,
        actions: (
          <span className="relative z-10 flex justify-end">
            <ProjectActionsButton projectId={project.id} />
          </span>
        ),
      }}
    />
  );
}

function LastDeployed({ at }: { at: number | null }) {
  if (at === null) {
    return <span className="text-gray-9">Never</span>;
  }
  return <DeployedAgo value={at} className="truncate text-gray-11" />;
}
