"use client";
import { ResourceCard } from "@/app/(app)/[workspaceSlug]/projects/_components/list/resource-card";
import { ResourceCardSkeleton } from "@/app/(app)/[workspaceSlug]/projects/_components/list/resource-card-skeleton";
import { useAppHomeHref } from "@/hooks/use-app-home-href";
import { useWorkspaceNavigation } from "@/hooks/use-workspace-navigation";
import { collection } from "@/lib/collections";
import { githubUrl } from "@/lib/github-url";
import { routes } from "@/lib/navigation/routes";
import { eq, useLiveQuery } from "@tanstack/react-db";
import { Dots, Github, Plus, Terminal } from "@unkey/icons";
import { Button, Empty } from "@unkey/ui";
import { useParams, useRouter } from "next/navigation";
import { AppActions } from "./app-actions";

// One row at the 3-column desktop width so loading doesn't tower over the
// real list before it resolves.
const MAX_SKELETON_COUNT = 3;

export const AppsList = () => {
  const params = useParams();
  const router = useRouter();
  const workspace = useWorkspaceNavigation();
  const appHomeHref = useAppHomeHref();
  const projectId = typeof params?.projectId === "string" ? params.projectId : "";
  const openCreateApp = () =>
    router.push(routes.projects.apps.new({ workspaceSlug: workspace.slug, projectId }));

  const apps = useLiveQuery(
    (q) => q.from({ app: collection.apps }).where(({ app }) => eq(app.projectId, projectId)),
    [projectId],
  );

  return (
    <>
      {apps.isLoading ? (
        <div className="grid gap-4 grid-cols-1 md:grid-cols-2 xl:grid-cols-3">
          {Array.from({ length: MAX_SKELETON_COUNT }).map((_, i) => (
            // biome-ignore lint/suspicious/noArrayIndexKey: skeleton items don't need stable keys
            <ResourceCardSkeleton key={i} />
          ))}
        </div>
      ) : apps.data.length === 0 ? (
        <div className="flex-1 flex justify-center items-center px-4 py-16 border border-grayA-4 rounded-[14px] overflow-hidden">
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
        <div className="grid gap-4 grid-cols-1 md:grid-cols-2 xl:grid-cols-3">
          {apps.data.map((app) => (
            <ResourceCard
              key={app.id}
              href={appHomeHref({
                workspaceSlug: workspace.slug,
                projectId,
                appId: app.id,
              })}
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
              sourceUrl={githubUrl.deployment({
                repoFullName: app.repositoryFullName,
                forkRepoFullName: app.forkRepositoryFullName,
                prNumber: app.prNumber,
                sha: app.commitSha,
              })}
              commitTimestamp={app.commitTimestamp}
              branch={app.branch}
              author={app.author}
              authorAvatar={app.authorAvatar}
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
    </>
  );
};
