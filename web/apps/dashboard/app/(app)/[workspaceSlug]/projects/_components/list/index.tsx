import { ProximityPrefetch } from "@/components/proximity-prefetch";
import { useWorkspaceNavigation } from "@/hooks/use-workspace-navigation";
import { collection } from "@/lib/collections";
import { routes } from "@/lib/navigation/routes";
import { useLiveQuery } from "@tanstack/react-db";
import { Dots, TriangleWarning2 } from "@unkey/icons";
import { Button, Empty } from "@unkey/ui";
import Link from "next/link";
import { useDeployGate } from "../hooks/use-deploy-gate";
import { ProjectActions } from "./project-actions";
import { ProjectCard } from "./project-card";
import { ProjectCardSkeleton } from "./project-card-skeleton";

// One row at the 3-column desktop width so loading doesn't tower over the
// real list before it resolves.
const MAX_SKELETON_COUNT = 3;

export const ProjectsList = () => {
  const workspace = useWorkspaceNavigation();
  const { gated } = useDeployGate();
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
    // No plan and no projects: the empty state is the paywall, so the primary
    // action is choosing a plan rather than a create form that dead-ends.
    if (gated) {
      return (
        <div className="w-full flex justify-center items-center h-full p-4">
          <Empty className="w-[400px] flex items-start">
            <Empty.Icon className="w-auto" />
            <Empty.Title>Compute plan required</Empty.Title>
            <Empty.Description className="text-left">
              You need a Compute plan before you can create projects.
            </Empty.Description>
            <Empty.Actions className="mt-4 justify-start">
              <Link href={routes.settings.billing({ workspaceSlug: workspace.slug })}>
                <Button size="md" variant="primary">
                  Choose a plan
                </Button>
              </Link>
            </Empty.Actions>
          </Empty>
        </div>
      );
    }

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
          <Link href={routes.settings.billing({ workspaceSlug: workspace.slug })}>
            <Button variant="outline" size="md">
              Choose a plan
            </Button>
          </Link>
        </div>
      ) : null}
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
