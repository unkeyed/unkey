import { collection, collectionManager } from "@/lib/collections";
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
        .orderBy(({ project }) => project.updatedAt, "desc")
        .where(({ project }) => ilike(project.name, `%${projectName}%`)),
    [projectName],
  );

  // Get deployments and domains for each project
  const deploymentQueries = projects.data.map((project) => {
    const collections = collectionManager.getProjectCollections(project.id);
    return useLiveQuery((q) => q.from({ deployment: collections.deployments }), [project.id]);
  });

  const domainQueries = projects.data.map((project) => {
    const collections = collectionManager.getProjectCollections(project.id);
    return useLiveQuery((q) => q.from({ domain: collections.domains }), [project.id]);
  });

  // Flatten the results
  const allDeployments = deploymentQueries.flatMap((query) => query.data || []);
  const allDomains = domainQueries.flatMap((query) => query.data || []);

  const isLoading =
    projects.isLoading ||
    deploymentQueries.some((q) => q.isLoading) ||
    domainQueries.some((q) => q.isLoading);

  if (isLoading) {
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
            gridTemplateColumns: "repeat(auto-fit, minmax(325px, 370px))",
          }}
        >
          {projects.data.map((project) => {
            // Find active deployment and associated domain for this project
            const activeDeployment = project.activeDeploymentId
              ? allDeployments.find((d) => d.id === project.activeDeploymentId)
              : null;

            // Find domain for this project
            const projectDomain = allDomains.find((d) => d.projectId === project.id);

            // Extract deployment regions for display
            const regions = activeDeployment?.runtimeConfig?.regions?.map((r) => r.region) ?? [];

            return (
              <ProjectCard
                projectId={project.id}
                key={project.id}
                name={project.name}
                domain={projectDomain?.domain ?? "No domain configured"}
                commitTitle={activeDeployment?.gitCommitMessage ?? "No deployments"}
                commitTimestamp={activeDeployment?.gitCommitTimestamp}
                branch={activeDeployment?.gitBranch ?? "—"}
                author={activeDeployment?.gitCommitAuthorName ?? "—"}
                regions={regions.length > 0 ? regions : ["No deployments"]}
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
