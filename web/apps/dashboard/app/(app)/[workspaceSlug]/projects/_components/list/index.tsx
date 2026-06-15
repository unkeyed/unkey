import { ProximityPrefetch } from "@/components/proximity-prefetch";
import { collection } from "@/lib/collections";
import { ilike, useLiveQuery } from "@tanstack/react-db";
import { Dots } from "@unkey/icons";
import { Button, Empty } from "@unkey/ui";
import { useProjectsFilters } from "../hooks/use-projects-filters";
import { ProjectActions } from "./project-actions";
import { ProjectCard } from "./project-card";
import { ProjectCardSkeleton } from "./project-card-skeleton";

const MAX_SKELETON_COUNT = 8;

export const ProjectsList = () => {
  const { filters } = useProjectsFilters();
  const projectName = filters.find((f) => f.field === "query")?.value ?? "";

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
            <ProjectCardSkeleton key={i} />
          ))}
        </div>
      </div>
    );
  }

  if (projects.data.length === 0) {
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
      <div className="grid gap-4 grid-cols-[repeat(auto-fit,minmax(325px,370px))]">
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
    </div>
  );
};
