"use client";
import { ResourceCard } from "@/app/(app)/[workspaceSlug]/projects/_components/list/resource-card";
import { ResourceCardSkeleton } from "@/app/(app)/[workspaceSlug]/projects/_components/list/resource-card-skeleton";
import { useProjectSlug } from "@/hooks/use-route-slugs";
import { useWorkspaceNavigation } from "@/hooks/use-workspace-navigation";
import { collection } from "@/lib/collections";
import { eq, useLiveQuery } from "@tanstack/react-db";
import { Dots, Github, Plus, Terminal } from "@unkey/icons";
import { Button, Empty } from "@unkey/ui";
import { useRouter } from "next/navigation";
import { ProjectHomeNavigation } from "../project-home-navigation";
import { AppActions } from "./app-actions";

const MAX_SKELETON_COUNT = 4;

export const AppsList = () => {
  const workspace = useWorkspaceNavigation();
  const projectSlug = useProjectSlug();
  const basePath = `/${workspace.slug}/projects/${projectSlug}`;

  return (
    <div className="flex flex-col h-full">
      <ProjectHomeNavigation projectSlug={projectSlug ?? ""} />
      <div className="p-4 flex flex-col gap-4">
        {projectSlug ? <AppsGrid projectSlug={projectSlug} basePath={basePath} /> : null}
      </div>
    </div>
  );
};

const AppsGrid = ({ projectSlug, basePath }: { projectSlug: string; basePath: string }) => {
  const router = useRouter();
  const openCreateApp = () => router.push(`${basePath}/apps/new`);

  const apps = useLiveQuery(
    (q) => q.from({ app: collection.apps }).where(({ app }) => eq(app.projectSlug, projectSlug)),
    [projectSlug],
  );

  if (apps.isLoading) {
    return (
      <div className="grid gap-4 grid-cols-[repeat(auto-fit,minmax(325px,370px))]">
        {Array.from({ length: MAX_SKELETON_COUNT }).map((_, i) => (
          // biome-ignore lint/suspicious/noArrayIndexKey: skeleton items don't need stable keys
          <ResourceCardSkeleton key={i} />
        ))}
      </div>
    );
  }

  if (apps.data.length === 0) {
    return (
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
              Create new app
            </Button>
          </Empty.Actions>
        </Empty>
      </div>
    );
  }

  return (
    <div className="grid gap-4 grid-cols-[repeat(auto-fit,minmax(325px,370px))]">
      {apps.data.map((app) => (
        <ResourceCard
          key={app.id}
          href={`${basePath}/apps/${app.slug}/deployments`}
          icon={
            app.repositoryFullName ? (
              <Github iconSize="xl-medium" className="shrink-0 size-5" />
            ) : (
              <Terminal iconSize="xl-medium" className="shrink-0 size-5" />
            )
          }
          name={app.name}
          domain={app.domain}
          commitTitle={app.commitTitle}
          commitTimestamp={app.commitTimestamp}
          branch={app.branch}
          author={app.author}
          authorAvatar={app.authorAvatar}
          actions={
            <AppActions basePath={basePath} appSlug={app.slug} appId={app.id}>
              <Button variant="ghost" size="icon" className="mb-auto shrink-0" title="App actions">
                <Dots iconSize="sm-regular" />
              </Button>
            </AppActions>
          }
        />
      ))}
    </div>
  );
};
