"use client";
import { useWorkspaceNavigation } from "@/hooks/use-workspace-navigation";
import { collection } from "@/lib/collections";
import { eq, useLiveQuery } from "@tanstack/react-db";
import { Dots, Plus } from "@unkey/icons";
import { Button, Empty } from "@unkey/ui";
import { useParams, useRouter } from "next/navigation";
import { ResourceCard } from "../../../_components/list/resource-card";
import { ProjectHomeNavigation } from "../project-home-navigation";
import { AppActions } from "./app-actions";

export const AppsList = () => {
  const params = useParams();
  const router = useRouter();
  const workspace = useWorkspaceNavigation();
  const projectId = typeof params?.projectId === "string" ? params.projectId : "";
  const openCreateApp = () => router.push(`/${workspace.slug}/projects/new?projectId=${projectId}`);

  const apps = useLiveQuery(
    (q) => q.from({ app: collection.apps }).where(({ app }) => eq(app.projectId, projectId)),
    [projectId],
  );

  const projectQuery = useLiveQuery(
    (q) =>
      q.from({ project: collection.projects }).where(({ project }) => eq(project.id, projectId)),
    [projectId],
  );
  const projectName = projectQuery.data?.at(0)?.name;

  return (
    <div className="flex flex-col h-full">
      <ProjectHomeNavigation
        projectId={projectId}
        projectName={projectName}
        onCreateApp={openCreateApp}
      />
      <div className="p-4 flex flex-col gap-4">
        {apps.isLoading ? null : apps.data.length === 0 ? (
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
            {apps.data.map((app) => (
              <ResourceCard
                key={app.id}
                href={`/${workspace.slug}/projects/${projectId}/apps/${app.id}/deployments`}
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
                  <AppActions projectId={projectId} appId={app.id}>
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
