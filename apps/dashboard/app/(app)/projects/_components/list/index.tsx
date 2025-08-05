import { LoadMoreFooter } from "@/components/virtual-table/components/loading-indicator";
import { BookBookmark } from "@unkey/icons";
import { Button, Empty } from "@unkey/ui";
import { useProjectsListQuery } from "./hooks/use-projects-list-query";
import { ProjectCard } from "./projects-card";
import { ProjectCardSkeleton } from "./projects-card-skeleton";

const MAX_SKELETON_COUNT = 8;
const MINIMUM_DISPLAY_LIMIT = 10;

export const ProjectsList = () => {
  const { projects, isLoading, totalCount, hasMore, loadMore, isLoadingMore } =
    useProjectsListQuery();

  if (isLoading) {
    return (
      <div className="p-4">
        <div
          className="grid gap-4"
          style={{
            gridTemplateColumns: "repeat(auto-fit, minmax(350px, 1fr))",
          }}
        >
          {Array.from({ length: MAX_SKELETON_COUNT }).map((_, i) => (
            // biome-ignore lint/suspicious/noArrayIndexKey: <explanation>
            <ProjectCardSkeleton key={i} />
          ))}
        </div>
      </div>
    );
  }

  if (projects.length === 0) {
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
              href="https://docs.example.com/projects" // Replace with your actual docs URL
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
          {projects.map((project) => {
            const primaryHostname = project.hostnames[0]?.hostname || "No domain";
            return (
              <ProjectCard
                key={project.id}
                name={project.name}
                domain={primaryHostname}
                commitTitle="Latest deployment"
                commitDate={new Date(project.updatedAt || project.createdAt).toLocaleDateString()}
                branch={project.branch || "main"}
                author="Unknown"
                regions={["us-east-1", "us-west-2", "ap-east-1"]}
                repository={project.gitRepositoryUrl || undefined}
              />
            );
          })}
        </div>
      </div>
      {totalCount > MINIMUM_DISPLAY_LIMIT ? (
        <LoadMoreFooter
          onLoadMore={loadMore}
          isFetchingNextPage={isLoadingMore}
          totalVisible={projects.length}
          totalCount={totalCount}
          itemLabel="projects"
          buttonText="Load more projects"
          hasMore={hasMore}
          hide={!hasMore && projects.length === totalCount}
          countInfoText={
            <div className="flex gap-2">
              <span>Viewing</span>
              <span className="text-accent-12">{projects.length}</span>
              <span>of</span>
              <span className="text-grayA-12">{totalCount}</span>
              <span>projects</span>
            </div>
          }
        />
      ) : null}
    </>
  );
};
