import { ProximityPrefetch } from "@/components/proximity-prefetch";
import { collection } from "@/lib/collections";
import { useLiveQuery } from "@tanstack/react-db";
import { Dots } from "@unkey/icons";
import { Button, Empty } from "@unkey/ui";
import { ProjectActions } from "./project-actions";
import { ProjectCard } from "./project-card";
import { ProjectCardSkeleton } from "./project-card-skeleton";

const MAX_SKELETON_COUNT = 8;

export const ProjectsList = () => {
  const projects = useLiveQuery((q) => q.from({ project: collection.projects }));

  if (projects.isLoading) {
    return (
      <div className="grid gap-4 grid-cols-[repeat(auto-fit,minmax(325px,1fr))]">
        {Array.from({ length: MAX_SKELETON_COUNT }).map((_, i) => (
          // biome-ignore lint/suspicious/noArrayIndexKey: skeleton items don't need stable keys
          <ProjectCardSkeleton key={i} />
        ))}
      </div>
    );
  }

  if (projects.data.length === 0) {
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
    <div className="grid gap-4 grid-cols-[repeat(auto-fit,minmax(325px,1fr))]">
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
  );
};
