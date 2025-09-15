import { collection } from "@/lib/collections";
import { useLiveQuery } from "@tanstack/react-db";
import { BookBookmark, Dots } from "@unkey/icons";
import { Button, Empty } from "@unkey/ui";
import { ProjectActions } from "./project-actions";
import { ProjectCard } from "./projects-card";
import { ProjectCardSkeleton } from "./projects-card-skeleton";

const MAX_SKELETON_COUNT = 8;

export const ProjectsList = () => {
  const projects = useLiveQuery((q) =>
    q.from({ project: collection.projects }).orderBy(({ project }) => project.updatedAt, "desc"),
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
            There are no projects configured yet. Create your first project to start deploying and
            managing your applications.
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
    <>
      <div className="p-4">
        <div
          className="grid gap-4"
          style={{
            gridTemplateColumns: "repeat(auto-fit, minmax(325px, 350px))",
          }}
        >
          {projects.data.map((project) => {
            return (
              <ProjectCard
                projectId={project.id}
                key={project.id}
                name={project.name}
                domain="TODO"
                commitTitle="Latest deployment"
                commitDate="TODO"
                branch="TODO"
                author="TODO"
                regions={["us-east-1", "us-west-2", "ap-east-1"]}
                repository={project.gitRepositoryUrl || undefined}
                actions={
                  <ProjectActions projectId={project.id}>
                    <Button
                      variant="ghost"
                      size="icon"
                      className="mb-auto shrink-0"
                      title="Project actions"
                    >
                      <Dots size="sm-regular" />
                    </Button>
                  </ProjectActions>
                }
              />
            );
          })}
        </div>
      </div>
    </>
  );
};
