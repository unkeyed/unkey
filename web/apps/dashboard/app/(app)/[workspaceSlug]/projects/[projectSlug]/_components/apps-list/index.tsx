"use client";
import { useWorkspaceNavigation } from "@/hooks/use-workspace-navigation";
import { collection } from "@/lib/collections";
import { eq, useLiveQuery } from "@tanstack/react-db";
import { Dots, Plus } from "@unkey/icons";
import { Button, Empty } from "@unkey/ui";
import { notFound, useParams, useRouter } from "next/navigation";
import { ResourceCard } from "../../../_components/list/resource-card";
import { ProjectHomeNavigation } from "../project-home-navigation";
import { AppActions } from "./app-actions";

export const AppsList = () => {
  const params = useParams();
  const router = useRouter();
  const workspace = useWorkspaceNavigation();
  const projectSlug = typeof params?.projectSlug === "string" ? params.projectSlug : "";

  // This view sits above ProjectDataProvider, so it resolves the slug itself.
  const projectQuery = useLiveQuery(
    (q) =>
      projectSlug
        ? q
            .from({ project: collection.projects })
            .where(({ project }) => eq(project.slug, projectSlug))
        : undefined,
    [projectSlug],
  );
  const project = projectQuery.data?.at(0);
  const projectId = project?.id;
  const projectName = project?.name;

  const openCreateApp = () =>
    router.push(`/${workspace.slug}/projects/new?projectId=${projectId ?? ""}`);

  const apps = useLiveQuery(
    (q) =>
      projectId
        ? q.from({ app: collection.apps }).where(({ app }) => eq(app.projectId, projectId))
        : undefined,
    [projectId],
  );

  // isReady is true for a disabled (slug-less) query too, so require the slug.
  if (projectSlug && projectQuery.isReady && !project) {
    notFound();
  }

  const appRows = apps.data ?? [];

  return (
    <div className="flex flex-col h-full">
      <ProjectHomeNavigation
        projectSlug={projectSlug}
        projectName={projectName}
        onCreateApp={openCreateApp}
      />
      <div className="p-4 flex flex-col gap-4">
        {apps.isLoading || !projectId ? null : appRows.length === 0 ? (
          <div className="w-full flex justify-center items-center p-4">
            <Empty className="w-[400px] flex items-start">
              <Empty.Icon className="w-auto" />
              <Empty.Title>No Apps Found</Empty.Title>
              <Empty.Description className="text-left">
                This project has no apps yet. Create an app to start deploying.
              </Empty.Description>
              <Empty.Actions className="mt-4 justify-start">
                <Button size="md" onClick={openCreateApp}>
                  <Plus />
                  Create app
                </Button>
              </Empty.Actions>
            </Empty>
          </div>
        ) : (
          <div className="grid gap-4 grid-cols-[repeat(auto-fit,minmax(325px,370px))]">
            {appRows.map((app) => (
              <ResourceCard
                key={app.id}
                href={`/${workspace.slug}/projects/${projectSlug}/apps/${app.slug}/deployments`}
                name={app.name}
                domain={app.domain}
                commitTitle={app.commitTitle}
                commitTimestamp={app.commitTimestamp}
                branch={app.branch}
                author={app.author}
                authorAvatar={app.authorAvatar}
                repository={
                  app.repositoryFullName
                    ? `https://github.com/${app.repositoryFullName}`
                    : undefined
                }
                actions={
                  <AppActions projectSlug={projectSlug} appSlug={app.slug} appId={app.id}>
                    <Button
                      variant="ghost"
                      size="icon"
                      className="mb-auto shrink-0"
                      title="App actions"
                    >
                      <Dots iconSize="sm-regular" />
                    </Button>
                  </AppActions>
                }
              />
            ))}
          </div>
        )}
      </div>
    </div>
  );
};
