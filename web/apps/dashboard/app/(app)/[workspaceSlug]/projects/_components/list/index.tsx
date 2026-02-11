import { ProximityPrefetch } from "@/components/proximity-prefetch";
import { collection } from "@/lib/collections";
import { ilike, useLiveQuery } from "@tanstack/react-db";
import { BookBookmark, Dots } from "@unkey/icons";
import { Button, Empty } from "@unkey/ui";
import { useProjectsFilters } from "../hooks/use-projects-filters";
import { ProjectActions } from "./project-actions";
import { ProjectCard } from "./projects-card";
import { ProjectCardSkeleton } from "./projects-card-skeleton";

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
        <div
          className="grid gap-4"
          style={{
            gridTemplateColumns: "repeat(auto-fit, minmax(325px, 350px))",
          }}
        >
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
            {projectName
              ? `No projects found matching "${projectName}". Try a different search term.`
              : "There are no projects configured yet. Create your first project to start deploying and managing your applications."}
          </Empty.Description>
          <Empty.Actions className="mt-4 justify-start">
            <a
              href="https://www.unkey.com/docs/introduction"
              target="_blank"
              rel="noopener noreferrer"
            >
              <Button size="md">
                <BookBookmark />
                Learn about Deploy
              </Button>
            </a>
          </Empty.Actions>
        </Empty>
      </div>
    );
  }

  return (
    <div className="p-4">
      <div
        className="grid gap-4"
        style={{
          gridTemplateColumns: "repeat(auto-fit, minmax(325px, 370px))",
        }}
      >
        {projects.data.map((project) => (
          <ProximityPrefetch
            distance={300}
            debounceDelay={150}
            key={project.id}
            onEnterProximity={() => {
              // Preloading is now handled automatically by query-driven sync
            }}
          >
            <ProjectCard
              projectId={project.id}
              name={project.name}
              domain={project.domain}
              commitTitle={project.commitTitle}
              commitTimestamp={project.commitTimestamp}
              branch={project.branch}
              author={project.author}
              authorAvatar={project.authorAvatar}
              regions={project.regions}
              actions={
                <ProjectActions projectId={project.id} projectName={project.name}>
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
