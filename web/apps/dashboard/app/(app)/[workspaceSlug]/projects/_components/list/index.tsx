import { ProximityPrefetch } from "@/components/proximity-prefetch";
import { useWorkspaceNavigation } from "@/hooks/use-workspace-navigation";
import { collection } from "@/lib/collections";
import { ilike, useLiveQuery } from "@tanstack/react-db";
import { Dots, TriangleWarning2 } from "@unkey/icons";
import { Button, Empty } from "@unkey/ui";
import Link from "next/link";
import { useDeployGate } from "../hooks/use-deploy-gate";
import { useProjectsFilters } from "../hooks/use-projects-filters";
import { ProjectActions } from "./project-actions";
import { ResourceCard } from "./resource-card";
import { ResourceCardSkeleton } from "./resource-card-skeleton";

const MAX_SKELETON_COUNT = 8;

export const ProjectsList = () => {
  const workspace = useWorkspaceNavigation();
  const { gated } = useDeployGate();
  const { filters } = useProjectsFilters();
  const projectName = filters.find((f) => f.field === "query")?.value ?? "";
  const billingHref = `/${workspace.slug}/settings/billing`;

  const projects = useLiveQuery(
    (q) =>
      q
        .from({ project: collection.projects })
        .where(({ project }) => ilike(project.name, `%${projectName}%`)),
    [projectName],
  );

  if (projects.isLoading) {
    return (
      <div className="p-4">
        <div className="grid gap-4 grid-cols-[repeat(auto-fit,minmax(325px,370px))]">
          {Array.from({ length: MAX_SKELETON_COUNT }).map((_, i) => (
            // biome-ignore lint/suspicious/noArrayIndexKey: skeleton items don't need stable keys
            <ResourceCardSkeleton key={i} />
          ))}
        </div>
      </div>
    );
  }

  if (projects.data.length === 0) {
    // First run with no Deploy plan: the empty state is the paywall, so the
    // primary action is choosing a plan rather than a create form that dead-ends.
    if (gated && !projectName) {
      return (
        <div className="w-full flex justify-center items-center h-full p-4">
          <Empty className="w-[400px] flex items-start">
            <Empty.Icon className="w-auto" />
            <Empty.Title>Compute plan required</Empty.Title>
            <Empty.Description className="text-left">
              You need a Compute plan before you can create projects.
            </Empty.Description>
            <Empty.Actions className="mt-4 justify-start">
              <Link href={billingHref}>
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
      <div className="w-full flex justify-center items-center h-full p-4">
        <Empty className="w-[400px] flex items-start">
          <Empty.Icon className="w-auto" />
          <Empty.Title>No Projects Found</Empty.Title>
          <Empty.Description className="text-left">
            {`No projects found matching "${projectName}". Try a different search term.`}
          </Empty.Description>
        </Empty>
      </div>
    );
  }

  return (
    <div className="p-4">
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
          <Link href={billingHref}>
            <Button variant="outline" size="md">
              Choose a plan
            </Button>
          </Link>
        </div>
      ) : null}
      <div className="grid gap-4 grid-cols-[repeat(auto-fit,minmax(325px,370px))]">
        {projects.data.map((project) => (
          <ProximityPrefetch distance={300} debounceDelay={150} key={project.id}>
            <ResourceCard
              href={`/${workspace.slug}/projects/${project.id}`}
              name={project.name}
              domain={project.domain}
              commitTitle={project.commitTitle}
              commitTimestamp={project.commitTimestamp}
              branch={project.branch}
              author={project.author}
              authorAvatar={project.authorAvatar}
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
    </div>
  );
};
