import { useProjectsListQuery } from "./hooks/use-projects-list-query";
import { ProjectCard } from "./projects-card";

export const ProjectsList = () => {
  const { projects, isLoading, totalCount } = useProjectsListQuery();

  if (isLoading) {
    return (
      <div className="p-4 flex justify-center">
        <div className="text-sm text-gray-11">Loading projects...</div>
      </div>
    );
  }

  if (projects.length === 0) {
    return (
      <div className="p-4 flex justify-center">
        <div className="text-sm text-gray-11">No projects found</div>
      </div>
    );
  }

  console.log({ projects });
  return (
    <div className="p-4">
      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 gap-4">
        {projects.map((project) => {
          const primaryHostname = project.hostnames[0]?.hostname || "No domain";

          return (
            <ProjectCard
              key={project.id}
              name={project.name}
              domain={primaryHostname}
              commitTitle="Latest deployment" // You don't have commit data, so use placeholder
              commitDate={new Date(
                project.updatedAt || project.createdAt
              ).toLocaleDateString()}
              branch={project.branch || "main"}
              author="Unknown" // You don't have author data
              regions={["us-east-1", "us-west-2", "ap-east-1"]} // You don't have regions data
              repository={project.gitRepositoryUrl || undefined}
            />
          );
        })}
      </div>

      {/*{hasMore && (
        <div className="p-4 flex justify-center">
          <button
            onClick={() => loadMore()}
            disabled={isLoadingMore}
            className="px-4 py-2 bg-accent-9 text-accent-12 rounded-lg hover:bg-accent-10 disabled:opacity-50"
          >
            {isLoadingMore ? "Loading..." : "Load More"}
          </button>
        </div>
      )}
*/}
      <div className="p-4 text-center text-sm text-gray-11">
        Showing {projects.length} of {totalCount} projects
      </div>
    </div>
  );
};
